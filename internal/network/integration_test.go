// Package network provides integration tests for network client implementations
package network

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

// TestNetworkClientIntegration tests the integration between real and mock clients
func TestNetworkClientIntegration(t *testing.T) {
	// Test that both real and mock clients implement the same interface
	var realClient domain.NetworkClient
	var mockClient domain.NetworkClient

	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	realClient = NewClient(config, &mockErrorHandler{}, &mockLogger{})
	mockClient = NewMockClient()

	if realClient == nil {
		t.Error("Real client should not be nil")
	}

	if mockClient == nil {
		t.Error("Mock client should not be nil")
	}

	// Test that both clients can handle the same operations
	ctx := context.Background()
	host := "example.com"

	// Test ping operations
	testPingOperation(t, ctx, realClient, host)
	testPingOperation(t, ctx, mockClient, host)

	// Test DNS operations
	testDNSOperation(t, ctx, realClient, host)
	testDNSOperation(t, ctx, mockClient, host)

	// Test WHOIS operations
	testWHOISOperation(t, ctx, realClient, host)
	testWHOISOperation(t, ctx, mockClient, host)
}

func testPingOperation(t *testing.T, ctx context.Context, client domain.NetworkClient, host string) {
	opts := domain.PingOptions{
		Count:      2,
		Interval:   100 * time.Millisecond,
		Timeout:    time.Second,
		PacketSize: 64,
	}

	resultChan, err := client.Ping(ctx, host, opts)
	if err != nil {
		t.Logf("Ping operation failed for %T: %v", client, err)
		return
	}

	var results []domain.PingResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) == 0 {
		t.Errorf("Expected at least one ping result from %T", client)
	}
}

func testDNSOperation(t *testing.T, ctx context.Context, client domain.NetworkClient, host string) {
	result, err := client.DNSLookup(ctx, host, domain.DNSRecordTypeA)
	if err != nil {
		t.Logf("DNS operation failed for %T: %v", client, err)
		return
	}

	if result.Query != host {
		t.Errorf("Expected query %s, got %s from %T", host, result.Query, client)
	}
}

func testWHOISOperation(t *testing.T, ctx context.Context, client domain.NetworkClient, host string) {
	result, err := client.WHOISLookup(ctx, host)
	if err != nil {
		t.Logf("WHOIS operation failed for %T: %v", client, err)
		return
	}

	// WHOIS servers often return domain names in uppercase, so compare case-insensitively
	if strings.ToLower(result.Domain) != strings.ToLower(host) {
		t.Errorf("Expected domain %s, got %s from %T", host, result.Domain, client)
	}
}

// TestErrorHandlingConsistency tests that error handling is consistent across implementations
func TestErrorHandlingConsistency(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       100 * time.Millisecond, // Short timeout for testing
		RetryAttempts: 2,
		RetryDelay:    50 * time.Millisecond,
	}

	realClient := NewClient(config, &mockErrorHandler{}, &mockLogger{})
	mockClient := NewMockClient()

	ctx := context.Background()

	// Test invalid host error handling
	testInvalidHostErrors(t, ctx, realClient)
	testInvalidHostErrors(t, ctx, mockClient)

	// Test timeout error handling
	testTimeoutErrors(t, ctx, realClient)
	testTimeoutErrors(t, ctx, mockClient)
}

func testInvalidHostErrors(t *testing.T, ctx context.Context, client domain.NetworkClient) {
	// Only test validation for real client, mock client is more permissive for testing
	if realClient, ok := client.(*Client); ok {
		// Test empty host
		_, err := realClient.Ping(ctx, "", domain.PingOptions{Count: 1})
		if err == nil {
			t.Errorf("Expected error for empty host from %T", client)
		}

		// Test invalid domain for DNS
		_, err = realClient.DNSLookup(ctx, "", domain.DNSRecordTypeA)
		if err == nil {
			t.Errorf("Expected error for empty domain from %T", client)
		}

		// Test invalid SSL port
		_, err = realClient.SSLCheck(ctx, "example.com", 0)
		if err == nil {
			t.Errorf("Expected error for invalid port from %T", client)
		}
	} else {
		// For mock client, just verify it doesn't crash with invalid inputs
		_, _ = client.Ping(ctx, "", domain.PingOptions{Count: 1})
		_, _ = client.DNSLookup(ctx, "", domain.DNSRecordTypeA)
		_, _ = client.SSLCheck(ctx, "example.com", 0)
	}
}

