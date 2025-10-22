// Package network provides tests for network client implementations
package network

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

// mockErrorHandler implements domain.ErrorHandler for testing
type mockErrorHandler struct{}

func (m *mockErrorHandler) Handle(err error) error                                                    { return err }
func (m *mockErrorHandler) HandleWithContext(err error, ctx map[string]interface{}) error           { return err }
func (m *mockErrorHandler) CanRecover(err error) bool                                                { return false }
func (m *mockErrorHandler) Recover(err error) error                                                  { return err }

// mockLogger implements domain.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {}
func (m *mockLogger) Info(msg string, fields ...interface{})  {}
func (m *mockLogger) Warn(msg string, fields ...interface{})  {}
func (m *mockLogger) Error(msg string, fields ...interface{}) {}
func (m *mockLogger) Fatal(msg string, fields ...interface{}) {}

func TestNewClient(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	errorHandler := &mockErrorHandler{}
	logger := &mockLogger{}

	client := NewClient(config, errorHandler, logger)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.config != config {
		t.Error("Client config not set correctly")
	}

	if client.errorHandler != errorHandler {
		t.Error("Client error handler not set correctly")
	}

	if client.logger != logger {
		t.Error("Client logger not set correctly")
	}

	if client.retryManager == nil {
		t.Error("Client retry manager not initialized")
	}
}

func TestClient_Ping_ValidHost(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	host := "127.0.0.1"
	opts := domain.PingOptions{
		Count:      3,
		Interval:   100 * time.Millisecond,
		Timeout:    time.Second,
		PacketSize: 64,
		TTL:        64,
		IPv6:       false,
	}

	resultChan, err := client.Ping(ctx, host, opts)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	if resultChan == nil {
		t.Fatal("Ping returned nil result channel")
	}

	// Collect results
	var results []domain.PingResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != opts.Count {
		t.Errorf("Expected %d ping results, got %d", opts.Count, len(results))
	}

	for i, result := range results {
		if result.Sequence != i+1 {
			t.Errorf("Expected sequence %d, got %d", i+1, result.Sequence)
		}

		if result.Host.Hostname != host {
			t.Errorf("Expected hostname %s, got %s", host, result.Host.Hostname)
		}

		if result.PacketSize != opts.PacketSize {
			t.Errorf("Expected packet size %d, got %d", opts.PacketSize, result.PacketSize)
		}
	}
}

func TestClient_Ping_InvalidHost(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	opts := domain.PingOptions{Count: 1}

	// Test empty host
	_, err := client.Ping(ctx, "", opts)
	if err == nil {
		t.Error("Expected error for empty host")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", netErr.Type)
	}

	if netErr.Code != "PING_INVALID_HOST" {
		t.Errorf("Expected error code PING_INVALID_HOST, got %s", netErr.Code)
	}
}

func TestClient_Traceroute_ValidHost(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	host := "127.0.0.1"
	opts := domain.TraceOptions{
		MaxHops:    5,
		Timeout:    time.Second,
		PacketSize: 64,
		Queries:    3,
		IPv6:       false,
	}

	resultChan, err := client.Traceroute(ctx, host, opts)
	if err != nil {
		t.Fatalf("Traceroute failed: %v", err)
	}

	if resultChan == nil {
		t.Fatal("Traceroute returned nil result channel")
	}

	// Collect results
	var hops []domain.TraceHop
	for hop := range resultChan {
		hops = append(hops, hop)
	}

	if len(hops) == 0 {
		t.Error("Expected at least one hop")
	}

	for i, hop := range hops {
		if hop.Number != i+1 {
			t.Errorf("Expected hop number %d, got %d", i+1, hop.Number)
		}
	}
}

func TestClient_Traceroute_InvalidHost(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	opts := domain.TraceOptions{MaxHops: 5}

	// Test empty host
	_, err := client.Traceroute(ctx, "", opts)
	if err == nil {
		t.Error("Expected error for empty host")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", netErr.Type)
	}

	if netErr.Code != "TRACE_INVALID_HOST" {
		t.Errorf("Expected error code TRACE_INVALID_HOST, got %s", netErr.Code)
	}
}

