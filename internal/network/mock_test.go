// Package network provides tests for mock network client
package network

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
)

func TestNewMockClient(t *testing.T) {
	mock := NewMockClient()

	if mock == nil {
		t.Fatal("NewMockClient returned nil")
	}

	if mock.pingResponses == nil {
		t.Error("Mock ping responses not initialized")
	}

	if mock.traceResponses == nil {
		t.Error("Mock trace responses not initialized")
	}

	if mock.dnsResponses == nil {
		t.Error("Mock DNS responses not initialized")
	}

	if mock.whoisResponses == nil {
		t.Error("Mock WHOIS responses not initialized")
	}

	if mock.sslResponses == nil {
		t.Error("Mock SSL responses not initialized")
	}
}

func TestMockClient_Ping(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	host := "example.com"
	opts := domain.PingOptions{
		Count:      3,
		Interval:   100 * time.Millisecond,
		Timeout:    time.Second,
		PacketSize: 64,
	}

	// Test default behavior
	resultChan, err := mock.Ping(ctx, host, opts)
	if err != nil {
		t.Fatalf("Mock ping failed: %v", err)
	}

	var results []domain.PingResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != opts.Count {
		t.Errorf("Expected %d results, got %d", opts.Count, len(results))
	}

	// Verify call was recorded
	calls := mock.GetPingCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 ping call, got %d", len(calls))
	}

	if calls[0].Method != "Ping" {
		t.Errorf("Expected method 'Ping', got %s", calls[0].Method)
	}

	if mock.GetCallCount() != 1 {
		t.Errorf("Expected call count 1, got %d", mock.GetCallCount())
	}
}

func TestMockClient_Ping_WithConfiguredResponse(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	host := "example.com"
	opts := domain.PingOptions{Count: 2}

	// Configure custom response
	customResults := []domain.PingResult{
		{
			Host: domain.NetworkHost{
				Hostname:  host,
				IPAddress: net.IPv4(192, 168, 1, 1),
			},
			Sequence:   1,
			RTT:        10 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  time.Now(),
		},
		{
			Host: domain.NetworkHost{
				Hostname:  host,
				IPAddress: net.IPv4(192, 168, 1, 1),
			},
			Sequence:   2,
			RTT:        15 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  time.Now(),
			Error:      fmt.Errorf("timeout"),
		},
	}

	mock.SetPingResponse(host, customResults)

	resultChan, err := mock.Ping(ctx, host, opts)
	if err != nil {
		t.Fatalf("Mock ping failed: %v", err)
	}

	var results []domain.PingResult
	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0].RTT != 10*time.Millisecond {
		t.Errorf("Expected RTT 10ms, got %v", results[0].RTT)
	}

	if results[1].Error == nil {
		t.Error("Expected error in second result")
	}
}

func TestMockClient_Ping_WithError(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	host := "error.example.com"
	opts := domain.PingOptions{Count: 1}

	// Configure error response
	expectedErr := fmt.Errorf("network unreachable")
	mock.SetPingError(host, expectedErr)

	_, err := mock.Ping(ctx, host, opts)
	if err == nil {
		t.Error("Expected error from mock ping")
	}

	if err.Error() != expectedErr.Error() {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestMockClient_Ping_WithDelay(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	host := "slow.example.com"
	opts := domain.PingOptions{Count: 1}

	// Configure delay
	delay := 100 * time.Millisecond
	mock.SetPingDelay(host, delay)

	start := time.Now()
	resultChan, err := mock.Ping(ctx, host, opts)
	if err != nil {
		t.Fatalf("Mock ping failed: %v", err)
	}

	// Consume results
	for range resultChan {
	}

	elapsed := time.Since(start)
	if elapsed < delay {
		t.Errorf("Expected delay of at least %v, got %v", delay, elapsed)
	}
}

func TestMockClient_Traceroute(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	host := "example.com"
	opts := domain.TraceOptions{
		MaxHops: 5,
		Queries: 3,
	}

	resultChan, err := mock.Traceroute(ctx, host, opts)
	if err != nil {
		t.Fatalf("Mock traceroute failed: %v", err)
	}

	var hops []domain.TraceHop
	for hop := range resultChan {
		hops = append(hops, hop)
	}

	if len(hops) == 0 {
		t.Error("Expected at least one hop")
	}

	// Verify call was recorded
	calls := mock.GetTraceCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 traceroute call, got %d", len(calls))
	}
}

func TestMockClient_DNSLookup(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	domainName := "example.com"
	recordType := domain.DNSRecordTypeA

	result, err := mock.DNSLookup(ctx, domainName, recordType)
	if err != nil {
		t.Fatalf("Mock DNS lookup failed: %v", err)
	}

	if result.Query != domainName {
		t.Errorf("Expected query %s, got %s", domainName, result.Query)
	}

	if result.RecordType != recordType {
		t.Errorf("Expected record type %v, got %v", recordType, result.RecordType)
	}

	// Verify call was recorded
	calls := mock.GetDNSCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 DNS call, got %d", len(calls))
	}
}

