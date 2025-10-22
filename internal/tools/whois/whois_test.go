// Package whois provides tests for WHOIS diagnostic functionality
package whois

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNetworkClient implements domain.NetworkClient for testing
type MockNetworkClient struct {
	mock.Mock
}

func (m *MockNetworkClient) Ping(ctx context.Context, host string, opts domain.PingOptions) (<-chan domain.PingResult, error) {
	args := m.Called(ctx, host, opts)
	return args.Get(0).(<-chan domain.PingResult), args.Error(1)
}

func (m *MockNetworkClient) Traceroute(ctx context.Context, host string, opts domain.TraceOptions) (<-chan domain.TraceHop, error) {
	args := m.Called(ctx, host, opts)
	return args.Get(0).(<-chan domain.TraceHop), args.Error(1)
}

func (m *MockNetworkClient) DNSLookup(ctx context.Context, domainName string, recordType domain.DNSRecordType) (domain.DNSResult, error) {
	args := m.Called(ctx, domainName, recordType)
	return args.Get(0).(domain.DNSResult), args.Error(1)
}

func (m *MockNetworkClient) WHOISLookup(ctx context.Context, query string) (domain.WHOISResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(domain.WHOISResult), args.Error(1)
}

func (m *MockNetworkClient) SSLCheck(ctx context.Context, host string, port int) (domain.SSLResult, error) {
	args := m.Called(ctx, host, port)
	return args.Get(0).(domain.SSLResult), args.Error(1)
}

// MockLogger implements domain.Logger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func TestNewTool(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}

	tool := NewTool(mockClient, mockLogger)

	assert.NotNil(t, tool)
	assert.Equal(t, "whois", tool.Name())
	assert.Contains(t, tool.Description(), "WHOIS")
	assert.Equal(t, mockClient, tool.client)
	assert.Equal(t, mockLogger, tool.logger)
}

func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	assert.Equal(t, "whois", tool.Name())
}

func TestTool_Description(t *testing.T) {
	tool := &Tool{}
	description := tool.Description()
	assert.Contains(t, description, "WHOIS")
	assert.Contains(t, description, "domain")
	assert.Contains(t, description, "IP")
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
			name:        "missing query parameter",
			params:      domain.NewParameters(),
			expectError: true,
			errorMsg:    "query parameter is required",
		},
		{
			name: "empty query parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", "")
				return p
			}(),
			expectError: true,
			errorMsg:    "query parameter cannot be empty",
		},
		{
			name: "whitespace only query",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", "   ")
				return p
			}(),
			expectError: true,
			errorMsg:    "query parameter cannot be empty",
		},
		{
			name: "non-string query parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", 123)
				return p
			}(),
			expectError: true,
			errorMsg:    "query parameter must be a string",
		},
		{
			name: "invalid domain format",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", "invalid..domain")
				return p
			}(),
			expectError: true,
			errorMsg:    "query must be a valid domain name or IP address",
		},
		{
			name: "valid domain",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", "example.com")
				return p
			}(),
			expectError: false,
		},
		{
			name: "valid IPv4 address",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", "8.8.8.8")
				return p
			}(),
			expectError: false,
		},
		{
			name: "valid IPv6 address",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("query", "2001:4860:4860::8888")
				return p
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.Validate(tt.params)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTool_Execute_Success(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Setup expected WHOIS result
	expectedResult := domain.WHOISResult{
		Domain:      "example.com",
		Registrar:   "Example Registrar",
		Created:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Expires:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
		Status:      []string{"clientTransferProhibited"},
		RawData:     "Domain Name: EXAMPLE.COM\nRegistrar: Example Registrar\n",
	}

	// Setup mock expectations
	mockLogger.On("Info", "Executing WHOIS lookup", mock.Anything).Return()
	mockLogger.On("Info", "WHOIS lookup completed successfully", mock.Anything, mock.Anything, mock.Anything).Return()
	mockClient.On("WHOISLookup", mock.Anything, "example.com").Return(expectedResult, nil)

	// Create parameters
	params := domain.NewWHOISParameters("example.com")

	// Execute
	result, err := tool.Execute(context.Background(), params)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)

	whoisResult, ok := result.Data().(domain.WHOISResult)
	assert.True(t, ok)
	assert.Equal(t, expectedResult.Domain, whoisResult.Domain)
	assert.Equal(t, expectedResult.Registrar, whoisResult.Registrar)
	assert.Equal(t, expectedResult.Created, whoisResult.Created)

	// Check metadata
	metadata := result.Metadata()
	assert.Equal(t, "whois", metadata["tool"])
	assert.Equal(t, "example.com", metadata["query"])
	assert.Equal(t, "domain", metadata["query_type"])

	mockClient.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestTool_Execute_ValidationError(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Setup mock expectations
	mockLogger.On("Info", "Executing WHOIS lookup", mock.Anything).Return()

	// Create invalid parameters
	params := domain.NewParameters()

	// Execute
	result, err := tool.Execute(context.Background(), params)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)

	netErr, ok := err.(*domain.NetTraceError)
	assert.True(t, ok)
	assert.Equal(t, domain.ErrorTypeValidation, netErr.Type)
	assert.Equal(t, "WHOIS_VALIDATION_FAILED", netErr.Code)

	mockLogger.AssertExpectations(t)
}

