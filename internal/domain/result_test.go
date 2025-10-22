package domain

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockOutputFormatter for testing
type MockOutputFormatter struct {
	formatFunc func(data interface{}) string
}

func (m *MockOutputFormatter) Format(data interface{}) string {
	if m.formatFunc != nil {
		return m.formatFunc(data)
	}
	return "formatted data"
}

func (m *MockOutputFormatter) SetOptions(options map[string]interface{}) {
	// Mock implementation
}

func TestBaseResult(t *testing.T) {
	testData := map[string]interface{}{
		"test": "data",
		"number": 42,
	}
	
	result := NewResult(testData)
	
	// Test Data method
	assert.Equal(t, testData, result.Data())
	
	// Test Metadata methods
	assert.NotNil(t, result.Metadata())
	assert.Empty(t, result.Metadata())
	
	result.SetMetadata("timestamp", time.Now())
	result.SetMetadata("source", "test")
	
	metadata := result.Metadata()
	assert.Len(t, metadata, 2)
	assert.Contains(t, metadata, "timestamp")
	assert.Contains(t, metadata, "source")
	assert.Equal(t, "test", metadata["source"])
}

func TestBaseResultFormat(t *testing.T) {
	testData := "test data"
	result := NewResult(testData)
	
	formatter := &MockOutputFormatter{
		formatFunc: func(data interface{}) string {
			return "custom formatted: " + data.(string)
		},
	}
	
	formatted := result.Format(formatter)
	assert.Equal(t, "custom formatted: test data", formatted)
}

func TestBaseResultExportJSON(t *testing.T) {
	testData := map[string]interface{}{
		"host": "example.com",
		"port": 80,
	}
	
	result := NewResult(testData)
	result.SetMetadata("test_run", "unit_test")
	
	exported, err := result.Export(ExportFormatJSON)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	// Verify JSON structure
	var exportedData map[string]interface{}
	err = json.Unmarshal(exported, &exportedData)
	assert.NoError(t, err)
	
	assert.Contains(t, exportedData, "data")
	assert.Contains(t, exportedData, "metadata")
	assert.Contains(t, exportedData, "timestamp")
	
	data := exportedData["data"].(map[string]interface{})
	assert.Equal(t, "example.com", data["host"])
	assert.Equal(t, float64(80), data["port"]) // JSON numbers are float64
	
	metadata := exportedData["metadata"].(map[string]interface{})
	assert.Equal(t, "unit_test", metadata["test_run"])
}

func TestBaseResultExportText(t *testing.T) {
	testData := "simple test data"
	result := NewResult(testData)
	result.SetMetadata("source", "unit_test")
	
	exported, err := result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	assert.Contains(t, exportedStr, "=== NetTraceX Result ===")
	assert.Contains(t, exportedStr, "source: unit_test")
	assert.Contains(t, exportedStr, "=== Data ===")
	assert.Contains(t, exportedStr, "simple test data")
}