func TestMockClient_DNSLookup_WithConfiguredResponse(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	domainName := "custom.example.com"
	recordType := domain.DNSRecordTypeA

	// Configure custom response
	customResult := domain.DNSResult{
		Query:      domainName,
		RecordType: recordType,
		Records: []domain.DNSRecord{
			{
				Name:  domainName,
				Type:  recordType,
				Value: "203.0.113.1",
				TTL:   600,
			},
		},
		ResponseTime: 25 * time.Millisecond,
		Server:       "custom-server",
	}

	mock.SetDNSResponse(domainName, recordType, customResult)

	result, err := mock.DNSLookup(ctx, domainName, recordType)
	if err != nil {
		t.Fatalf("Mock DNS lookup failed: %v", err)
	}

	if result.Server != "custom-server" {
		t.Errorf("Expected server 'custom-server', got %s", result.Server)
	}

	if len(result.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result.Records))
	}

	if result.Records[0].Value != "203.0.113.1" {
		t.Errorf("Expected IP 203.0.113.1, got %s", result.Records[0].Value)
	}
}

func TestMockClient_WHOISLookup(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	query := "example.com"

	result, err := mock.WHOISLookup(ctx, query)
	if err != nil {
		t.Fatalf("Mock WHOIS lookup failed: %v", err)
	}

	if result.Domain != query {
		t.Errorf("Expected domain %s, got %s", query, result.Domain)
	}

	// Verify call was recorded
	calls := mock.GetWHOISCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 WHOIS call, got %d", len(calls))
	}
}

func TestMockClient_SSLCheck(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()
	host := "example.com"
	port := 443

	result, err := mock.SSLCheck(ctx, host, port)
	if err != nil {
		t.Fatalf("Mock SSL check failed: %v", err)
	}

	if result.Host != host {
		t.Errorf("Expected host %s, got %s", host, result.Host)
	}

	if result.Port != port {
		t.Errorf("Expected port %d, got %d", port, result.Port)
	}

	// Verify call was recorded
	calls := mock.GetSSLCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 SSL call, got %d", len(calls))
	}
}

func TestMockClient_Reset(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// Make some calls
	mock.Ping(ctx, "example.com", domain.PingOptions{Count: 1})
	mock.DNSLookup(ctx, "example.com", domain.DNSRecordTypeA)

	if mock.GetCallCount() != 2 {
		t.Errorf("Expected call count 2, got %d", mock.GetCallCount())
	}

	// Reset mock
	mock.Reset()

	if mock.GetCallCount() != 0 {
		t.Errorf("Expected call count 0 after reset, got %d", mock.GetCallCount())
	}

	if len(mock.GetPingCalls()) != 0 {
		t.Errorf("Expected 0 ping calls after reset, got %d", len(mock.GetPingCalls()))
	}

	if len(mock.GetDNSCalls()) != 0 {
		t.Errorf("Expected 0 DNS calls after reset, got %d", len(mock.GetDNSCalls()))
	}
}

func TestMockClient_SimulateTimeout(t *testing.T) {
	mock := NewMockClient()
	mock.SetSimulateTimeout(true)

	ctx := context.Background()
	host := "example.com"
	opts := domain.TraceOptions{MaxHops: 20}

	resultChan, err := mock.Traceroute(ctx, host, opts)
	if err != nil {
		t.Fatalf("Mock traceroute failed: %v", err)
	}

	var timeoutCount int
	for hop := range resultChan {
		if hop.Timeout {
			timeoutCount++
		}
	}

	if timeoutCount == 0 {
		t.Error("Expected at least one timeout when simulation is enabled")
	}
}

