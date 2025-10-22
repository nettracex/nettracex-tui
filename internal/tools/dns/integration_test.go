// Package dns provides integration tests for DNS diagnostic tool
package dns

import (
	"context"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)

func TestDNSTool_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create real network client for integration testing
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     1 * time.Second,
	}
	
	mockLogger := &MockLogger{}
	client := network.NewClient(config, nil, mockLogger)
	tool := NewTool(client, mockLogger)
	
	tests := []struct {
		name           string
		domainName     string
		recordTypes    []domain.DNSRecordType
		expectRecords  bool
		expectError    bool
	}{
		{
			name:       "lookup google.com A records",
			domainName: "google.com",
			recordTypes: []domain.DNSRecordType{
				domain.DNSRecordTypeA,
			},
			expectRecords: true,
			expectError:   false,
		},
		{
			name:       "lookup google.com multiple record types",
			domainName: "google.com",
			recordTypes: []domain.DNSRecordType{
				domain.DNSRecordTypeA,
				domain.DNSRecordTypeAAAA,
				domain.DNSRecordTypeMX,
				domain.DNSRecordTypeNS,
			},
			expectRecords: true,
			expectError:   false,
		},
		{
			name:       "lookup cloudflare.com TXT records",
			domainName: "cloudflare.com",
			recordTypes: []domain.DNSRecordType{
				domain.DNSRecordTypeTXT,
			},
			expectRecords: true,
			expectError:   false,
		},
		{
			name:       "lookup nonexistent domain",
			domainName: "this-domain-definitely-does-not-exist-12345.com",
			recordTypes: []domain.DNSRecordType{
				domain.DNSRecordTypeA,
			},
			expectRecords: false,
			expectError:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parameters
			params := domain.NewDNSParameters(tt.domainName, domain.DNSRecordTypeA)
			if len(tt.recordTypes) > 0 {
				params.Set("record_types", tt.recordTypes)
			}
			
			// Execute DNS lookup
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			result, err := tool.Execute(ctx, params)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}
			
			if result == nil {
				t.Error("Expected result but got nil")
				return
			}
			
			// Verify result metadata
			metadata := result.Metadata()
			if metadata["tool"] != "dns" {
				t.Errorf("Expected tool metadata 'dns', got '%v'", metadata["tool"])
			}
			
			if metadata["domain"] != tt.domainName {
				t.Errorf("Expected domain metadata '%s', got '%v'", tt.domainName, metadata["domain"])
			}
			
			// Verify result data
			dnsResult, ok := result.Data().(domain.DNSResult)
			if !ok {
				t.Error("Expected DNSResult data type")
				return
			}
			
			if dnsResult.Query != tt.domainName {
				t.Errorf("Expected query '%s', got '%s'", tt.domainName, dnsResult.Query)
			}
			
			if tt.expectRecords {
				if len(dnsResult.Records) == 0 {
					t.Error("Expected at least one DNS record but got none")
				}
				
				// Verify response time is reasonable
				if dnsResult.ResponseTime <= 0 {
					t.Error("Expected positive response time")
				}
				
				if dnsResult.ResponseTime > 10*time.Second {
					t.Errorf("Response time too high: %v", dnsResult.ResponseTime)
				}
				
				// Verify records have valid data
				for i, record := range dnsResult.Records {
					if record.Name == "" {
						t.Errorf("Record %d has empty name", i)
					}
					
					if record.Value == "" {
						t.Errorf("Record %d has empty value", i)
					}
					
					if record.TTL == 0 {
						t.Errorf("Record %d has zero TTL", i)
					}
					
					// Verify record type is one of the requested types
					if len(tt.recordTypes) > 0 {
						found := false
						for _, requestedType := range tt.recordTypes {
							if record.Type == requestedType {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Record %d has unexpected type %v", i, record.Type)
						}
					}
				}
			}
			
			// Test result validation
			if err := ValidateDNSResult(dnsResult); err != nil {
				t.Errorf("DNS result validation failed: %v", err)
			}
			
			// Test result formatting
			formatted := FormatDNSResult(dnsResult)
			if formatted == "" {
				t.Error("Formatted result should not be empty")
			}
			
			if !containsIgnoreCase(formatted, tt.domainName) {
				t.Error("Formatted result should contain domain name")
			}
		})
	}
}

