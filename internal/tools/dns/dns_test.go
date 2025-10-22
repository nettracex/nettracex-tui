// Package dns provides DNS diagnostic functionality tests
package dns

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)

// MockLogger implements domain.Logger for testing
type MockLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  []interface{}
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "DEBUG", Message: msg, Fields: fields})
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "INFO", Message: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "WARN", Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "ERROR", Message: msg, Fields: fields})
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "FATAL", Message: msg, Fields: fields})
}

func (m *MockLogger) GetLogs() []LogEntry {
	return m.logs
}

func (m *MockLogger) ClearLogs() {
	m.logs = []LogEntry{}
}

func TestNewTool(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	
	if tool == nil {
		t.Fatal("NewTool returned nil")
	}
	
	if tool.Name() != "dns" {
		t.Errorf("Expected tool name 'dns', got '%s'", tool.Name())
	}
	
	if tool.Description() == "" {
		t.Error("Tool description should not be empty")
	}
}

func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	
	expected := "dns"
	actual := tool.Name()
	
	if actual != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, actual)
	}
}

func TestTool_Description(t *testing.T) {
	tool := &Tool{}
	
	description := tool.Description()
	
	if description == "" {
		t.Error("Description should not be empty")
	}
	
	// Check that description mentions key features
	expectedKeywords := []string{"DNS", "multiple", "record types", "concurrent"}
	for _, keyword := range expectedKeywords {
		if !containsIgnoreCase(description, keyword) {
			t.Errorf("Description should contain '%s': %s", keyword, description)
		}
	}
}

func TestTool_Validate(t *testing.T) {
	tool := &Tool{}
	
	tests := []struct {
		name        string
		params      domain.Parameters
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing domain parameter",
			params:      domain.NewParameters(),
			expectError: true,
			errorMsg:    "domain parameter is required",
		},
		{
			name: "empty domain parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "")
				return p
			}(),
			expectError: true,
			errorMsg:    "domain parameter cannot be empty",
		},
		{
			name: "invalid domain parameter type",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", 123)
				return p
			}(),
			expectError: true,
			errorMsg:    "domain parameter must be a string",
		},
		{
			name: "invalid domain format",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "invalid..domain")
				return p
			}(),
			expectError: true,
			errorMsg:    "domain must be a valid domain name",
		},
		{
			name: "valid domain",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "example.com")
				return p
			}(),
			expectError: false,
		},
		{
			name: "valid domain with subdomain",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "www.example.com")
				return p
			}(),
			expectError: false,
		},
		{
			name: "localhost domain",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "localhost")
				return p
			}(),
			expectError: false,
		},
		{
			name: "valid record types",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "example.com")
				p.Set("record_types", []domain.DNSRecordType{
					domain.DNSRecordTypeA,
					domain.DNSRecordTypeAAAA,
				})
				return p
			}(),
			expectError: false,
		},
		{
			name: "invalid record types parameter type",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "example.com")
				p.Set("record_types", "invalid")
				return p
			}(),
			expectError: true,
			errorMsg:    "record_types parameter must be a slice of DNSRecordType",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.Validate(tt.params)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsIgnoreCase(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestTool_Execute(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	
	// Set up mock responses
	mockClient.SetDNSResponse("example.com", domain.DNSRecordTypeA, domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeA,
		Records: []domain.DNSRecord{
			{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeA,
				Value: "93.184.216.34",
				TTL:   300,
			},
		},
		ResponseTime: 50 * time.Millisecond,
		Server:       "system",
	})
	
	mockClient.SetDNSResponse("example.com", domain.DNSRecordTypeAAAA, domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeAAAA,
		Records: []domain.DNSRecord{
			{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeAAAA,
				Value: "2606:2800:220:1:248:1893:25c8:1946",
				TTL:   300,
			},
		},
		ResponseTime: 45 * time.Millisecond,
		Server:       "system",
	})
	
	tests := []struct {
		name        string
		params      domain.Parameters
		expectError bool
		errorCode   string
	}{
		{
			name: "successful DNS lookup",
			params: func() domain.Parameters {
				p := domain.NewDNSParameters("example.com", domain.DNSRecordTypeA)
				return p
			}(),
			expectError: false,
		},
		{
			name: "successful DNS lookup with multiple record types",
			params: func() domain.Parameters {
				p := domain.NewDNSParameters("example.com", domain.DNSRecordTypeA)
				p.Set("record_types", []domain.DNSRecordType{
					domain.DNSRecordTypeA,
					domain.DNSRecordTypeAAAA,
				})
				return p
			}(),
			expectError: false,
		},
		{
			name: "invalid parameters",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("domain", "")
				return p
			}(),
			expectError: true,
			errorCode:   "DNS_VALIDATION_FAILED",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := tool.Execute(ctx, tt.params)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					if netErr, ok := err.(*domain.NetTraceError); ok {
						if tt.errorCode != "" && netErr.Code != tt.errorCode {
							t.Errorf("Expected error code '%s', got '%s'", tt.errorCode, netErr.Code)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				
				if result == nil {
					t.Error("Expected result but got nil")
				} else {
					// Verify result metadata
					metadata := result.Metadata()
					if metadata["tool"] != "dns" {
						t.Errorf("Expected tool metadata 'dns', got '%v'", metadata["tool"])
					}
					
					// Verify result data
					dnsResult, ok := result.Data().(domain.DNSResult)
					if !ok {
						t.Error("Expected DNSResult data type")
					} else {
						if dnsResult.Query == "" {
							t.Error("Expected non-empty query in result")
						}
						if len(dnsResult.Records) == 0 {
							t.Error("Expected at least one DNS record in result")
						}
					}
				}
			}
		})
	}
}