func TestTool_Execute_NetworkError(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Setup mock expectations
	mockLogger.On("Info", "Executing WHOIS lookup", mock.Anything).Return()
	mockClient.On("WHOISLookup", mock.Anything, "example.com").Return(domain.WHOISResult{}, errors.New("network error"))

	// Create parameters
	params := domain.NewWHOISParameters("example.com")

	// Execute
	result, err := tool.Execute(context.Background(), params)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, result)

	netErr, ok := err.(*domain.NetTraceError)
	assert.True(t, ok)
	assert.Equal(t, domain.ErrorTypeNetwork, netErr.Type)
	assert.Equal(t, "WHOIS_LOOKUP_FAILED", netErr.Code)

	mockClient.AssertExpectations(t)
	mockLogger.AssertExpectations(t)
}

func TestTool_isValidQuery(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"valid domain", "example.com", true},
		{"valid subdomain", "sub.example.com", true},
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:4860:4860::8888", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"invalid domain", "invalid..domain", false},
		{"domain too long", strings.Repeat("a", 254), false},
		{"single character", "a", false},
		{"no TLD", "example", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isValidQuery(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTool_isValidDomain(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"valid domain", "example.com", true},
		{"valid subdomain", "sub.example.com", true},
		{"valid with numbers", "test123.com", true},
		{"valid with hyphens", "test-site.com", true},
		{"empty string", "", false},
		{"too long", strings.Repeat("a", 254), false},
		{"starts with hyphen", "-example.com", false},
		{"ends with hyphen", "example-.com", false},
		{"double dots", "example..com", false},
		{"starts with dot", ".example.com", false},
		{"ends with dot", "example.com.", false},
		{"invalid characters", "example@.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.isValidDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTool_determineQueryType(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{"IPv4 address", "192.168.1.1", "ip"},
		{"IPv6 address", "2001:4860:4860::8888", "ip"},
		{"domain name", "example.com", "domain"},
		{"subdomain", "sub.example.com", "domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.determineQueryType(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseWHOISData(t *testing.T) {
	rawData := `Domain Name: EXAMPLE.COM
Registrar: Example Registrar Inc.
Creation Date: 2020-01-01T00:00:00Z
Updated Date: 2023-06-15T12:30:00Z
Expiry Date: 2025-01-01T00:00:00Z
Name Server: ns1.example.com
Name Server: ns2.example.com
Status: clientTransferProhibited
Registrant Name: John Doe
Registrant Organization: Example Corp
Registrant Email: admin@example.com
Admin Name: Jane Smith
Admin Email: jane@example.com
Tech Name: Bob Johnson
Tech Email: bob@example.com`

	result := ParseWHOISData(rawData, "example.com")

	assert.Equal(t, "EXAMPLE.COM", result.Domain)
	assert.Equal(t, "Example Registrar Inc.", result.Registrar)
	assert.Equal(t, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), result.Created)
	assert.Equal(t, time.Date(2023, 6, 15, 12, 30, 0, 0, time.UTC), result.Updated)
	assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), result.Expires)
	assert.Contains(t, result.NameServers, "ns1.example.com")
	assert.Contains(t, result.NameServers, "ns2.example.com")
	assert.Contains(t, result.Status, "clientTransferProhibited")
	assert.Equal(t, "John Doe", result.Contacts["registrant"].Name)
	assert.Equal(t, "Example Corp", result.Contacts["registrant"].Organization)
	assert.Equal(t, "admin@example.com", result.Contacts["registrant"].Email)
	assert.Equal(t, "Jane Smith", result.Contacts["admin"].Name)
	assert.Equal(t, "Bob Johnson", result.Contacts["tech"].Name)
	assert.Equal(t, rawData, result.RawData)
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		expectError bool
		expected    time.Time
	}{
		{
			name:        "ISO 8601 format",
			dateStr:     "2020-01-01T00:00:00Z",
			expectError: false,
			expected:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "simple date format",
			dateStr:     "2020-01-01",
			expectError: false,
			expected:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "date with time",
			dateStr:     "2020-01-01 15:30:45",
			expectError: false,
			expected:    time.Date(2020, 1, 1, 15, 30, 45, 0, time.UTC),
		},
		{
			name:        "invalid date format",
			dateStr:     "invalid-date",
			expectError: true,
		},
		{
			name:        "empty string",
			dateStr:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDate(tt.dateStr)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatWHOISResult(t *testing.T) {
	result := domain.WHOISResult{
		Domain:      "example.com",
		Registrar:   "Example Registrar",
		Created:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated:     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Expires:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
		Status:      []string{"clientTransferProhibited"},
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:         "John Doe",
				Organization: "Example Corp",
				Email:        "admin@example.com",
			},
		},
	}

	formatted := FormatWHOISResult(result)

	assert.Contains(t, formatted, "Domain: example.com")
	assert.Contains(t, formatted, "Registrar: Example Registrar")
	assert.Contains(t, formatted, "Created: 2020-01-01")
	assert.Contains(t, formatted, "Expires: 2025-01-01")
	assert.Contains(t, formatted, "ns1.example.com")
	assert.Contains(t, formatted, "ns2.example.com")
	assert.Contains(t, formatted, "clientTransferProhibited")
	assert.Contains(t, formatted, "John Doe")
	assert.Contains(t, formatted, "Example Corp")
}

