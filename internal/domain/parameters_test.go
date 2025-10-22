package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBaseParameters(t *testing.T) {
	params := NewParameters()
	
	// Test Set and Get
	params.Set("host", "example.com")
	params.Set("port", 80)
	params.Set("timeout", 30*time.Second)
	
	assert.Equal(t, "example.com", params.Get("host"))
	assert.Equal(t, 80, params.Get("port"))
	assert.Equal(t, 30*time.Second, params.Get("timeout"))
	assert.Nil(t, params.Get("nonexistent"))
	
	// Test ToMap
	paramMap := params.ToMap()
	assert.Len(t, paramMap, 3)
	assert.Equal(t, "example.com", paramMap["host"])
	assert.Equal(t, 80, paramMap["port"])
	assert.Equal(t, 30*time.Second, paramMap["timeout"])
	
	// Test Validate with valid parameters
	err := params.Validate()
	assert.NoError(t, err)
}

func TestBaseParametersValidation(t *testing.T) {
	params := NewParameters()
	
	// Test validation with empty string
	params.Set("host", "")
	err := params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
	
	// Test validation with nil value (should pass)
	params.Set("optional", nil)
	params.Set("host", "example.com") // Fix the empty string
	err = params.Validate()
	assert.NoError(t, err)
}

func TestPingParameters(t *testing.T) {
	options := PingOptions{
		Count:      10,
		Interval:   time.Second,
		Timeout:    5 * time.Second,
		PacketSize: 64,
		TTL:        64,
		IPv6:       false,
	}
	
	params := NewPingParameters("example.com", options)
	
	// Test parameter values
	assert.Equal(t, "example.com", params.Get("host"))
	assert.Equal(t, 10, params.Get("count"))
	assert.Equal(t, time.Second, params.Get("interval"))
	assert.Equal(t, 5*time.Second, params.Get("timeout"))
	assert.Equal(t, 64, params.Get("packet_size"))
	assert.Equal(t, 64, params.Get("ttl"))
	assert.Equal(t, false, params.Get("ipv6"))
	
	// Test validation with valid parameters
	err := params.Validate()
	assert.NoError(t, err)
}

func TestPingParametersValidation(t *testing.T) {
	// Test missing host
	params := NewPingParameters("", PingOptions{})
	err := params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host parameter is required")
	
	// Test invalid count
	params = NewPingParameters("example.com", PingOptions{Count: -1})
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "count must be positive")
	
	// Test invalid packet size (too small)
	params = NewPingParameters("example.com", PingOptions{Count: 1, PacketSize: 0})
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packet_size must be between 1 and 65507")
	
	// Test invalid packet size (too large)
	params = NewPingParameters("example.com", PingOptions{Count: 1, PacketSize: 70000})
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packet_size must be between 1 and 65507")
}

func TestTracerouteParameters(t *testing.T) {
	options := TraceOptions{
		MaxHops:    30,
		Timeout:    5 * time.Second,
		PacketSize: 64,
		Queries:    3,
		IPv6:       false,
	}
	
	params := NewTracerouteParameters("example.com", options)
	
	// Test parameter values
	assert.Equal(t, "example.com", params.Get("host"))
	assert.Equal(t, 30, params.Get("max_hops"))
	assert.Equal(t, 5*time.Second, params.Get("timeout"))
	assert.Equal(t, 64, params.Get("packet_size"))
	assert.Equal(t, 3, params.Get("queries"))
	assert.Equal(t, false, params.Get("ipv6"))
	
	// Test validation with valid parameters
	err := params.Validate()
	assert.NoError(t, err)
}

func TestTracerouteParametersValidation(t *testing.T) {
	// Test missing host
	params := NewTracerouteParameters("", TraceOptions{})
	err := params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host parameter is required")
	
	// Test invalid max_hops (too small)
	params = NewTracerouteParameters("example.com", TraceOptions{MaxHops: 0})
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_hops must be between 1 and 255")
	
	// Test invalid max_hops (too large)
	params = NewTracerouteParameters("example.com", TraceOptions{MaxHops: 300})
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_hops must be between 1 and 255")
	
	// Test invalid queries
	params = NewTracerouteParameters("example.com", TraceOptions{MaxHops: 30, Queries: 0})
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queries must be positive")
}

func TestDNSParameters(t *testing.T) {
	params := NewDNSParameters("example.com", DNSRecordTypeA)
	
	// Test parameter values
	assert.Equal(t, "example.com", params.Get("domain"))
	assert.Equal(t, DNSRecordTypeA, params.Get("record_type"))
	
	// Test validation with valid parameters
	err := params.Validate()
	assert.NoError(t, err)
}

func TestDNSParametersValidation(t *testing.T) {
	// Test missing domain
	params := NewDNSParameters("", DNSRecordTypeA)
	err := params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "domain parameter is required")
	
	// Test missing record type
	params = NewDNSParameters("example.com", DNSRecordTypeA)
	params.Set("record_type", nil)
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "record_type parameter is required")
	
	// Test invalid record type
	params = NewDNSParameters("example.com", DNSRecordTypeA)
	params.Set("record_type", "invalid")
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid record_type")
	
	// Test out of range record type
	params = NewDNSParameters("example.com", DNSRecordTypeA)
	params.Set("record_type", DNSRecordType(999))
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid record_type")
}

func TestWHOISParameters(t *testing.T) {
	params := NewWHOISParameters("example.com")
	
	// Test parameter values
	assert.Equal(t, "example.com", params.Get("query"))
	
	// Test validation with valid parameters
	err := params.Validate()
	assert.NoError(t, err)
}

func TestWHOISParametersValidation(t *testing.T) {
	// Test missing query
	params := NewWHOISParameters("")
	err := params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query parameter is required")
}

func TestSSLParameters(t *testing.T) {
	params := NewSSLParameters("example.com", 443)
	
	// Test parameter values
	assert.Equal(t, "example.com", params.Get("host"))
	assert.Equal(t, 443, params.Get("port"))
	
	// Test validation with valid parameters
	err := params.Validate()
	assert.NoError(t, err)
}

func TestSSLParametersValidation(t *testing.T) {
	// Test missing host
	params := NewSSLParameters("", 443)
	err := params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host parameter is required")
	
	// Test missing port
	params = NewSSLParameters("example.com", 443)
	params.Set("port", nil)
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port parameter is required")
	
	// Test invalid port (too small)
	params = NewSSLParameters("example.com", 0)
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
	
	// Test invalid port (too large)
	params = NewSSLParameters("example.com", 70000)
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
	
	// Test invalid port type
	params = NewSSLParameters("example.com", 443)
	params.Set("port", "invalid")
	err = params.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
}

func TestParametersInterfaceCompliance(t *testing.T) {
	// Test that all parameter types implement the Parameters interface
	var _ Parameters = (*BaseParameters)(nil)
	var _ Parameters = (*PingParameters)(nil)
	var _ Parameters = (*TracerouteParameters)(nil)
	var _ Parameters = (*DNSParameters)(nil)
	var _ Parameters = (*WHOISParameters)(nil)
	var _ Parameters = (*SSLParameters)(nil)
}