func TestTool_GetModel(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	
	model := tool.GetModel()
	
	if model == nil {
		t.Error("GetModel returned nil")
	}
	
	// Verify it's the correct type
	if _, ok := model.(*Model); !ok {
		t.Error("GetModel should return *Model type")
	}
}

func TestIsValidDomain(t *testing.T) {
	tool := &Tool{}
	
	tests := []struct {
		domain string
		valid  bool
	}{
		{"example.com", true},
		{"www.example.com", true},
		{"sub.domain.example.com", true},
		{"localhost", true},
		{"test-domain.com", true},
		{"123.example.com", true},
		{"", false},
		{".", false},
		{".com", false},
		{"example.", false},
		{"example..com", false},
		{"-example.com", false},
		{"example-.com", false},
		{"example.com-", false},
		{"very-long-domain-name-that-exceeds-the-maximum-allowed-length-for-a-single-label-which-is-sixty-three-characters.com", false},
		{string(make([]byte, 254)), false}, // Too long domain
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("domain_%s", tt.domain), func(t *testing.T) {
			result := tool.isValidDomain(tt.domain)
			if result != tt.valid {
				t.Errorf("isValidDomain(%q) = %v, want %v", tt.domain, result, tt.valid)
			}
		})
	}
}

func TestIsValidRecordType(t *testing.T) {
	tool := &Tool{}
	
	tests := []struct {
		recordType domain.DNSRecordType
		valid      bool
	}{
		{domain.DNSRecordTypeA, true},
		{domain.DNSRecordTypeAAAA, true},
		{domain.DNSRecordTypeMX, true},
		{domain.DNSRecordTypeTXT, true},
		{domain.DNSRecordTypeCNAME, true},
		{domain.DNSRecordTypeNS, true},
		{domain.DNSRecordTypeSOA, false}, // Not supported in this implementation
		{domain.DNSRecordTypePTR, false}, // Not supported in this implementation
		{domain.DNSRecordType(999), false}, // Invalid type
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("type_%d", tt.recordType), func(t *testing.T) {
			result := tool.isValidRecordType(tt.recordType)
			if result != tt.valid {
				t.Errorf("isValidRecordType(%v) = %v, want %v", tt.recordType, result, tt.valid)
			}
		})
	}
}

func TestGetRecordTypes(t *testing.T) {
	tool := &Tool{}
	
	tests := []struct {
		name           string
		params         domain.Parameters
		expectedLength int
		expectedTypes  []domain.DNSRecordType
	}{
		{
			name:           "default record types",
			params:         domain.NewParameters(),
			expectedLength: 6,
			expectedTypes: []domain.DNSRecordType{
				domain.DNSRecordTypeA,
				domain.DNSRecordTypeAAAA,
				domain.DNSRecordTypeMX,
				domain.DNSRecordTypeTXT,
				domain.DNSRecordTypeCNAME,
				domain.DNSRecordTypeNS,
			},
		},
		{
			name: "custom record types",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("record_types", []domain.DNSRecordType{
					domain.DNSRecordTypeA,
					domain.DNSRecordTypeMX,
				})
				return p
			}(),
			expectedLength: 2,
			expectedTypes: []domain.DNSRecordType{
				domain.DNSRecordTypeA,
				domain.DNSRecordTypeMX,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.getRecordTypes(tt.params)
			
			if len(result) != tt.expectedLength {
				t.Errorf("Expected %d record types, got %d", tt.expectedLength, len(result))
			}
			
			if tt.expectedTypes != nil {
				for _, expectedType := range tt.expectedTypes {
					found := false
					for _, actualType := range result {
						if actualType == expectedType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected record type %v not found in result", expectedType)
					}
				}
			}
		})
	}
}

