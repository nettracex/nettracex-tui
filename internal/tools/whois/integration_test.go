// +build integration

// Package whois provides integration tests for WHOIS diagnostic functionality
package whois

import (
	"context"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogger implements domain.Logger for integration tests
type TestLogger struct{}

func (l *TestLogger) Debug(msg string, fields ...interface{}) {}
func (l *TestLogger) Info(msg string, fields ...interface{})  {}
func (l *TestLogger) Warn(msg string, fields ...interface{})  {}
func (l *TestLogger) Error(msg string, fields ...interface{}) {}
func (l *TestLogger) Fatal(msg string, fields ...interface{}) {}

// TestErrorHandler implements domain.ErrorHandler for integration tests
type TestErrorHandler struct{}

func (h *TestErrorHandler) Handle(err error) error                                                { return err }
func (h *TestErrorHandler) HandleWithContext(err error, ctx map[string]interface{}) error       { return err }
func (h *TestErrorHandler) CanRecover(err error) bool                                            { return false }
func (h *TestErrorHandler) Recover(err error) error                                              { return err }

func TestWHOISIntegration_RealDomains(t *testing.T) {
	// Skip if running in CI or if network access is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}
	
	logger := &TestLogger{}
	errorHandler := &TestErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	
	// Create WHOIS tool
	tool := NewTool(client, logger)

	tests := []struct {
		name     string
		domain   string
		validate func(t *testing.T, result domain.WHOISResult)
	}{
		{
			name:   "google.com",
			domain: "google.com",
			validate: func(t *testing.T, result domain.WHOISResult) {
				assert.NotEmpty(t, result.Domain)
				assert.NotEmpty(t, result.RawData)
				assert.NotEmpty(t, result.Registrar)
				assert.False(t, result.Created.IsZero())
				assert.False(t, result.Expires.IsZero())
				assert.NotEmpty(t, result.NameServers)
			},
		},
		{
			name:   "github.com",
			domain: "github.com",
			validate: func(t *testing.T, result domain.WHOISResult) {
				assert.NotEmpty(t, result.Domain)
				assert.NotEmpty(t, result.RawData)
				assert.NotEmpty(t, result.Registrar)
				assert.False(t, result.Created.IsZero())
				assert.False(t, result.Expires.IsZero())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parameters
			params := domain.NewWHOISParameters(tt.domain)

			// Execute WHOIS lookup with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := tool.Execute(ctx, params)
			require.NoError(t, err, "WHOIS lookup should succeed for %s", tt.domain)
			require.NotNil(t, result, "Result should not be nil")

			// Extract WHOIS result
			whoisResult, ok := result.Data().(domain.WHOISResult)
			require.True(t, ok, "Result should be WHOISResult type")

			// Validate result
			err = ValidateWHOISResult(whoisResult)
			assert.NoError(t, err, "WHOIS result should be valid")

			// Run custom validation
			tt.validate(t, whoisResult)

			// Check metadata
			metadata := result.Metadata()
			assert.Equal(t, "whois", metadata["tool"])
			assert.Equal(t, tt.domain, metadata["query"])
			assert.Equal(t, "domain", metadata["query_type"])
		})
	}
}

func TestWHOISIntegration_IPAddresses(t *testing.T) {
	// Skip if running in CI or if network access is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}
	
	logger := &TestLogger{}
	errorHandler := &TestErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	
	// Create WHOIS tool
	tool := NewTool(client, logger)

	tests := []struct {
		name string
		ip   string
	}{
		{
			name: "Google DNS",
			ip:   "8.8.8.8",
		},
		{
			name: "Cloudflare DNS",
			ip:   "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parameters
			params := domain.NewWHOISParameters(tt.ip)

			// Execute WHOIS lookup with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := tool.Execute(ctx, params)
			require.NoError(t, err, "WHOIS lookup should succeed for %s", tt.ip)
			require.NotNil(t, result, "Result should not be nil")

			// Extract WHOIS result
			whoisResult, ok := result.Data().(domain.WHOISResult)
			require.True(t, ok, "Result should be WHOISResult type")

			// Basic validation for IP WHOIS
			assert.NotEmpty(t, whoisResult.RawData, "Raw data should not be empty")

			// Check metadata
			metadata := result.Metadata()
			assert.Equal(t, "whois", metadata["tool"])
			assert.Equal(t, tt.ip, metadata["query"])
			assert.Equal(t, "ip", metadata["query_type"])
		})
	}
}