func TestDNSTool_ConcurrentLookups_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create real network client for integration testing
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxConcurrency: 3,
		RetryAttempts:  1,
		RetryDelay:     500 * time.Millisecond,
	}
	
	mockLogger := &MockLogger{}
	client := network.NewClient(config, nil, mockLogger)
	tool := NewTool(client, mockLogger)
	
	// Test concurrent lookups for multiple record types
	domainName := "google.com"
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeNS,
		domain.DNSRecordTypeTXT,
	}
	
	params := domain.NewDNSParameters(domainName, domain.DNSRecordTypeA)
	params.Set("record_types", recordTypes)
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	startTime := time.Now()
	result, err := tool.Execute(ctx, params)
	duration := time.Since(startTime)
	
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
		return
	}
	
	if result == nil {
		t.Error("Expected result but got nil")
		return
	}
	
	dnsResult, ok := result.Data().(domain.DNSResult)
	if !ok {
		t.Error("Expected DNSResult data type")
		return
	}
	
	// Verify we got records for multiple types
	recordTypesSeen := make(map[domain.DNSRecordType]bool)
	for _, record := range dnsResult.Records {
		recordTypesSeen[record.Type] = true
	}
	
	if len(recordTypesSeen) < 2 {
		t.Errorf("Expected records for multiple types, got types: %v", recordTypesSeen)
	}
	
	// Verify concurrent execution was faster than sequential would be
	// (This is a rough heuristic - concurrent should be significantly faster)
	expectedSequentialTime := time.Duration(len(recordTypes)) * 1 * time.Second
	if duration > expectedSequentialTime {
		t.Errorf("Concurrent lookup took too long: %v (expected less than %v)", duration, expectedSequentialTime)
	}
	
	t.Logf("Concurrent lookup for %d record types completed in %v", len(recordTypes), duration)
	t.Logf("Found %d total records across %d record types", len(dnsResult.Records), len(recordTypesSeen))
}

func TestDNSTool_ErrorHandling_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create real network client for integration testing
	config := &domain.NetworkConfig{
		Timeout:        2 * time.Second, // Short timeout to trigger timeouts
		MaxConcurrency: 3,
		RetryAttempts:  1,
		RetryDelay:     100 * time.Millisecond,
	}
	
	mockLogger := &MockLogger{}
	client := network.NewClient(config, nil, mockLogger)
	tool := NewTool(client, mockLogger)
	
	tests := []struct {
		name        string
		domainName  string
		expectError bool
		errorType   domain.ErrorType
	}{
		{
			name:        "invalid domain format",
			domainName:  "invalid..domain",
			expectError: true,
			errorType:   domain.ErrorTypeValidation,
		},
		{
			name:        "nonexistent domain",
			domainName:  "this-domain-definitely-does-not-exist-12345.com",
			expectError: true,
			errorType:   domain.ErrorTypeNetwork,
		},
		{
			name:        "empty domain",
			domainName:  "",
			expectError: true,
			errorType:   domain.ErrorTypeValidation,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := domain.NewDNSParameters(tt.domainName, domain.DNSRecordTypeA)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			result, err := tool.Execute(ctx, params)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				
				// Verify error type if specified
				if netErr, ok := err.(*domain.NetTraceError); ok {
					if netErr.Type != tt.errorType {
						t.Errorf("Expected error type %v, got %v", tt.errorType, netErr.Type)
					}
				}
				
				if result != nil {
					t.Error("Expected nil result on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				
				if result == nil {
					t.Error("Expected result but got nil")
				}
			}
		})
	}
}