func TestMockClient_SimulateNetworkError(t *testing.T) {
	mock := NewMockClient()
	mock.SetSimulateNetworkError(true)

	ctx := context.Background()
	host := "example.com"
	opts := domain.PingOptions{Count: 20}

	resultChan, err := mock.Ping(ctx, host, opts)
	if err != nil {
		t.Fatalf("Mock ping failed: %v", err)
	}

	var errorCount int
	for result := range resultChan {
		if result.Error != nil {
			errorCount++
		}
	}

	if errorCount == 0 {
		t.Error("Expected at least one error when simulation is enabled")
	}
}

func TestMockClient_GenerateDefaultResults(t *testing.T) {
	mock := NewMockClient()

	// Test ping results generation
	host := "test.example.com"
	opts := domain.PingOptions{Count: 3, PacketSize: 64}
	pingResults := mock.generateDefaultPingResults(host, opts)

	if len(pingResults) != opts.Count {
		t.Errorf("Expected %d ping results, got %d", opts.Count, len(pingResults))
	}

	for i, result := range pingResults {
		if result.Sequence != i+1 {
			t.Errorf("Expected sequence %d, got %d", i+1, result.Sequence)
		}
		if result.PacketSize != opts.PacketSize {
			t.Errorf("Expected packet size %d, got %d", opts.PacketSize, result.PacketSize)
		}
	}

	// Test traceroute results generation
	traceOpts := domain.TraceOptions{MaxHops: 5, Queries: 3}
	traceResults := mock.generateDefaultTraceResults(host, traceOpts)

	if len(traceResults) > traceOpts.MaxHops {
		t.Errorf("Expected at most %d hops, got %d", traceOpts.MaxHops, len(traceResults))
	}

	for i, hop := range traceResults {
		if hop.Number != i+1 {
			t.Errorf("Expected hop number %d, got %d", i+1, hop.Number)
		}
		if !hop.Timeout && len(hop.RTT) != traceOpts.Queries {
			t.Errorf("Expected %d RTT values, got %d", traceOpts.Queries, len(hop.RTT))
		}
	}

	// Test DNS results generation
	dnsResult := mock.generateDefaultDNSResult("test.com", domain.DNSRecordTypeA)
	if dnsResult.Query != "test.com" {
		t.Errorf("Expected query 'test.com', got %s", dnsResult.Query)
	}
	if dnsResult.RecordType != domain.DNSRecordTypeA {
		t.Errorf("Expected A record type, got %v", dnsResult.RecordType)
	}
	if len(dnsResult.Records) == 0 {
		t.Error("Expected at least one DNS record")
	}

	// Test WHOIS results generation
	whoisResult := mock.generateDefaultWHOISResult("test.com")
	if whoisResult.Domain != "test.com" {
		t.Errorf("Expected domain 'test.com', got %s", whoisResult.Domain)
	}
	if whoisResult.Registrar == "" {
		t.Error("Expected non-empty registrar")
	}

	// Test SSL results generation
	sslResult := mock.generateDefaultSSLResult("test.com", 443)
	if sslResult.Host != "test.com" {
		t.Errorf("Expected host 'test.com', got %s", sslResult.Host)
	}
	if sslResult.Port != 443 {
		t.Errorf("Expected port 443, got %d", sslResult.Port)
	}
	if !sslResult.Valid {
		t.Error("Expected valid SSL result by default")
	}
}

