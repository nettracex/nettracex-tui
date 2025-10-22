// Package domain contains parameter implementations
package domain

import (
	"fmt"
)

// BaseParameters provides a basic implementation of the Parameters interface
type BaseParameters struct {
	data map[string]interface{}
}

// NewParameters creates a new BaseParameters instance
func NewParameters() *BaseParameters {
	return &BaseParameters{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a parameter value by key
func (p *BaseParameters) Get(key string) interface{} {
	return p.data[key]
}

// Set sets a parameter value by key
func (p *BaseParameters) Set(key string, value interface{}) {
	p.data[key] = value
}

// Validate validates all parameters
func (p *BaseParameters) Validate() error {
	// Basic validation - can be extended by specific parameter types
	for key, value := range p.data {
		if value == nil {
			continue
		}
		
		// Check for empty strings
		if str, ok := value.(string); ok && str == "" {
			return fmt.Errorf("parameter '%s' cannot be empty", key)
		}
	}
	return nil
}

// ToMap returns all parameters as a map
func (p *BaseParameters) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range p.data {
		result[k] = v
	}
	return result
}

// PingParameters represents parameters for ping operations
type PingParameters struct {
	*BaseParameters
}

// NewPingParameters creates new ping parameters
func NewPingParameters(host string, options PingOptions) *PingParameters {
	params := &PingParameters{
		BaseParameters: NewParameters(),
	}
	params.Set("host", host)
	params.Set("count", options.Count)
	params.Set("interval", options.Interval)
	params.Set("timeout", options.Timeout)
	params.Set("packet_size", options.PacketSize)
	params.Set("ttl", options.TTL)
	params.Set("ipv6", options.IPv6)
	return params
}

// Validate validates ping parameters
func (p *PingParameters) Validate() error {
	host := p.Get("host")
	if host == nil || host.(string) == "" {
		return fmt.Errorf("host parameter is required")
	}
	
	count := p.Get("count")
	if count != nil && count.(int) <= 0 {
		return fmt.Errorf("count must be positive")
	}
	
	packetSize := p.Get("packet_size")
	if packetSize != nil && (packetSize.(int) <= 0 || packetSize.(int) > 65507) {
		return fmt.Errorf("packet_size must be between 1 and 65507")
	}
	
	return nil
}

// TracerouteParameters represents parameters for traceroute operations
type TracerouteParameters struct {
	*BaseParameters
}

// NewTracerouteParameters creates new traceroute parameters
func NewTracerouteParameters(host string, options TraceOptions) *TracerouteParameters {
	params := &TracerouteParameters{
		BaseParameters: NewParameters(),
	}
	params.Set("host", host)
	params.Set("max_hops", options.MaxHops)
	params.Set("timeout", options.Timeout)
	params.Set("packet_size", options.PacketSize)
	params.Set("queries", options.Queries)
	params.Set("ipv6", options.IPv6)
	return params
}

// Validate validates traceroute parameters
func (p *TracerouteParameters) Validate() error {
	host := p.Get("host")
	if host == nil || host.(string) == "" {
		return fmt.Errorf("host parameter is required")
	}
	
	maxHops := p.Get("max_hops")
	if maxHops != nil && (maxHops.(int) <= 0 || maxHops.(int) > 255) {
		return fmt.Errorf("max_hops must be between 1 and 255")
	}
	
	queries := p.Get("queries")
	if queries != nil && queries.(int) <= 0 {
		return fmt.Errorf("queries must be positive")
	}
	
	return nil
}

// DNSParameters represents parameters for DNS operations
type DNSParameters struct {
	*BaseParameters
}

// NewDNSParameters creates new DNS parameters
func NewDNSParameters(domain string, recordType DNSRecordType) *DNSParameters {
	params := &DNSParameters{
		BaseParameters: NewParameters(),
	}
	params.Set("domain", domain)
	params.Set("record_type", recordType)
	return params
}

// Validate validates DNS parameters
func (p *DNSParameters) Validate() error {
	domain := p.Get("domain")
	if domain == nil || domain.(string) == "" {
		return fmt.Errorf("domain parameter is required")
	}
	
	recordType := p.Get("record_type")
	if recordType == nil {
		return fmt.Errorf("record_type parameter is required")
	}
	
	// Validate record type is within valid range
	rt, ok := recordType.(DNSRecordType)
	if !ok || rt < DNSRecordTypeA || rt > DNSRecordTypePTR {
		return fmt.Errorf("invalid record_type")
	}
	
	return nil
}

// WHOISParameters represents parameters for WHOIS operations
type WHOISParameters struct {
	*BaseParameters
}

// NewWHOISParameters creates new WHOIS parameters
func NewWHOISParameters(query string) *WHOISParameters {
	params := &WHOISParameters{
		BaseParameters: NewParameters(),
	}
	params.Set("query", query)
	return params
}

// Validate validates WHOIS parameters
func (p *WHOISParameters) Validate() error {
	query := p.Get("query")
	if query == nil || query.(string) == "" {
		return fmt.Errorf("query parameter is required")
	}
	
	return nil
}

// SSLParameters represents parameters for SSL operations
type SSLParameters struct {
	*BaseParameters
}

// NewSSLParameters creates new SSL parameters
func NewSSLParameters(host string, port int) *SSLParameters {
	params := &SSLParameters{
		BaseParameters: NewParameters(),
	}
	params.Set("host", host)
	params.Set("port", port)
	return params
}

// Validate validates SSL parameters
func (p *SSLParameters) Validate() error {
	host := p.Get("host")
	if host == nil || host.(string) == "" {
		return fmt.Errorf("host parameter is required")
	}
	
	port := p.Get("port")
	if port == nil {
		return fmt.Errorf("port parameter is required")
	}
	
	portNum, ok := port.(int)
	if !ok || portNum <= 0 || portNum > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	
	return nil
}