func TestPerformConcurrentLookups(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	
	// Set up mock responses
	mockClient.SetDNSResponse("example.com", domain.DNSRecordTypeA, domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeA,
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
		},
		ResponseTime: 50 * time.Millisecond,
		Server:       "system",
	})
	
	mockClient.SetDNSResponse("example.com", domain.DNSRecordTypeAAAA, domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeAAAA,
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeAAAA, Value: "2606:2800:220:1:248:1893:25c8:1946", TTL: 300},
		},
		ResponseTime: 45 * time.Millisecond,
		Server:       "system",
	})
	
	ctx := context.Background()
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
	}
	
	results, err := tool.performConcurrentLookups(ctx, "example.com", recordTypes)
	
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
	
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	
	// Verify A record result
	if aResult, exists := results[domain.DNSRecordTypeA]; exists {
		if len(aResult.Records) != 1 {
			t.Errorf("Expected 1 A record, got %d", len(aResult.Records))
		}
		if aResult.Records[0].Value != "93.184.216.34" {
			t.Errorf("Expected A record value '93.184.216.34', got '%s'", aResult.Records[0].Value)
		}
	} else {
		t.Error("Expected A record result not found")
	}
	
	// Verify AAAA record result
	if aaaaResult, exists := results[domain.DNSRecordTypeAAAA]; exists {
		if len(aaaaResult.Records) != 1 {
			t.Errorf("Expected 1 AAAA record, got %d", len(aaaaResult.Records))
		}
		if aaaaResult.Records[0].Value != "2606:2800:220:1:248:1893:25c8:1946" {
			t.Errorf("Expected AAAA record value '2606:2800:220:1:248:1893:25c8:1946', got '%s'", aaaaResult.Records[0].Value)
		}
	} else {
		t.Error("Expected AAAA record result not found")
	}
}

func TestConsolidateResults(t *testing.T) {
	tool := &Tool{}
	
	results := map[domain.DNSRecordType]domain.DNSResult{
		domain.DNSRecordTypeA: {
			Query:      "example.com",
			RecordType: domain.DNSRecordTypeA,
			Records: []domain.DNSRecord{
				{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
			},
			ResponseTime: 50 * time.Millisecond,
			Server:       "system",
		},
		domain.DNSRecordTypeAAAA: {
			Query:      "example.com",
			RecordType: domain.DNSRecordTypeAAAA,
			Records: []domain.DNSRecord{
				{Name: "example.com", Type: domain.DNSRecordTypeAAAA, Value: "2606:2800:220:1:248:1893:25c8:1946", TTL: 300},
			},
			ResponseTime: 45 * time.Millisecond,
			Server:       "system",
		},
	}
	
	consolidated := tool.consolidateResults("example.com", results)
	
	if consolidated.Query != "example.com" {
		t.Errorf("Expected query 'example.com', got '%s'", consolidated.Query)
	}
	
	if len(consolidated.Records) != 2 {
		t.Errorf("Expected 2 consolidated records, got %d", len(consolidated.Records))
	}
	
	// Verify average response time calculation (allow for small precision differences)
	expectedAvgTime := (50 + 45) / 2 * time.Millisecond
	if consolidated.ResponseTime < expectedAvgTime-time.Millisecond || consolidated.ResponseTime > expectedAvgTime+time.Millisecond {
		t.Errorf("Expected average response time around %v, got %v", expectedAvgTime, consolidated.ResponseTime)
	}
}

func TestGetRecordTypeString(t *testing.T) {
	tests := []struct {
		recordType domain.DNSRecordType
		expected   string
	}{
		{domain.DNSRecordTypeA, "A"},
		{domain.DNSRecordTypeAAAA, "AAAA"},
		{domain.DNSRecordTypeMX, "MX"},
		{domain.DNSRecordTypeTXT, "TXT"},
		{domain.DNSRecordTypeCNAME, "CNAME"},
		{domain.DNSRecordTypeNS, "NS"},
		{domain.DNSRecordTypeSOA, "SOA"},
		{domain.DNSRecordTypePTR, "PTR"},
		{domain.DNSRecordType(999), "UNKNOWN(999)"},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("type_%d", tt.recordType), func(t *testing.T) {
			result := GetRecordTypeString(tt.recordType)
			if result != tt.expected {
				t.Errorf("GetRecordTypeString(%v) = %s, want %s", tt.recordType, result, tt.expected)
			}
		})
	}
}