func TestMockClient_ConfigurationMethods(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	// Test SetTraceResponse and SetTraceError
	host := "trace.example.com"
	traceOpts := domain.TraceOptions{MaxHops: 3}
	
	customHops := []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "hop1.example.com",
				IPAddress: net.IPv4(10, 0, 0, 1),
			},
			RTT:       []time.Duration{10 * time.Millisecond},
			Timestamp: time.Now(),
		},
	}
	
	mock.SetTraceResponse(host, customHops)
	
	resultChan, err := mock.Traceroute(ctx, host, traceOpts)
	if err != nil {
		t.Fatalf("Traceroute failed: %v", err)
	}
	
	var hops []domain.TraceHop
	for hop := range resultChan {
		hops = append(hops, hop)
	}
	
	if len(hops) != 1 {
		t.Errorf("Expected 1 hop, got %d", len(hops))
	}
	
	if hops[0].Host.Hostname != "hop1.example.com" {
		t.Errorf("Expected hostname 'hop1.example.com', got %s", hops[0].Host.Hostname)
	}

	// Test SetTraceError
	errorHost := "error.example.com"
	expectedErr := fmt.Errorf("traceroute failed")
	mock.SetTraceError(errorHost, expectedErr)
	
	_, err = mock.Traceroute(ctx, errorHost, traceOpts)
	if err == nil {
		t.Error("Expected error from traceroute")
	}
	
	if err.Error() != expectedErr.Error() {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Test SetDNSError
	dnsHost := "dns.example.com"
	recordType := domain.DNSRecordTypeA
	dnsErr := fmt.Errorf("DNS lookup failed")
	mock.SetDNSError(dnsHost, recordType, dnsErr)
	
	_, err = mock.DNSLookup(ctx, dnsHost, recordType)
	if err == nil {
		t.Error("Expected error from DNS lookup")
	}
	
	if err.Error() != dnsErr.Error() {
		t.Errorf("Expected error %v, got %v", dnsErr, err)
	}

	// Test SetWHOISResponse and SetWHOISError
	whoisQuery := "whois.example.com"
	customWHOIS := domain.WHOISResult{
		Domain:    whoisQuery,
		Registrar: "Custom Registrar",
		Created:   time.Now().AddDate(-1, 0, 0),
		Expires:   time.Now().AddDate(1, 0, 0),
	}
	
	mock.SetWHOISResponse(whoisQuery, customWHOIS)
	
	result, err := mock.WHOISLookup(ctx, whoisQuery)
	if err != nil {
		t.Fatalf("WHOIS lookup failed: %v", err)
	}
	
	if result.Registrar != "Custom Registrar" {
		t.Errorf("Expected registrar 'Custom Registrar', got %s", result.Registrar)
	}

	// Test SetWHOISError
	errorQuery := "error.whois.com"
	whoisErr := fmt.Errorf("WHOIS lookup failed")
	mock.SetWHOISError(errorQuery, whoisErr)
	
	_, err = mock.WHOISLookup(ctx, errorQuery)
	if err == nil {
		t.Error("Expected error from WHOIS lookup")
	}

	// Test SetSSLResponse and SetSSLError
	sslHost := "ssl.example.com"
	sslPort := 443
	customSSL := domain.SSLResult{
		Host:    sslHost,
		Port:    sslPort,
		Valid:   false,
		Errors:  []string{"Certificate expired"},
		Issuer:  "Custom CA",
		Subject: "CN=ssl.example.com",
	}
	
	mock.SetSSLResponse(sslHost, sslPort, customSSL)
	
	sslResult, err := mock.SSLCheck(ctx, sslHost, sslPort)
	if err != nil {
		t.Fatalf("SSL check failed: %v", err)
	}
	
	if sslResult.Valid {
		t.Error("Expected invalid SSL result")
	}
	
	if len(sslResult.Errors) != 1 || sslResult.Errors[0] != "Certificate expired" {
		t.Errorf("Expected error 'Certificate expired', got %v", sslResult.Errors)
	}

	// Test SetSSLError
	errorSSLHost := "error.ssl.com"
	sslErr := fmt.Errorf("SSL connection failed")
	mock.SetSSLError(errorSSLHost, sslPort, sslErr)
	
	_, err = mock.SSLCheck(ctx, errorSSLHost, sslPort)
	if err == nil {
		t.Error("Expected error from SSL check")
	}
}

func TestMockClient_DNSRecordTypes(t *testing.T) {
	mock := NewMockClient()

	// Test all DNS record types for default generation
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}

	for _, recordType := range recordTypes {
		result := mock.generateDefaultDNSResult("test.com", recordType)
		
		if result.Query != "test.com" {
			t.Errorf("Expected query 'test.com', got %s", result.Query)
		}
		
		if result.RecordType != recordType {
			t.Errorf("Expected record type %v, got %v", recordType, result.RecordType)
		}
		
		if len(result.Records) == 0 {
			t.Errorf("Expected at least one record for type %v", recordType)
		}
		
		// Verify record type matches
		for _, record := range result.Records {
			if record.Type != recordType {
				t.Errorf("Expected record type %v, got %v", recordType, record.Type)
			}
		}
	}
}