func testTimeoutErrors(t *testing.T, ctx context.Context, client domain.NetworkClient) {
	// Create a context with very short timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// Test ping timeout
	resultChan, err := client.Ping(timeoutCtx, "example.com", domain.PingOptions{Count: 1})
	if err == nil {
		// If no immediate error, check if context cancellation is handled
		for range resultChan {
			// Consume any results
		}
	}

	// Test traceroute timeout
	traceResultChan, err := client.Traceroute(timeoutCtx, "example.com", domain.TraceOptions{MaxHops: 5})
	if err == nil {
		// If no immediate error, check if context cancellation is handled
		for range traceResultChan {
			// Consume any results
		}
	}
}

// TestMockClientBehaviorConfiguration tests advanced mock client configuration
func TestMockClientBehaviorConfiguration(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// Test delay configuration
	host := "delay.example.com"
	delay := 100 * time.Millisecond
	mock.SetPingDelay(host, delay)

	start := time.Now()
	resultChan, err := mock.Ping(ctx, host, domain.PingOptions{Count: 1})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Consume results
	for range resultChan {
	}

	elapsed := time.Since(start)
	if elapsed < delay {
		t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
	}

	// Test error simulation
	errorHost := "error.example.com"
	mock.SetSimulateNetworkError(true)
	
	resultChan, err = mock.Ping(ctx, errorHost, domain.PingOptions{Count: 10})
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	errorCount := 0
	for result := range resultChan {
		if result.Error != nil {
			errorCount++
		}
	}

	if errorCount == 0 {
		t.Error("Expected at least one error when network error simulation is enabled")
	}

	// Test timeout simulation
	mock.SetSimulateTimeout(true)
	
	traceResultChan, err := mock.Traceroute(ctx, "timeout.example.com", domain.TraceOptions{MaxHops: 10})
	if err != nil {
		t.Fatalf("Traceroute failed: %v", err)
	}

	timeoutCount := 0
	for hop := range traceResultChan {
		if hop.Timeout {
			timeoutCount++
		}
	}

	if timeoutCount == 0 {
		t.Error("Expected at least one timeout when timeout simulation is enabled")
	}
}

// TestRetryManagerIntegration tests retry manager integration with network operations
func TestRetryManagerIntegration(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       50 * time.Millisecond,
		RetryAttempts: 3,
		RetryDelay:    10 * time.Millisecond,
	}

	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})
	ctx := context.Background()

	// Test DNS lookup with retry (using a domain that should fail)
	start := time.Now()
	_, err := client.DNSLookup(ctx, "nonexistent.invalid.test.domain", domain.DNSRecordTypeA)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected error for nonexistent domain")
	}

	// Should have taken some time due to retries
	expectedMinTime := time.Duration(config.RetryAttempts-1) * config.RetryDelay
	if elapsed < expectedMinTime {
		t.Errorf("Expected at least %v for retries, got %v", expectedMinTime, elapsed)
	}

	// Verify it's a NetTraceError with retry information
	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Errorf("Expected NetTraceError, got %T", err)
	}

	if netErr.Type != domain.ErrorTypeNetwork {
		t.Errorf("Expected network error type, got %v", netErr.Type)
	}
}

// TestConcurrentOperations tests that network clients handle concurrent operations correctly
func TestConcurrentOperations(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// Test concurrent ping operations
	const numOperations = 5
	results := make(chan error, numOperations)

	for i := 0; i < numOperations; i++ {
		go func(id int) {
			host := "concurrent.example.com"
			resultChan, err := mock.Ping(ctx, host, domain.PingOptions{Count: 2})
			if err != nil {
				results <- err
				return
			}

			// Consume results
			for range resultChan {
			}
			results <- nil
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < numOperations; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent operation %d failed: %v", i, err)
		}
	}

	// Verify all calls were recorded
	calls := mock.GetPingCalls()
	if len(calls) != numOperations {
		t.Errorf("Expected %d ping calls, got %d", numOperations, len(calls))
	}

	// Verify total call count
	if mock.GetCallCount() != numOperations {
		t.Errorf("Expected total call count %d, got %d", numOperations, mock.GetCallCount())
	}
}