func TestDNSTool_Performance_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create real network client for performance testing
	config := &domain.NetworkConfig{
		Timeout:        10 * time.Second,
		MaxConcurrency: 5,
		RetryAttempts:  1,
		RetryDelay:     100 * time.Millisecond,
	}
	
	mockLogger := &MockLogger{}
	client := network.NewClient(config, nil, mockLogger)
	tool := NewTool(client, mockLogger)
	
	// Test performance with multiple domains
	domains := []string{
		"google.com",
		"github.com",
		"stackoverflow.com",
		"cloudflare.com",
	}
	
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
	}
	
	startTime := time.Now()
	
	for _, domainName := range domains {
		params := domain.NewDNSParameters(domainName, domain.DNSRecordTypeA)
		params.Set("record_types", recordTypes)
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		result, err := tool.Execute(ctx, params)
		cancel()
		
		if err != nil {
			t.Errorf("DNS lookup failed for %s: %v", domainName, err)
			continue
		}
		
		if result == nil {
			t.Errorf("No result for domain %s", domainName)
			continue
		}
		
		dnsResult, ok := result.Data().(domain.DNSResult)
		if !ok {
			t.Errorf("Invalid result type for domain %s", domainName)
			continue
		}
		
		if len(dnsResult.Records) == 0 {
			t.Errorf("No records found for domain %s", domainName)
		}
	}
	
	totalDuration := time.Since(startTime)
	averageDuration := totalDuration / time.Duration(len(domains))
	
	t.Logf("Performance test completed:")
	t.Logf("  Total time: %v", totalDuration)
	t.Logf("  Average per domain: %v", averageDuration)
	t.Logf("  Domains tested: %d", len(domains))
	t.Logf("  Record types per domain: %d", len(recordTypes))
	
	// Performance expectations (these are rough guidelines)
	maxAverageTime := 3 * time.Second
	if averageDuration > maxAverageTime {
		t.Errorf("Average lookup time too high: %v (expected less than %v)", averageDuration, maxAverageTime)
	}
}

func TestDNSTool_RealWorldDomains_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create real network client for real-world testing
	config := &domain.NetworkConfig{
		Timeout:        10 * time.Second,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     1 * time.Second,
	}
	
	mockLogger := &MockLogger{}
	client := network.NewClient(config, nil, mockLogger)
	tool := NewTool(client, mockLogger)
	
	// Test with real-world domains that should have various record types
	testCases := []struct {
		domainName  string
		recordType  domain.DNSRecordType
		expectValue bool
		description string
	}{
		{
			domainName:  "google.com",
			recordType:  domain.DNSRecordTypeA,
			expectValue: true,
			description: "Google should have A records",
		},
		{
			domainName:  "google.com",
			recordType:  domain.DNSRecordTypeAAAA,
			expectValue: true,
			description: "Google should have AAAA records",
		},
		{
			domainName:  "google.com",
			recordType:  domain.DNSRecordTypeMX,
			expectValue: true,
			description: "Google should have MX records",
		},
		{
			domainName:  "google.com",
			recordType:  domain.DNSRecordTypeNS,
			expectValue: true,
			description: "Google should have NS records",
		},
		{
			domainName:  "github.com",
			recordType:  domain.DNSRecordTypeA,
			expectValue: true,
			description: "GitHub should have A records",
		},
		{
			domainName:  "cloudflare.com",
			recordType:  domain.DNSRecordTypeTXT,
			expectValue: true,
			description: "Cloudflare should have TXT records",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			params := domain.NewDNSParameters(tc.domainName, tc.recordType)
			params.Set("record_types", []domain.DNSRecordType{tc.recordType})
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			result, err := tool.Execute(ctx, params)
			
			if err != nil {
				t.Errorf("DNS lookup failed: %v", err)
				return
			}
			
			if result == nil {
				t.Error("Expected result but got nil")
				return
			}
			
			dnsResult, ok := result.Data().(domain.DNSResult)
			if !ok {
				t.Error("Expected DNSResult data type")
				return
			}
			
			if tc.expectValue {
				if len(dnsResult.Records) == 0 {
					t.Errorf("Expected records for %s %s but got none", tc.domainName, GetRecordTypeString(tc.recordType))
					return
				}
				
				// Verify all records are of the expected type
				for _, record := range dnsResult.Records {
					if record.Type != tc.recordType {
						t.Errorf("Expected record type %v but got %v", tc.recordType, record.Type)
					}
					
					if record.Value == "" {
						t.Error("Record value should not be empty")
					}
					
					if record.TTL == 0 {
						t.Error("Record TTL should not be zero")
					}
				}
				
				t.Logf("Successfully found %d %s records for %s", 
					len(dnsResult.Records), GetRecordTypeString(tc.recordType), tc.domainName)
			}
		})
	}
}