func TestWHOISIntegration_ErrorCases(t *testing.T) {
	// Skip if running in CI or if network access is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second, // Short timeout for error testing
		RetryAttempts: 1,
		RetryDelay:    1 * time.Second,
	}
	
	logger := &TestLogger{}
	errorHandler := &TestErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	
	// Create WHOIS tool
	tool := NewTool(client, logger)

	tests := []struct {
		name        string
		query       string
		expectError bool
	}{
		{
			name:        "non-existent domain",
			query:       "this-domain-definitely-does-not-exist-12345.com",
			expectError: false, // WHOIS servers typically return "No match" rather than error
		},
		{
			name:        "invalid TLD",
			query:       "example.invalidtld",
			expectError: true, // Should fail to find WHOIS server or get connection error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parameters
			params := domain.NewWHOISParameters(tt.query)

			// Execute WHOIS lookup with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, err := tool.Execute(ctx, params)

			if tt.expectError {
				assert.Error(t, err, "Should get error for %s", tt.query)
			} else {
				// Even for non-existent domains, we should get a result with raw data
				if err == nil {
					require.NotNil(t, result, "Result should not be nil")
					whoisResult, ok := result.Data().(domain.WHOISResult)
					require.True(t, ok, "Result should be WHOISResult type")
					assert.NotEmpty(t, whoisResult.RawData, "Should have raw WHOIS data even for non-existent domains")
				}
			}
		})
	}
}

func TestWHOISIntegration_Timeout(t *testing.T) {
	// Skip if running in CI or if network access is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create network client with very short timeout
	config := &domain.NetworkConfig{
		Timeout:       1 * time.Millisecond, // Very short timeout to force timeout
		RetryAttempts: 1,
		RetryDelay:    1 * time.Millisecond,
	}
	
	logger := &TestLogger{}
	errorHandler := &TestErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	
	// Create WHOIS tool
	tool := NewTool(client, logger)

	// Create parameters
	params := domain.NewWHOISParameters("google.com")

	// Execute WHOIS lookup with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := tool.Execute(ctx, params)

	// Should get timeout error
	assert.Error(t, err, "Should get timeout error")
	assert.Nil(t, result, "Result should be nil on timeout")

	// Check that it's a network error
	if netErr, ok := err.(*domain.NetTraceError); ok {
		assert.Equal(t, domain.ErrorTypeNetwork, netErr.Type)
	}
}

func TestWHOISIntegration_ConcurrentRequests(t *testing.T) {
	// Skip if running in CI or if network access is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}
	
	logger := &TestLogger{}
	errorHandler := &TestErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	
	// Create WHOIS tool
	tool := NewTool(client, logger)

	domains := []string{"google.com", "github.com", "stackoverflow.com"}
	results := make(chan error, len(domains))

	// Launch concurrent WHOIS lookups
	for _, domain := range domains {
		go func(d string) {
			params := domain.NewWHOISParameters(d)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := tool.Execute(ctx, params)
			if err != nil {
				results <- err
				return
			}

			if result == nil {
				results <- assert.AnError
				return
			}

			whoisResult, ok := result.Data().(domain.WHOISResult)
			if !ok {
				results <- assert.AnError
				return
			}

			if err := ValidateWHOISResult(whoisResult); err != nil {
				results <- err
				return
			}

			results <- nil
		}(domain)
	}

	// Wait for all results
	for i := 0; i < len(domains); i++ {
		select {
		case err := <-results:
			assert.NoError(t, err, "Concurrent WHOIS lookup should succeed")
		case <-time.After(60 * time.Second):
			t.Fatal("Timeout waiting for concurrent WHOIS results")
		}
	}
}