func TestParseRecordTypeString(t *testing.T) {
	tests := []struct {
		input       string
		expected    domain.DNSRecordType
		expectError bool
	}{
		{"A", domain.DNSRecordTypeA, false},
		{"a", domain.DNSRecordTypeA, false},
		{"AAAA", domain.DNSRecordTypeAAAA, false},
		{"aaaa", domain.DNSRecordTypeAAAA, false},
		{"MX", domain.DNSRecordTypeMX, false},
		{"mx", domain.DNSRecordTypeMX, false},
		{"TXT", domain.DNSRecordTypeTXT, false},
		{"txt", domain.DNSRecordTypeTXT, false},
		{"CNAME", domain.DNSRecordTypeCNAME, false},
		{"cname", domain.DNSRecordTypeCNAME, false},
		{"NS", domain.DNSRecordTypeNS, false},
		{"ns", domain.DNSRecordTypeNS, false},
		{"SOA", domain.DNSRecordTypeSOA, false},
		{"soa", domain.DNSRecordTypeSOA, false},
		{"PTR", domain.DNSRecordTypePTR, false},
		{"ptr", domain.DNSRecordTypePTR, false},
		{"INVALID", domain.DNSRecordTypeA, true},
		{"", domain.DNSRecordTypeA, true},
		{"123", domain.DNSRecordTypeA, true},
	}
	
	for _, tt := range tests {
		t.Run(fmt.Sprintf("input_%s", tt.input), func(t *testing.T) {
			result, err := ParseRecordTypeString(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("ParseRecordTypeString(%s) = %v, want %v", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestValidateDNSResult(t *testing.T) {
	tests := []struct {
		name        string
		result      domain.DNSResult
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid DNS result",
			result: domain.DNSResult{
				Query:        "example.com",
				ResponseTime: 50 * time.Millisecond,
				Records: []domain.DNSRecord{
					{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
				},
			},
			expectError: false,
		},
		{
			name: "missing query",
			result: domain.DNSResult{
				ResponseTime: 50 * time.Millisecond,
				Records: []domain.DNSRecord{
					{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
				},
			},
			expectError: true,
			errorMsg:    "DNS result missing query",
		},
		{
			name: "invalid response time",
			result: domain.DNSResult{
				Query:        "example.com",
				ResponseTime: 0,
				Records: []domain.DNSRecord{
					{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
				},
			},
			expectError: true,
			errorMsg:    "DNS result has invalid response time",
		},
		{
			name: "no records",
			result: domain.DNSResult{
				Query:        "example.com",
				ResponseTime: 50 * time.Millisecond,
				Records:      []domain.DNSRecord{},
				Authority:    []domain.DNSRecord{},
				Additional:   []domain.DNSRecord{},
			},
			expectError: true,
			errorMsg:    "DNS result contains no records",
		},
		{
			name: "invalid DNS record",
			result: domain.DNSResult{
				Query:        "example.com",
				ResponseTime: 50 * time.Millisecond,
				Records: []domain.DNSRecord{
					{Name: "", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
				},
			},
			expectError: true,
			errorMsg:    "invalid DNS record",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSResult(tt.result)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsIgnoreCase(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateDNSRecord(t *testing.T) {
	tests := []struct {
		name        string
		record      domain.DNSRecord
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid DNS record",
			record: domain.DNSRecord{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeA,
				Value: "93.184.216.34",
				TTL:   300,
			},
			expectError: false,
		},
		{
			name: "missing name",
			record: domain.DNSRecord{
				Name:  "",
				Type:  domain.DNSRecordTypeA,
				Value: "93.184.216.34",
				TTL:   300,
			},
			expectError: true,
			errorMsg:    "DNS record missing name",
		},
		{
			name: "missing value",
			record: domain.DNSRecord{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeA,
				Value: "",
				TTL:   300,
			},
			expectError: true,
			errorMsg:    "DNS record missing value",
		},
		{
			name: "zero TTL",
			record: domain.DNSRecord{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeA,
				Value: "93.184.216.34",
				TTL:   0,
			},
			expectError: true,
			errorMsg:    "DNS record has zero TTL",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDNSRecord(tt.record)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !containsIgnoreCase(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains another string (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}