func TestClient_DNSLookup_ValidDomain(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	domainName := "localhost"
	recordType := domain.DNSRecordTypeA

	result, err := client.DNSLookup(ctx, domainName, recordType)
	if err != nil {
		t.Fatalf("DNS lookup failed: %v", err)
	}

	if result.Query != domainName {
		t.Errorf("Expected query %s, got %s", domainName, result.Query)
	}

	if result.RecordType != recordType {
		t.Errorf("Expected record type %v, got %v", recordType, result.RecordType)
	}

	if result.ResponseTime <= 0 {
		t.Error("Expected positive response time")
	}
}

func TestClient_DNSLookup_InvalidDomain(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	recordType := domain.DNSRecordTypeA

	// Test empty domain
	_, err := client.DNSLookup(ctx, "", recordType)
	if err == nil {
		t.Error("Expected error for empty domain")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", netErr.Type)
	}

	if netErr.Code != "DNS_INVALID_DOMAIN" {
		t.Errorf("Expected error code DNS_INVALID_DOMAIN, got %s", netErr.Code)
	}
}

func TestClient_WHOISLookup_ValidQuery(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	query := "example.com"

	result, err := client.WHOISLookup(ctx, query)
	if err != nil {
		t.Fatalf("WHOIS lookup failed: %v", err)
	}

	// WHOIS servers often return domain names in uppercase, so compare case-insensitively
	if strings.ToLower(result.Domain) != strings.ToLower(query) {
		t.Errorf("Expected domain %s, got %s", query, result.Domain)
	}

	if result.Registrar == "" {
		t.Error("Expected non-empty registrar")
	}

	if len(result.NameServers) == 0 {
		t.Error("Expected at least one name server")
	}
}

func TestClient_WHOISLookup_InvalidQuery(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()

	// Test empty query
	_, err := client.WHOISLookup(ctx, "")
	if err == nil {
		t.Error("Expected error for empty query")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", netErr.Type)
	}

	if netErr.Code != "WHOIS_INVALID_QUERY" {
		t.Errorf("Expected error code WHOIS_INVALID_QUERY, got %s", netErr.Code)
	}
}

func TestClient_SSLCheck_ValidHost(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	host := "google.com"
	port := 443

	// Note: This test may fail in environments without internet access
	// In a real test suite, we would mock the TLS connection
	result, err := client.SSLCheck(ctx, host, port)
	if err != nil {
		// Skip test if network is unavailable
		t.Skipf("SSL check failed (network may be unavailable): %v", err)
	}

	if result.Host != host {
		t.Errorf("Expected host %s, got %s", host, result.Host)
	}

	if result.Port != port {
		t.Errorf("Expected port %d, got %d", port, result.Port)
	}
}

func TestClient_SSLCheck_InvalidHost(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	port := 443

	// Test empty host
	_, err := client.SSLCheck(ctx, "", port)
	if err == nil {
		t.Error("Expected error for empty host")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeValidation {
		t.Errorf("Expected validation error, got %v", netErr.Type)
	}

	if netErr.Code != "SSL_INVALID_HOST" {
		t.Errorf("Expected error code SSL_INVALID_HOST, got %s", netErr.Code)
	}
}

func TestClient_SSLCheck_InvalidPort(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	host := "example.com"

	// Test invalid ports
	invalidPorts := []int{0, -1, 65536, 100000}
	for _, port := range invalidPorts {
		_, err := client.SSLCheck(ctx, host, port)
		if err == nil {
			t.Errorf("Expected error for invalid port %d", port)
		}

		netErr, ok := err.(*domain.NetTraceError)
		if !ok {
			t.Errorf("Expected NetTraceError for port %d", port)
		}

		if netErr.Type != domain.ErrorTypeValidation {
			t.Errorf("Expected validation error for port %d, got %v", port, netErr.Type)
		}

		if netErr.Code != "SSL_INVALID_PORT" {
			t.Errorf("Expected error code SSL_INVALID_PORT for port %d, got %s", port, netErr.Code)
		}
	}
}

func TestClient_ValidateHost(t *testing.T) {
	config := &domain.NetworkConfig{}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	// Valid hosts
	validHosts := []string{
		"127.0.0.1",
		"::1",
		"example.com",
		"sub.example.com",
		"192.168.1.1",
		"2001:db8::1",
	}

	for _, host := range validHosts {
		err := client.validateHost(host)
		if err != nil {
			t.Errorf("Expected host %s to be valid, got error: %v", host, err)
		}
	}

	// Invalid hosts
	invalidHosts := []string{
		"",
		string(make([]byte, 300)), // Too long
	}

	for _, host := range invalidHosts {
		err := client.validateHost(host)
		if err == nil {
			t.Errorf("Expected host %s to be invalid", host)
		}
	}
}

func TestClient_ValidateDomain(t *testing.T) {
	config := &domain.NetworkConfig{}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	// Valid domains
	validDomains := []string{
		"example.com",
		"sub.example.com",
		"test.co.uk",
		"localhost",
	}

	for _, domainName := range validDomains {
		err := client.validateDomain(domainName)
		if err != nil {
			t.Errorf("Expected domain %s to be valid, got error: %v", domainName, err)
		}
	}

	// Invalid domains
	invalidDomains := []string{
		"",
		string(make([]byte, 300)), // Too long
	}

	for _, domainName := range invalidDomains {
		err := client.validateDomain(domainName)
		if err == nil {
			t.Errorf("Expected domain %s to be invalid", domainName)
		}
	}
}

func TestClient_IsRetryableNetworkError(t *testing.T) {
	config := &domain.NetworkConfig{}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	// Create a mock network error
	mockNetErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &net.DNSError{
			Err:       "no such host",
			Name:      "nonexistent.example.com",
			IsTimeout: true,
		},
	}

	if !client.isRetryableNetworkError(mockNetErr) {
		t.Error("Expected timeout network error to be retryable")
	}

	// Test non-network error
	regularErr := net.ErrClosed
	if client.isRetryableNetworkError(regularErr) {
		t.Error("Expected non-network error to not be retryable")
	}
}

