// Package domain contains core domain types and value objects
package domain

import (
	"crypto/x509"
	"net"
	"time"
)

// NetworkHost represents a network endpoint
type NetworkHost struct {
	Hostname   string       `json:"hostname"`
	IPAddress  net.IP       `json:"ip_address"`
	Port       int          `json:"port"`
	ASN        *ASNInfo     `json:"asn,omitempty"`
	Geographic *GeoLocation `json:"geographic,omitempty"`
}

// PingOptions contains configuration for ping operations
type PingOptions struct {
	Count       int           `json:"count"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	PacketSize  int           `json:"packet_size"`
	TTL         int           `json:"ttl"`
	IPv6        bool          `json:"ipv6"`
}

// PingResult contains ping operation results
type PingResult struct {
	Host       NetworkHost   `json:"host"`
	Sequence   int           `json:"sequence"`
	RTT        time.Duration `json:"rtt"`
	TTL        int           `json:"ttl"`
	PacketSize int           `json:"packet_size"`
	Timestamp  time.Time     `json:"timestamp"`
	Error      error         `json:"error,omitempty"`
}

// TraceOptions contains configuration for traceroute operations
type TraceOptions struct {
	MaxHops     int           `json:"max_hops"`
	Timeout     time.Duration `json:"timeout"`
	PacketSize  int           `json:"packet_size"`
	Queries     int           `json:"queries"`
	IPv6        bool          `json:"ipv6"`
}

// TraceHop represents a single hop in traceroute
type TraceHop struct {
	Number    int           `json:"number"`
	Host      NetworkHost   `json:"host"`
	RTT       []time.Duration `json:"rtt"`
	Timeout   bool          `json:"timeout"`
	Timestamp time.Time     `json:"timestamp"`
}

// DNSRecordType represents different DNS record types
type DNSRecordType int

const (
	DNSRecordTypeA DNSRecordType = iota
	DNSRecordTypeAAAA
	DNSRecordTypeMX
	DNSRecordTypeTXT
	DNSRecordTypeCNAME
	DNSRecordTypeNS
	DNSRecordTypeSOA
	DNSRecordTypePTR
)

// DNSRecord represents a single DNS record
type DNSRecord struct {
	Name     string        `json:"name"`
	Type     DNSRecordType `json:"type"`
	Value    string        `json:"value"`
	TTL      uint32        `json:"ttl"`
	Priority int           `json:"priority,omitempty"`
}

// DNSResult contains DNS lookup results
type DNSResult struct {
	Query        string      `json:"query"`
	RecordType   DNSRecordType `json:"record_type"`
	Records      []DNSRecord `json:"records"`
	Authority    []DNSRecord `json:"authority"`
	Additional   []DNSRecord `json:"additional"`
	ResponseTime time.Duration `json:"response_time"`
	Server       string      `json:"server"`
}

// Contact represents WHOIS contact information
type Contact struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
}

// WHOISResult contains WHOIS lookup data
type WHOISResult struct {
	Domain      string             `json:"domain"`
	Registrar   string             `json:"registrar"`
	Created     time.Time          `json:"created"`
	Updated     time.Time          `json:"updated"`
	Expires     time.Time          `json:"expires"`
	NameServers []string           `json:"name_servers"`
	Contacts    map[string]Contact `json:"contacts"`
	Status      []string           `json:"status"`
	RawData     string             `json:"raw_data"`
}

// SSLResult contains SSL certificate information
type SSLResult struct {
	Host        string               `json:"host"`
	Port        int                  `json:"port"`
	Certificate *x509.Certificate    `json:"certificate"`
	Chain       []*x509.Certificate  `json:"chain"`
	Valid       bool                 `json:"valid"`
	Errors      []string             `json:"errors"`
	Expiry      time.Time            `json:"expiry"`
	Issuer      string               `json:"issuer"`
	Subject     string               `json:"subject"`
	SANs        []string             `json:"sans"`
}

// GeoLocation represents geographic coordinates
type GeoLocation struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Timezone    string  `json:"timezone"`
}

// ASNInfo contains Autonomous System information
type ASNInfo struct {
	Number      int    `json:"number"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Country     string `json:"country"`
	Registry    string `json:"registry"`
}

