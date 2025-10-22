// Package domain contains the core business logic interfaces and contracts
package domain

import (
	"context"
	"net"

	tea "github.com/charmbracelet/bubbletea"
)

// DiagnosticTool defines the contract for all network diagnostic tools
// Follows Single Responsibility Principle - each tool has one diagnostic purpose
type DiagnosticTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params Parameters) (Result, error)
	Validate(params Parameters) error
	GetModel() tea.Model
}

// Result represents the output of a diagnostic operation
// Follows Interface Segregation Principle - focused on result operations
type Result interface {
	Data() interface{}
	Metadata() map[string]interface{}
	Format(formatter OutputFormatter) string
	Export(format ExportFormat) ([]byte, error)
}

// Parameters represents input parameters for diagnostic operations
type Parameters interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Validate() error
	ToMap() map[string]interface{}
}

// OutputFormatter defines how results are formatted for display
type OutputFormatter interface {
	Format(data interface{}) string
	SetOptions(options map[string]interface{})
}

// ExportFormat represents different export formats
type ExportFormat int

const (
	ExportFormatJSON ExportFormat = iota
	ExportFormatCSV
	ExportFormatText
)

// NetworkClient abstracts network operations for testing and flexibility
// Follows Dependency Inversion Principle - high-level modules depend on abstractions
type NetworkClient interface {
	Ping(ctx context.Context, host string, opts PingOptions) (<-chan PingResult, error)
	Traceroute(ctx context.Context, host string, opts TraceOptions) (<-chan TraceHop, error)
	DNSLookup(ctx context.Context, domain string, recordType DNSRecordType) (DNSResult, error)
	WHOISLookup(ctx context.Context, query string) (WHOISResult, error)
	SSLCheck(ctx context.Context, host string, port int) (SSLResult, error)
}

// TUIComponent defines reusable UI components
// Follows Open/Closed Principle - extensible without modification
type TUIComponent interface {
	tea.Model
	SetSize(width, height int)
	SetTheme(theme Theme)
	Focus()
	Blur()
}

// PluginRegistry manages diagnostic tool plugins
// Follows Single Responsibility Principle - manages plugin lifecycle
type PluginRegistry interface {
	Register(tool DiagnosticTool) error
	Get(name string) (DiagnosticTool, bool)
	List() []DiagnosticTool
	Unregister(name string) error
}

// ConfigurationManager handles application configuration
// Follows Interface Segregation Principle - focused on configuration operations
type ConfigurationManager interface {
	Load() error
	Save() error
	Get(key string) interface{}
	Set(key string, value interface{}) error
	Validate() error
	GetNetworkConfig() NetworkConfig
	GetUIConfig() UIConfig
}

// Logger defines logging operations
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// ErrorHandler manages error processing and recovery
type ErrorHandler interface {
	Handle(err error) error
	HandleWithContext(err error, ctx map[string]interface{}) error
	CanRecover(err error) bool
	Recover(err error) error
}

// Validator defines validation operations
type Validator interface {
	Validate(data interface{}) error
	ValidateField(field string, value interface{}) error
	GetRules() map[string][]ValidationRule
}

// ValidationRule represents a single validation rule
type ValidationRule interface {
	Validate(value interface{}) error
	GetMessage() string
}

// GeoLocationService provides geographic data for network hops
type GeoLocationService interface {
	GetLocation(ip net.IP) (*GeoLocation, error)
	GetASNInfo(ip net.IP) (*ASNInfo, error)
	GetISPInfo(ip net.IP) (*ISPInfo, error)
}

// Theme defines UI theming interface
type Theme interface {
	GetColor(element string) string
	GetStyle(element string) map[string]interface{}
	SetColor(element, color string)
}