func TestBaseResultExportPingResults(t *testing.T) {
	now := time.Now()
	pingResults := []PingResult{
		{
			Host: NetworkHost{
				Hostname:  "example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			Sequence:   1,
			RTT:        10 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  now,
		},
		{
			Host: NetworkHost{
				Hostname:  "example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			Sequence:   2,
			RTT:        12 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  now.Add(time.Second),
		},
	}
	
	result := NewResult(pingResults)
	
	// Test CSV export
	exported, err := result.Export(ExportFormatCSV)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	lines := strings.Split(exportedStr, "\n")
	assert.Contains(t, lines[0], "timestamp,host,sequence,rtt_ms,ttl,packet_size")
	assert.Contains(t, lines[1], "example.com,1,10.000,64,64")
	assert.Contains(t, lines[2], "example.com,2,12.000,64,64")
	
	// Test text export
	exported, err = result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr = string(exported)
	assert.Contains(t, exportedStr, "Ping example.com: seq=1 time=10ms ttl=64")
	assert.Contains(t, exportedStr, "Ping example.com: seq=2 time=12ms ttl=64")
}

func TestBaseResultExportTraceHops(t *testing.T) {
	traceHops := []TraceHop{
		{
			Number: 1,
			Host: NetworkHost{
				Hostname:  "gateway.local",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:     []time.Duration{5 * time.Millisecond, 6 * time.Millisecond, 5 * time.Millisecond},
			Timeout: false,
		},
		{
			Number: 2,
			Host: NetworkHost{
				Hostname:  "isp.gateway.com",
				IPAddress: net.ParseIP("10.0.0.1"),
			},
			RTT:     []time.Duration{15 * time.Millisecond, 16 * time.Millisecond},
			Timeout: false,
		},
	}
	
	result := NewResult(traceHops)
	
	// Test CSV export
	exported, err := result.Export(ExportFormatCSV)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	lines := strings.Split(exportedStr, "\n")
	assert.Contains(t, lines[0], "hop,hostname,ip_address,rtt1_ms,rtt2_ms,rtt3_ms,timeout")
	assert.Contains(t, lines[1], "1,gateway.local,192.168.1.1,5.000,6.000,5.000,false")
	assert.Contains(t, lines[2], "2,isp.gateway.com,10.0.0.1,15.000,16.000,,false")
	
	// Test text export
	exported, err = result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr = string(exported)
	assert.Contains(t, exportedStr, "Hop 1: gateway.local (192.168.1.1)")
	assert.Contains(t, exportedStr, "Hop 2: isp.gateway.com (10.0.0.1)")
}

func TestBaseResultExportDNSResult(t *testing.T) {
	dnsResult := DNSResult{
		Query:      "example.com",
		RecordType: DNSRecordTypeA,
		Records: []DNSRecord{
			{
				Name:  "example.com",
				Type:  DNSRecordTypeA,
				Value: "192.168.1.1",
				TTL:   300,
			},
			{
				Name:  "example.com",
				Type:  DNSRecordTypeA,
				Value: "192.168.1.2",
				TTL:   300,
			},
		},
		ResponseTime: 50 * time.Millisecond,
		Server:       "8.8.8.8",
	}
	
	result := NewResult(dnsResult)
	
	// Test CSV export
	exported, err := result.Export(ExportFormatCSV)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	lines := strings.Split(exportedStr, "\n")
	assert.Contains(t, lines[0], "name,type,value,ttl,priority")
	assert.Contains(t, lines[1], "example.com,0,192.168.1.1,300,0")
	assert.Contains(t, lines[2], "example.com,0,192.168.1.2,300,0")
	
	// Test text export
	exported, err = result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr = string(exported)
	assert.Contains(t, exportedStr, "DNS Query: example.com (Type: 0)")
	assert.Contains(t, exportedStr, "example.com 300 192.168.1.1")
	assert.Contains(t, exportedStr, "example.com 300 192.168.1.2")
}

func TestBaseResultExportWHOISResult(t *testing.T) {
	created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expires := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	whoisResult := WHOISResult{
		Domain:      "example.com",
		Registrar:   "Example Registrar",
		Created:     created,
		Updated:     updated,
		Expires:     expires,
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
	}
	
	result := NewResult(whoisResult)
	
	// Test CSV export
	exported, err := result.Export(ExportFormatCSV)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	assert.Contains(t, exportedStr, "field,value")
	assert.Contains(t, exportedStr, "domain,example.com")
	assert.Contains(t, exportedStr, "registrar,Example Registrar")
	assert.Contains(t, exportedStr, "nameserver,ns1.example.com")
	assert.Contains(t, exportedStr, "nameserver,ns2.example.com")
	
	// Test text export
	exported, err = result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr = string(exported)
	assert.Contains(t, exportedStr, "Domain: example.com")
	assert.Contains(t, exportedStr, "Registrar: Example Registrar")
	assert.Contains(t, exportedStr, "Created: 2020-01-01T00:00:00Z")
	assert.Contains(t, exportedStr, "Expires: 2025-01-01T00:00:00Z")
}

func TestBaseResultExportSSLResult(t *testing.T) {
	expiry := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	
	sslResult := SSLResult{
		Host:    "example.com",
		Port:    443,
		Subject: "CN=example.com",
		Issuer:  "Let's Encrypt Authority X3",
		Valid:   true,
		Expiry:  expiry,
		SANs:    []string{"example.com", "www.example.com"},
	}
	
	result := NewResult(sslResult)
	
	// Test CSV export
	exported, err := result.Export(ExportFormatCSV)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	assert.Contains(t, exportedStr, "field,value")
	assert.Contains(t, exportedStr, "host,example.com")
	assert.Contains(t, exportedStr, "port,443")
	assert.Contains(t, exportedStr, "subject,CN=example.com")
	assert.Contains(t, exportedStr, "valid,true")
	assert.Contains(t, exportedStr, "san,example.com")
	assert.Contains(t, exportedStr, "san,www.example.com")
	
	// Test text export
	exported, err = result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr = string(exported)
	assert.Contains(t, exportedStr, "SSL Certificate for example.com:443")
	assert.Contains(t, exportedStr, "Subject: CN=example.com")
	assert.Contains(t, exportedStr, "Issuer: Let's Encrypt Authority X3")
	assert.Contains(t, exportedStr, "Valid: true")
	assert.Contains(t, exportedStr, "Expires: 2025-12-31T23:59:59Z")
}

func TestBaseResultExportUnsupportedFormat(t *testing.T) {
	result := NewResult("test data")
	
	_, err := result.Export(ExportFormat(999))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestBaseResultExportUnknownDataType(t *testing.T) {
	// Test with a custom struct that doesn't have specific export handling
	type CustomData struct {
		Field1 string
		Field2 int
	}
	
	customData := CustomData{
		Field1: "test",
		Field2: 42,
	}
	
	result := NewResult(customData)
	
	// CSV export should fall back to JSON
	exported, err := result.Export(ExportFormatCSV)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr := string(exported)
	assert.Contains(t, exportedStr, "data")
	assert.Contains(t, exportedStr, "Field1")
	assert.Contains(t, exportedStr, "test")
	
	// Text export should use default formatting
	exported, err = result.Export(ExportFormatText)
	assert.NoError(t, err)
	assert.NotEmpty(t, exported)
	
	exportedStr = string(exported)
	assert.Contains(t, exportedStr, "Field1:test")
	assert.Contains(t, exportedStr, "Field2:42")
}

func TestResultInterfaceCompliance(t *testing.T) {
	// Test that BaseResult implements the Result interface
	var _ Result = (*BaseResult)(nil)
}