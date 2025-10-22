// Package network provides concrete implementations of network diagnostic operations
package network

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Client implements the NetworkClient interface with real network operations
type Client struct {
	config       *domain.NetworkConfig
	errorHandler domain.ErrorHandler
	logger       domain.Logger
	retryManager *RetryManager
}

// NewClient creates a new network client with the provided configuration
func NewClient(config *domain.NetworkConfig, errorHandler domain.ErrorHandler, logger domain.Logger) *Client {
	return &Client{
		config:       config,
		errorHandler: errorHandler,
		logger:       logger,
		retryManager: NewRetryManager(config.RetryAttempts, config.RetryDelay),
	}
}

// Ping performs ping operations to the specified host
func (c *Client) Ping(ctx context.Context, host string, opts domain.PingOptions) (<-chan domain.PingResult, error) {
	if err := c.validateHost(host); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "invalid host for ping operation",
			Cause:     err,
			Context:   map[string]interface{}{"host": host},
			Timestamp: time.Now(),
			Code:      "PING_INVALID_HOST",
		}
	}

	resultChan := make(chan domain.PingResult, opts.Count)
	
	go func() {
		defer close(resultChan)
		c.executePing(ctx, host, opts, resultChan)
	}()

	return resultChan, nil
}

// Traceroute performs traceroute operations to the specified host
func (c *Client) Traceroute(ctx context.Context, host string, opts domain.TraceOptions) (<-chan domain.TraceHop, error) {
	if err := c.validateHost(host); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "invalid host for traceroute operation",
			Cause:     err,
			Context:   map[string]interface{}{"host": host},
			Timestamp: time.Now(),
			Code:      "TRACE_INVALID_HOST",
		}
	}

	resultChan := make(chan domain.TraceHop, opts.MaxHops)
	
	go func() {
		defer close(resultChan)
		c.executeTraceroute(ctx, host, opts, resultChan)
	}()

	return resultChan, nil
}

// DNSLookup performs DNS lookups for the specified domain and record type
func (c *Client) DNSLookup(ctx context.Context, domainName string, recordType domain.DNSRecordType) (domain.DNSResult, error) {
	if err := c.validateDomain(domainName); err != nil {
		return domain.DNSResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "invalid domain for DNS lookup",
			Cause:     err,
			Context:   map[string]interface{}{"domain": domainName, "record_type": recordType},
			Timestamp: time.Now(),
			Code:      "DNS_INVALID_DOMAIN",
		}
	}

	result, err := c.retryManager.ExecuteWithRetry(ctx, func() (interface{}, error) {
		return c.executeDNSLookup(ctx, domainName, recordType)
	}, func(err error) bool {
		return c.isRetryableNetworkError(err)
	})
	
	if err != nil {
		return domain.DNSResult{}, err
	}
	
	return result.(domain.DNSResult), nil
}

// WHOISLookup performs WHOIS lookups for the specified query
func (c *Client) WHOISLookup(ctx context.Context, query string) (domain.WHOISResult, error) {
	if err := c.validateQuery(query); err != nil {
		return domain.WHOISResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "invalid query for WHOIS lookup",
			Cause:     err,
			Context:   map[string]interface{}{"query": query},
			Timestamp: time.Now(),
			Code:      "WHOIS_INVALID_QUERY",
		}
	}

	result, err := c.retryManager.ExecuteWithRetry(ctx, func() (interface{}, error) {
		return c.executeWHOISLookup(ctx, query)
	}, func(err error) bool {
		return c.isRetryableNetworkError(err)
	})
	
	if err != nil {
		return domain.WHOISResult{}, err
	}
	
	return result.(domain.WHOISResult), nil
}

// SSLCheck performs SSL certificate checks for the specified host and port
func (c *Client) SSLCheck(ctx context.Context, host string, port int) (domain.SSLResult, error) {
	if err := c.validateHost(host); err != nil {
		return domain.SSLResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "invalid host for SSL check",
			Cause:     err,
			Context:   map[string]interface{}{"host": host, "port": port},
			Timestamp: time.Now(),
			Code:      "SSL_INVALID_HOST",
		}
	}

	if port <= 0 || port > 65535 {
		return domain.SSLResult{}, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "invalid port for SSL check",
			Context:   map[string]interface{}{"host": host, "port": port},
			Timestamp: time.Now(),
			Code:      "SSL_INVALID_PORT",
		}
	}

	result, err := c.retryManager.ExecuteWithRetry(ctx, func() (interface{}, error) {
		return c.executeSSLCheck(ctx, host, port)
	}, func(err error) bool {
		return c.isRetryableNetworkError(err)
	})
	
	if err != nil {
		return domain.SSLResult{}, err
	}
	
	return result.(domain.SSLResult), nil
}

// validateHost validates that the host is a valid hostname or IP address
func (c *Client) validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Try to parse as IP address first
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	// Validate as hostname
	if len(host) > 253 {
		return fmt.Errorf("hostname too long")
	}

	return nil
}

// validateDomain validates that the domain is valid for DNS lookup
func (c *Client) validateDomain(domainName string) error {
	if domainName == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if len(domainName) > 253 {
		return fmt.Errorf("domain name too long")
	}

	return nil
}

// validateQuery validates that the query is valid for WHOIS lookup
func (c *Client) validateQuery(query string) error {
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	return nil
}

// isRetryableNetworkError determines if an error is retryable
func (c *Client) isRetryableNetworkError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}