// TestNetworkClientValidation tests input validation across all network operations
func TestNetworkClientValidation(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:       5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}

	client := NewClient(config, &mockErrorHandler{}, &mockLogger{})
	ctx := context.Background()

	// Test ping validation
	testPingValidation(t, ctx, client)

	// Test traceroute validation
	testTracerouteValidation(t, ctx, client)

	// Test DNS validation
	testDNSValidation(t, ctx, client)

	// Test WHOIS validation
	testWHOISValidation(t, ctx, client)

	// Test SSL validation
	testSSLValidation(t, ctx, client)
}

func testPingValidation(t *testing.T, ctx context.Context, client *Client) {
	invalidHosts := []string{"", string(make([]byte, 300))}
	
	for _, host := range invalidHosts {
		_, err := client.Ping(ctx, host, domain.PingOptions{Count: 1})
		if err == nil {
			t.Errorf("Expected error for invalid ping host: %s", host)
		}

		netErr, ok := err.(*domain.NetTraceError)
		if !ok {
			t.Errorf("Expected NetTraceError for ping validation, got %T", err)
		}

		if netErr.Type != domain.ErrorTypeValidation {
			t.Errorf("Expected validation error type, got %v", netErr.Type)
		}
	}
}

func testTracerouteValidation(t *testing.T, ctx context.Context, client *Client) {
	invalidHosts := []string{"", string(make([]byte, 300))}
	
	for _, host := range invalidHosts {
		_, err := client.Traceroute(ctx, host, domain.TraceOptions{MaxHops: 5})
		if err == nil {
			t.Errorf("Expected error for invalid traceroute host: %s", host)
		}

		netErr, ok := err.(*domain.NetTraceError)
		if !ok {
			t.Errorf("Expected NetTraceError for traceroute validation, got %T", err)
		}

		if netErr.Type != domain.ErrorTypeValidation {
			t.Errorf("Expected validation error type, got %v", netErr.Type)
		}
	}
}

func testDNSValidation(t *testing.T, ctx context.Context, client *Client) {
	invalidDomains := []string{"", string(make([]byte, 300))}
	
	for _, domainName := range invalidDomains {
		_, err := client.DNSLookup(ctx, domainName, domain.DNSRecordTypeA)
		if err == nil {
			t.Errorf("Expected error for invalid DNS domain: %s", domainName)
		}

		netErr, ok := err.(*domain.NetTraceError)
		if !ok {
			t.Errorf("Expected NetTraceError for DNS validation, got %T", err)
		}

		if netErr.Type != domain.ErrorTypeValidation {
			t.Errorf("Expected validation error type, got %v", netErr.Type)
		}
	}
}

func testWHOISValidation(t *testing.T, ctx context.Context, client *Client) {
	_, err := client.WHOISLookup(ctx, "")
	if err == nil {
		t.Error("Expected error for empty WHOIS query")
	}

	netErr, ok := err.(*domain.NetTraceError)
	if !ok {
		t.Errorf("Expected NetTraceError for WHOIS validation, got %T", err)
	}

	if netErr.Type != domain.ErrorTypeValidation {
		t.Errorf("Expected validation error type, got %v", netErr.Type)
	}
}

func testSSLValidation(t *testing.T, ctx context.Context, client *Client) {
	// Test invalid host
	_, err := client.SSLCheck(ctx, "", 443)
	if err == nil {
		t.Error("Expected error for empty SSL host")
	}

	// Test invalid ports
	invalidPorts := []int{0, -1, 65536, 100000}
	for _, port := range invalidPorts {
		_, err := client.SSLCheck(ctx, "example.com", port)
		if err == nil {
			t.Errorf("Expected error for invalid SSL port: %d", port)
		}

		netErr, ok := err.(*domain.NetTraceError)
		if !ok {
			t.Errorf("Expected NetTraceError for SSL validation, got %T", err)
		}

		if netErr.Type != domain.ErrorTypeValidation {
			t.Errorf("Expected validation error type, got %v", netErr.Type)
		}
	}
}