// ISPInfo contains Internet Service Provider information
type ISPInfo struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	ASN          int    `json:"asn"`
	Country      string `json:"country"`
}

// NetworkConfig contains network operation settings
type NetworkConfig struct {
	Timeout        time.Duration `json:"timeout" mapstructure:"timeout"`
	MaxHops        int           `json:"max_hops" mapstructure:"max_hops"`
	PacketSize     int           `json:"packet_size" mapstructure:"packet_size"`
	DNSServers     []string      `json:"dns_servers" mapstructure:"dns_servers"`
	UserAgent      string        `json:"user_agent" mapstructure:"user_agent"`
	MaxConcurrency int           `json:"max_concurrency" mapstructure:"max_concurrency"`
	RetryAttempts  int           `json:"retry_attempts" mapstructure:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay" mapstructure:"retry_delay"`
}

// UIConfig contains UI preferences
type UIConfig struct {
	Theme           string            `json:"theme" mapstructure:"theme"`
	AnimationSpeed  time.Duration     `json:"animation_speed" mapstructure:"animation_speed"`
	KeyBindings     map[string]string `json:"key_bindings" mapstructure:"key_bindings"`
	AutoRefresh     bool              `json:"auto_refresh" mapstructure:"auto_refresh"`
	RefreshInterval time.Duration     `json:"refresh_interval" mapstructure:"refresh_interval"`
	ShowHelp        bool              `json:"show_help" mapstructure:"show_help"`
	ColorMode       string            `json:"color_mode" mapstructure:"color_mode"`
}

// PluginConfig contains plugin settings
type PluginConfig struct {
	EnabledPlugins  []string          `json:"enabled_plugins" mapstructure:"enabled_plugins"`
	DisabledPlugins []string          `json:"disabled_plugins" mapstructure:"disabled_plugins"`
	PluginPaths     []string          `json:"plugin_paths" mapstructure:"plugin_paths"`
	PluginSettings  map[string]interface{} `json:"plugin_settings" mapstructure:"plugin_settings"`
}

// ExportConfig contains export settings
type ExportConfig struct {
	DefaultFormat   ExportFormat `json:"default_format" mapstructure:"default_format"`
	OutputDirectory string       `json:"output_directory" mapstructure:"output_directory"`
	IncludeMetadata bool         `json:"include_metadata" mapstructure:"include_metadata"`
	Compression     bool         `json:"compression" mapstructure:"compression"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `json:"level" mapstructure:"level"`
	Format     string `json:"format" mapstructure:"format"`
	Output     string `json:"output" mapstructure:"output"`
	MaxSize    int    `json:"max_size" mapstructure:"max_size"`
	MaxBackups int    `json:"max_backups" mapstructure:"max_backups"`
	MaxAge     int    `json:"max_age" mapstructure:"max_age"`
}

// Config represents the complete application configuration
type Config struct {
	Network NetworkConfig `json:"network" mapstructure:"network"`
	UI      UIConfig      `json:"ui" mapstructure:"ui"`
	Plugins PluginConfig  `json:"plugins" mapstructure:"plugins"`
	Export  ExportConfig  `json:"export" mapstructure:"export"`
	Logging LoggingConfig `json:"logging" mapstructure:"logging"`
}

// ErrorType represents different categories of errors
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypeValidation
	ErrorTypeConfiguration
	ErrorTypePlugin
	ErrorTypeUI
	ErrorTypeExport
	ErrorTypeSystem
)

// NetTraceError represents application-specific errors
type NetTraceError struct {
	Type      ErrorType              `json:"type"`
	Message   string                 `json:"message"`
	Cause     error                  `json:"cause,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Code      string                 `json:"code"`
}

// Error implements the error interface
func (e *NetTraceError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *NetTraceError) Unwrap() error {
	return e.Cause
}