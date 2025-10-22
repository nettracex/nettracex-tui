package domain

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNetworkHost(t *testing.T) {
	host := NetworkHost{
		Hostname:  "example.com",
		IPAddress: net.ParseIP("192.168.1.1"),
		Port:      80,
		ASN: &ASNInfo{
			Number:      12345,
			Name:        "Example ASN",
			Description: "Example Autonomous System",
			Country:     "US",
		},
		Geographic: &GeoLocation{
			Latitude:    37.7749,
			Longitude:   -122.4194,
			City:        "San Francisco",
			Country:     "United States",
			CountryCode: "US",
		},
	}
	
	assert.Equal(t, "example.com", host.Hostname)
	assert.Equal(t, "192.168.1.1", host.IPAddress.String())
	assert.Equal(t, 80, host.Port)
	assert.NotNil(t, host.ASN)
	assert.Equal(t, 12345, host.ASN.Number)
	assert.NotNil(t, host.Geographic)
	assert.Equal(t, "San Francisco", host.Geographic.City)
}

func TestPingOptions(t *testing.T) {
	opts := PingOptions{
		Count:      10,
		Interval:   time.Second,
		Timeout:    5 * time.Second,
		PacketSize: 64,
		TTL:        64,
		IPv6:       false,
	}
	
	assert.Equal(t, 10, opts.Count)
	assert.Equal(t, time.Second, opts.Interval)
	assert.Equal(t, 5*time.Second, opts.Timeout)
	assert.Equal(t, 64, opts.PacketSize)
	assert.Equal(t, 64, opts.TTL)
	assert.False(t, opts.IPv6)
}

func TestPingResult(t *testing.T) {
	now := time.Now()
	result := PingResult{
		Host: NetworkHost{
			Hostname:  "example.com",
			IPAddress: net.ParseIP("192.168.1.1"),
		},
		Sequence:   1,
		RTT:        10 * time.Millisecond,
		TTL:        64,
		PacketSize: 64,
		Timestamp:  now,
		Error:      nil,
	}
	
	assert.Equal(t, "example.com", result.Host.Hostname)
	assert.Equal(t, 1, result.Sequence)
	assert.Equal(t, 10*time.Millisecond, result.RTT)
	assert.Equal(t, 64, result.TTL)
	assert.Equal(t, 64, result.PacketSize)
	assert.Equal(t, now, result.Timestamp)
	assert.Nil(t, result.Error)
}

func TestTraceHop(t *testing.T) {
	now := time.Now()
	hop := TraceHop{
		Number: 1,
		Host: NetworkHost{
			Hostname:  "gateway.example.com",
			IPAddress: net.ParseIP("192.168.1.1"),
		},
		RTT:       []time.Duration{10 * time.Millisecond, 12 * time.Millisecond, 11 * time.Millisecond},
		Timeout:   false,
		Timestamp: now,
	}
	
	assert.Equal(t, 1, hop.Number)
	assert.Equal(t, "gateway.example.com", hop.Host.Hostname)
	assert.Len(t, hop.RTT, 3)
	assert.Equal(t, 10*time.Millisecond, hop.RTT[0])
	assert.False(t, hop.Timeout)
	assert.Equal(t, now, hop.Timestamp)
}

func TestDNSRecordType(t *testing.T) {
	assert.Equal(t, DNSRecordType(0), DNSRecordTypeA)
	assert.Equal(t, DNSRecordType(1), DNSRecordTypeAAAA)
	assert.Equal(t, DNSRecordType(2), DNSRecordTypeMX)
	assert.Equal(t, DNSRecordType(3), DNSRecordTypeTXT)
	assert.Equal(t, DNSRecordType(4), DNSRecordTypeCNAME)
	assert.Equal(t, DNSRecordType(5), DNSRecordTypeNS)
}

func TestDNSRecord(t *testing.T) {
	record := DNSRecord{
		Name:     "example.com",
		Type:     DNSRecordTypeA,
		Value:    "192.168.1.1",
		TTL:      300,
		Priority: 0,
	}
	
	assert.Equal(t, "example.com", record.Name)
	assert.Equal(t, DNSRecordTypeA, record.Type)
	assert.Equal(t, "192.168.1.1", record.Value)
	assert.Equal(t, uint32(300), record.TTL)
	assert.Equal(t, 0, record.Priority)
}

func TestDNSResult(t *testing.T) {
	result := DNSResult{
		Query:      "example.com",
		RecordType: DNSRecordTypeA,
		Records: []DNSRecord{
			{
				Name:  "example.com",
				Type:  DNSRecordTypeA,
				Value: "192.168.1.1",
				TTL:   300,
			},
		},
		Authority:    []DNSRecord{},
		Additional:   []DNSRecord{},
		ResponseTime: 50 * time.Millisecond,
		Server:       "8.8.8.8",
	}
	
	assert.Equal(t, "example.com", result.Query)
	assert.Equal(t, DNSRecordTypeA, result.RecordType)
	assert.Len(t, result.Records, 1)
	assert.Equal(t, "192.168.1.1", result.Records[0].Value)
	assert.Equal(t, 50*time.Millisecond, result.ResponseTime)
	assert.Equal(t, "8.8.8.8", result.Server)
}