func TestClient_DNSLookup_AllRecordTypes(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	domainName := "localhost"

	// Test all DNS record types
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}

	for _, recordType := range recordTypes {
		result, err := client.DNSLookup(ctx, domainName, recordType)
		if err != nil {
			// Some record types may not exist for localhost, which is expected
			t.Logf("DNS lookup for %v failed (expected for some record types): %v", recordType, err)
			continue
		}

		if result.Query != domainName {
			t.Errorf("Expected query %s, got %s", domainName, result.Query)
		}

		if result.RecordType != recordType {
			t.Errorf("Expected record type %v, got %v", recordType, result.RecordType)
		}
	}
}

func TestClient_DNSLookup_UnsupportedRecordType(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	domainName := "example.com"
	
	// Use an invalid record type (cast to avoid compile error)
	invalidRecordType := domain.DNSRecordType(999)

	_, err := client.DNSLookup(ctx, domainName, invalidRecordType)
	if err == nil {
		t.Error("Expected error for unsupported record type")
	}
}

func TestClient_DNSLookup_WithRetryFailure(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       1 * time.Millisecond, // Very short timeout to force failure
		RetryAttempts: 2,
		RetryDelay:    1 * time.Millisecond,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	domainName := "nonexistent.invalid.domain.test"
	recordType := domain.DNSRecordTypeA

	_, err := client.DNSLookup(ctx, domainName, recordType)
	if err == nil {
		t.Error("Expected error for nonexistent domain")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeNetwork {
		t.Errorf("Expected network error, got %v", netErr.Type)
	}
}

func TestClient_WHOISLookup_WithRetryFailure(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       1 * time.Millisecond, // Very short timeout
		RetryAttempts: 2,
		RetryDelay:    1 * time.Millisecond,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	query := "example.com"

	// This should succeed since WHOIS is mocked, but test the retry path
	result, err := client.WHOISLookup(ctx, query)
	if err != nil {
		t.Logf("WHOIS lookup failed (may be expected): %v", err)
		return
	}

	if result.Domain != query {
		t.Errorf("Expected domain %s, got %s", query, result.Domain)
	}
}

func TestClient_SSLCheck_WithRetryFailure(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       1 * time.Millisecond, // Very short timeout
		RetryAttempts: 2,
		RetryDelay:    1 * time.Millisecond,
	}
	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})

	ctx := context.Background()
	host := "nonexistent.invalid.domain.test"
	port := 443

	_, err := client.SSLCheck(ctx, host, port)
	if err == nil {
		t.Error("Expected error for nonexistent host")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Error("Expected NetTraceError")
	}

	if netErr.Type != domain.ErrorTypeNetwork {
		t.Errorf("Expected network error, got %v", netErr.Type)
	}
}