func TestFormatWHOISResult_ExpirationWarning(t *testing.T) {
	// Test domain expiring soon
	soonExpiry := domain.WHOISResult{
		Domain:  "example.com",
		Expires: time.Now().AddDate(0, 0, 15), // 15 days from now
	}

	formatted := FormatWHOISResult(soonExpiry)
	assert.Contains(t, formatted, "WARNING: Domain expires in")

	// Test expired domain
	expiredDomain := domain.WHOISResult{
		Domain:  "example.com",
		Expires: time.Now().AddDate(0, 0, -1), // 1 day ago
	}

	formatted = FormatWHOISResult(expiredDomain)
	assert.Contains(t, formatted, "WARNING: Domain has expired!")
}

func TestValidateWHOISResult(t *testing.T) {
	tests := []struct {
		name        string
		result      domain.WHOISResult
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid result",
			result: domain.WHOISResult{
				Domain:    "example.com",
				RawData:   "Domain Name: EXAMPLE.COM",
				Registrar: "Example Registrar",
			},
			expectError: false,
		},
		{
			name: "missing domain",
			result: domain.WHOISResult{
				RawData: "Some data",
			},
			expectError: true,
			errorMsg:    "missing domain name",
		},
		{
			name: "missing raw data",
			result: domain.WHOISResult{
				Domain: "example.com",
			},
			expectError: true,
			errorMsg:    "missing raw data",
		},
		{
			name: "no meaningful data",
			result: domain.WHOISResult{
				Domain:  "example.com",
				RawData: "Domain Name: EXAMPLE.COM",
			},
			expectError: true,
			errorMsg:    "no meaningful data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWHOISResult(tt.result)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}