func TestWHOISResult(t *testing.T) {
	created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	expires := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	result := WHOISResult{
		Domain:      "example.com",
		Registrar:   "Example Registrar",
		Created:     created,
		Updated:     updated,
		Expires:     expires,
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
		Contacts: map[string]Contact{
			"registrant": {
				Name:         "John Doe",
				Organization: "Example Corp",
				Email:        "john@example.com",
			},
		},
		Status:  []string{"clientTransferProhibited"},
		RawData: "Raw WHOIS data...",
	}
	
	assert.Equal(t, "example.com", result.Domain)
	assert.Equal(t, "Example Registrar", result.Registrar)
	assert.Equal(t, created, result.Created)
	assert.Equal(t, updated, result.Updated)
	assert.Equal(t, expires, result.Expires)
	assert.Len(t, result.NameServers, 2)
	assert.Contains(t, result.NameServers, "ns1.example.com")
	assert.Len(t, result.Contacts, 1)
	assert.Equal(t, "John Doe", result.Contacts["registrant"].Name)
	assert.Len(t, result.Status, 1)
	assert.Equal(t, "clientTransferProhibited", result.Status[0])
}

func TestSSLResult(t *testing.T) {
	expiry := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	
	result := SSLResult{
		Host:        "example.com",
		Port:        443,
		Certificate: nil, // Would be a real certificate in practice
		Chain:       nil, // Would be certificate chain in practice
		Valid:       true,
		Errors:      []string{},
		Expiry:      expiry,
		Issuer:      "Let's Encrypt Authority X3",
		Subject:     "CN=example.com",
		SANs:        []string{"example.com", "www.example.com"},
	}
	
	assert.Equal(t, "example.com", result.Host)
	assert.Equal(t, 443, result.Port)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Equal(t, expiry, result.Expiry)
	assert.Equal(t, "Let's Encrypt Authority X3", result.Issuer)
	assert.Equal(t, "CN=example.com", result.Subject)
	assert.Len(t, result.SANs, 2)
	assert.Contains(t, result.SANs, "example.com")
	assert.Contains(t, result.SANs, "www.example.com")
}

func TestConfig(t *testing.T) {
	config := Config{
		Network: NetworkConfig{
			Timeout:        30 * time.Second,
			MaxHops:        30,
			PacketSize:     64,
			DNSServers:     []string{"8.8.8.8", "1.1.1.1"},
			UserAgent:      "NetTraceX/1.0",
			MaxConcurrency: 10,
			RetryAttempts:  3,
			RetryDelay:     time.Second,
		},
		UI: UIConfig{
			Theme:           "default",
			AnimationSpeed:  250 * time.Millisecond,
			KeyBindings:     map[string]string{"quit": "q", "help": "?"},
			AutoRefresh:     false,
			RefreshInterval: 5 * time.Second,
			ShowHelp:        true,
			ColorMode:       "auto",
		},
		Plugins: PluginConfig{
			EnabledPlugins:  []string{"ping", "traceroute"},
			DisabledPlugins: []string{},
			PluginPaths:     []string{"./plugins"},
			PluginSettings:  map[string]interface{}{},
		},
		Export: ExportConfig{
			DefaultFormat:   ExportFormatJSON,
			OutputDirectory: "./output",
			IncludeMetadata: true,
			Compression:     false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}
	
	assert.Equal(t, 30*time.Second, config.Network.Timeout)
	assert.Equal(t, 30, config.Network.MaxHops)
	assert.Equal(t, "default", config.UI.Theme)
	assert.Equal(t, 250*time.Millisecond, config.UI.AnimationSpeed)
	assert.Len(t, config.Plugins.EnabledPlugins, 2)
	assert.Equal(t, ExportFormatJSON, config.Export.DefaultFormat)
	assert.Equal(t, "info", config.Logging.Level)
}

func TestNetTraceError(t *testing.T) {
	now := time.Now()
	err := &NetTraceError{
		Type:      ErrorTypeNetwork,
		Message:   "Network connection failed",
		Cause:     nil,
		Context:   map[string]interface{}{"host": "example.com"},
		Timestamp: now,
		Code:      "NET001",
	}
	
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "Network connection failed", err.Message)
	assert.Nil(t, err.Cause)
	assert.Equal(t, "example.com", err.Context["host"])
	assert.Equal(t, now, err.Timestamp)
	assert.Equal(t, "NET001", err.Code)
	assert.Equal(t, "Network connection failed", err.Error())
}

func TestNetTraceErrorWithCause(t *testing.T) {
	cause := assert.AnError
	err := &NetTraceError{
		Type:    ErrorTypeNetwork,
		Message: "Network operation failed",
		Cause:   cause,
	}
	
	assert.Equal(t, cause, err.Unwrap())
	assert.Contains(t, err.Error(), "Network operation failed")
	assert.Contains(t, err.Error(), cause.Error())
}

func TestExportFormat(t *testing.T) {
	assert.Equal(t, ExportFormat(0), ExportFormatJSON)
	assert.Equal(t, ExportFormat(1), ExportFormatCSV)
	assert.Equal(t, ExportFormat(2), ExportFormatText)
}

func TestErrorType(t *testing.T) {
	assert.Equal(t, ErrorType(0), ErrorTypeNetwork)
	assert.Equal(t, ErrorType(1), ErrorTypeValidation)
	assert.Equal(t, ErrorType(2), ErrorTypeConfiguration)
	assert.Equal(t, ErrorType(3), ErrorTypePlugin)
	assert.Equal(t, ErrorType(4), ErrorTypeUI)
	assert.Equal(t, ErrorType(5), ErrorTypeExport)
	assert.Equal(t, ErrorType(6), ErrorTypeSystem)
}