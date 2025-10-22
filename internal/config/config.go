// Package config provides configuration management functionality
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/spf13/viper"
)

// Manager implements the ConfigurationManager interface
type Manager struct {
	config     *domain.Config
	viper      *viper.Viper
	configFile string
	validator  *Validator
	listeners  []ConfigChangeListener
}

// ConfigChangeListener defines a callback for configuration changes
type ConfigChangeListener func(key string, oldValue, newValue interface{})

// NewManager creates a new configuration manager
func NewManager() *Manager {
	v := viper.New()
	
	// Set configuration file properties
	v.SetConfigName("nettracex")
	v.SetConfigType("yaml")
	
	// Add configuration paths
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/nettracex")
	v.AddConfigPath("/etc/nettracex")
	
	// Set environment variable prefix and enable automatic env binding
	v.SetEnvPrefix("NETTRACEX")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	
	// Bind all configuration keys to environment variables
	bindEnvironmentVariables(v)
	
	// Set default values
	setDefaults(v)
	
	validator := NewValidator()
	
	return &Manager{
		config:    &domain.Config{},
		viper:     v,
		validator: validator,
		listeners: make([]ConfigChangeListener, 0),
	}
}

// bindEnvironmentVariables binds all configuration keys to environment variables
func bindEnvironmentVariables(v *viper.Viper) {
	// Network configuration
	v.BindEnv("network.timeout", "NETTRACEX_NETWORK_TIMEOUT")
	v.BindEnv("network.max_hops", "NETTRACEX_NETWORK_MAX_HOPS")
	v.BindEnv("network.packet_size", "NETTRACEX_NETWORK_PACKET_SIZE")
	v.BindEnv("network.dns_servers", "NETTRACEX_NETWORK_DNS_SERVERS")
	v.BindEnv("network.user_agent", "NETTRACEX_NETWORK_USER_AGENT")
	v.BindEnv("network.max_concurrency", "NETTRACEX_NETWORK_MAX_CONCURRENCY")
	v.BindEnv("network.retry_attempts", "NETTRACEX_NETWORK_RETRY_ATTEMPTS")
	v.BindEnv("network.retry_delay", "NETTRACEX_NETWORK_RETRY_DELAY")
	
	// UI configuration
	v.BindEnv("ui.theme", "NETTRACEX_UI_THEME")
	v.BindEnv("ui.animation_speed", "NETTRACEX_UI_ANIMATION_SPEED")
	v.BindEnv("ui.auto_refresh", "NETTRACEX_UI_AUTO_REFRESH")
	v.BindEnv("ui.refresh_interval", "NETTRACEX_UI_REFRESH_INTERVAL")
	v.BindEnv("ui.show_help", "NETTRACEX_UI_SHOW_HELP")
	v.BindEnv("ui.color_mode", "NETTRACEX_UI_COLOR_MODE")
	
	// Plugin configuration
	v.BindEnv("plugins.enabled_plugins", "NETTRACEX_PLUGINS_ENABLED_PLUGINS")
	v.BindEnv("plugins.disabled_plugins", "NETTRACEX_PLUGINS_DISABLED_PLUGINS")
	v.BindEnv("plugins.plugin_paths", "NETTRACEX_PLUGINS_PLUGIN_PATHS")
	
	// Export configuration
	v.BindEnv("export.default_format", "NETTRACEX_EXPORT_DEFAULT_FORMAT")
	v.BindEnv("export.output_directory", "NETTRACEX_EXPORT_OUTPUT_DIRECTORY")
	v.BindEnv("export.include_metadata", "NETTRACEX_EXPORT_INCLUDE_METADATA")
	v.BindEnv("export.compression", "NETTRACEX_EXPORT_COMPRESSION")
	
	// Logging configuration
	v.BindEnv("logging.level", "NETTRACEX_LOGGING_LEVEL")
	v.BindEnv("logging.format", "NETTRACEX_LOGGING_FORMAT")
	v.BindEnv("logging.output", "NETTRACEX_LOGGING_OUTPUT")
	v.BindEnv("logging.max_size", "NETTRACEX_LOGGING_MAX_SIZE")
	v.BindEnv("logging.max_backups", "NETTRACEX_LOGGING_MAX_BACKUPS")
	v.BindEnv("logging.max_age", "NETTRACEX_LOGGING_MAX_AGE")
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Network defaults
	v.SetDefault("network.timeout", "30s")
	v.SetDefault("network.max_hops", 30)
	v.SetDefault("network.packet_size", 64)
	v.SetDefault("network.dns_servers", []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"})
	v.SetDefault("network.user_agent", "NetTraceX/1.0")
	v.SetDefault("network.max_concurrency", 10)
	v.SetDefault("network.retry_attempts", 3)
	v.SetDefault("network.retry_delay", "1s")
	
	// UI defaults
	v.SetDefault("ui.theme", "default")
	v.SetDefault("ui.animation_speed", "250ms")
	v.SetDefault("ui.auto_refresh", false)
	v.SetDefault("ui.refresh_interval", "5s")
	v.SetDefault("ui.show_help", true)
	v.SetDefault("ui.color_mode", "auto")
	
	// Default key bindings
	keyBindings := map[string]string{
		"quit":         "q",
		"help":         "?",
		"back":         "esc",
		"up":           "up",
		"down":         "down",
		"left":         "left",
		"right":        "right",
		"select":       "enter",
		"tab":          "tab",
		"shift_tab":    "shift+tab",
		"page_up":      "pgup",
		"page_down":    "pgdown",
		"home":         "home",
		"end":          "end",
		"export":       "e",
		"save":         "s",
		"refresh":      "r",
	}
	v.SetDefault("ui.key_bindings", keyBindings)
	
	// Plugin defaults
	v.SetDefault("plugins.enabled_plugins", []string{})
	v.SetDefault("plugins.disabled_plugins", []string{})
	v.SetDefault("plugins.plugin_paths", []string{"./plugins"})
	v.SetDefault("plugins.plugin_settings", map[string]interface{}{})
	
	// Export defaults
	v.SetDefault("export.default_format", int(domain.ExportFormatJSON))
	v.SetDefault("export.output_directory", "./output")
	v.SetDefault("export.include_metadata", true)
	v.SetDefault("export.compression", false)
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_backups", 3)
	v.SetDefault("logging.max_age", 28)
}

// Load loads configuration from file and environment variables
func (m *Manager) Load() error {
	// Try to read configuration file
	if err := m.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable, we'll use defaults and environment variables
	} else {
		// Store the config file path for future saves
		m.configFile = m.viper.ConfigFileUsed()
	}
	
	// Unmarshal configuration into struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate the loaded configuration
	if err := m.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return nil
}

// LoadFromFile loads configuration from a specific file path
func (m *Manager) LoadFromFile(filePath string) error {
	m.viper.SetConfigFile(filePath)
	
	if err := m.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}
	
	m.configFile = filePath
	
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return m.Validate()
}

// GetConfigFile returns the path of the currently loaded config file
func (m *Manager) GetConfigFile() string {
	return m.configFile
}

// Save saves the current configuration to file
func (m *Manager) Save() error {
	var configFile string
	
	if m.configFile != "" {
		// Use existing config file location
		configFile = m.configFile
	} else {
		// Create default config file location
		configDir := filepath.Join(os.Getenv("HOME"), ".config", "nettracex")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configFile = filepath.Join(configDir, "nettracex.yaml")
		m.configFile = configFile
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := m.viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// SaveAs saves the current configuration to a specific file path
func (m *Manager) SaveAs(filePath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	if err := m.viper.WriteConfigAs(filePath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	m.configFile = filePath
	return nil
}

// Get retrieves a configuration value by key
func (m *Manager) Get(key string) interface{} {
	return m.viper.Get(key)
}

// Set sets a configuration value by key
func (m *Manager) Set(key string, value interface{}) error {
	oldValue := m.viper.Get(key)
	
	m.viper.Set(key, value)
	
	// Re-unmarshal to update the config struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	
	// Validate the new configuration
	if err := m.Validate(); err != nil {
		// Rollback on validation failure
		m.viper.Set(key, oldValue)
		m.viper.Unmarshal(m.config)
		return fmt.Errorf("validation failed for key %s: %w", key, err)
	}
	
	// Notify listeners of the change
	m.notifyListeners(key, oldValue, value)
	
	return nil
}

// SetMultiple sets multiple configuration values atomically
func (m *Manager) SetMultiple(values map[string]interface{}) error {
	// Store original values for rollback
	originalValues := make(map[string]interface{})
	for key := range values {
		originalValues[key] = m.viper.Get(key)
	}
	
	// Apply all changes
	for key, value := range values {
		m.viper.Set(key, value)
	}
	
	// Re-unmarshal to update the config struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		// Rollback all changes
		for key, value := range originalValues {
			m.viper.Set(key, value)
		}
		m.viper.Unmarshal(m.config)
		return fmt.Errorf("failed to update config: %w", err)
	}
	
	// Validate the new configuration
	if err := m.Validate(); err != nil {
		// Rollback all changes
		for key, value := range originalValues {
			m.viper.Set(key, value)
		}
		m.viper.Unmarshal(m.config)
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Notify listeners of all changes
	for key, newValue := range values {
		m.notifyListeners(key, originalValues[key], newValue)
	}
	
	return nil
}

// AddChangeListener adds a configuration change listener
func (m *Manager) AddChangeListener(listener ConfigChangeListener) {
	m.listeners = append(m.listeners, listener)
}

// RemoveChangeListener removes a configuration change listener
func (m *Manager) RemoveChangeListener(listener ConfigChangeListener) {
	for i, l := range m.listeners {
		if reflect.ValueOf(l).Pointer() == reflect.ValueOf(listener).Pointer() {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			break
		}
	}
}

// notifyListeners notifies all registered listeners of configuration changes
func (m *Manager) notifyListeners(key string, oldValue, newValue interface{}) {
	for _, listener := range m.listeners {
		listener(key, oldValue, newValue)
	}
}

// Validate validates the current configuration
func (m *Manager) Validate() error {
	validator := NewValidator()
	return validator.Validate(m.config)
}

// GetNetworkConfig returns the network configuration
func (m *Manager) GetNetworkConfig() domain.NetworkConfig {
	return m.config.Network
}

// GetUIConfig returns the UI configuration
func (m *Manager) GetUIConfig() domain.UIConfig {
	return m.config.UI
}

// GetConfig returns the complete configuration
func (m *Manager) GetConfig() *domain.Config {
	return m.config
}

// GetPluginConfig returns the plugin configuration
func (m *Manager) GetPluginConfig() domain.PluginConfig {
	return m.config.Plugins
}

// GetExportConfig returns the export configuration
func (m *Manager) GetExportConfig() domain.ExportConfig {
	return m.config.Export
}

// GetLoggingConfig returns the logging configuration
func (m *Manager) GetLoggingConfig() domain.LoggingConfig {
	return m.config.Logging
}

// Reset resets configuration to default values
func (m *Manager) Reset() error {
	// Create a new viper instance with defaults
	v := viper.New()
	setDefaults(v)
	bindEnvironmentVariables(v)
	
	// Replace the current viper instance
	m.viper = v
	
	// Re-unmarshal to update the config struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to reset config: %w", err)
	}
	
	return nil
}

// ResetSection resets a specific configuration section to defaults
func (m *Manager) ResetSection(section string) error {
	switch section {
	case "network":
		m.viper.Set("network.timeout", "30s")
		m.viper.Set("network.max_hops", 30)
		m.viper.Set("network.packet_size", 64)
		m.viper.Set("network.dns_servers", []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"})
		m.viper.Set("network.user_agent", "NetTraceX/1.0")
		m.viper.Set("network.max_concurrency", 10)
		m.viper.Set("network.retry_attempts", 3)
		m.viper.Set("network.retry_delay", "1s")
	case "ui":
		m.viper.Set("ui.theme", "default")
		m.viper.Set("ui.animation_speed", "250ms")
		m.viper.Set("ui.auto_refresh", false)
		m.viper.Set("ui.refresh_interval", "5s")
		m.viper.Set("ui.show_help", true)
		m.viper.Set("ui.color_mode", "auto")
		// Reset key bindings to defaults
		keyBindings := map[string]string{
			"quit": "q", "help": "?", "back": "esc",
			"up": "up", "down": "down", "left": "left", "right": "right",
			"select": "enter", "tab": "tab", "shift_tab": "shift+tab",
			"page_up": "pgup", "page_down": "pgdown", "home": "home", "end": "end",
			"export": "e", "save": "s", "refresh": "r",
		}
		m.viper.Set("ui.key_bindings", keyBindings)
	case "plugins":
		m.viper.Set("plugins.enabled_plugins", []string{})
		m.viper.Set("plugins.disabled_plugins", []string{})
		m.viper.Set("plugins.plugin_paths", []string{"./plugins"})
		m.viper.Set("plugins.plugin_settings", map[string]interface{}{})
	case "export":
		m.viper.Set("export.default_format", int(domain.ExportFormatJSON))
		m.viper.Set("export.output_directory", "./output")
		m.viper.Set("export.include_metadata", true)
		m.viper.Set("export.compression", false)
	case "logging":
		m.viper.Set("logging.level", "info")
		m.viper.Set("logging.format", "text")
		m.viper.Set("logging.output", "stdout")
		m.viper.Set("logging.max_size", 100)
		m.viper.Set("logging.max_backups", 3)
		m.viper.Set("logging.max_age", 28)
	default:
		return fmt.Errorf("unknown configuration section: %s", section)
	}
	
	// Re-unmarshal to update the config struct
	if err := m.viper.Unmarshal(m.config); err != nil {
		return fmt.Errorf("failed to reset section %s: %w", section, err)
	}
	
	return m.Validate()
}

// Validator implements configuration validation
type Validator struct {
	rules map[string][]ValidationRule
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name     string
	Validate func(interface{}) error
	Message  string
}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	v := &Validator{
		rules: make(map[string][]ValidationRule),
	}
	v.setupValidationRules()
	return v
}

// setupValidationRules sets up all validation rules
func (v *Validator) setupValidationRules() {
	// Network timeout validation
	v.rules["network.timeout"] = []ValidationRule{
		{
			Name: "positive_duration",
			Validate: func(value interface{}) error {
				if duration, ok := value.(time.Duration); ok {
					if duration <= 0 {
						return fmt.Errorf("timeout must be positive")
					}
				}
				return nil
			},
			Message: "Timeout must be a positive duration",
		},
		{
			Name: "reasonable_timeout",
			Validate: func(value interface{}) error {
				if duration, ok := value.(time.Duration); ok {
					if duration > 5*time.Minute {
						return fmt.Errorf("timeout should not exceed 5 minutes")
					}
				}
				return nil
			},
			Message: "Timeout should not exceed 5 minutes for practical use",
		},
	}
	
	// Max hops validation
	v.rules["network.max_hops"] = []ValidationRule{
		{
			Name: "valid_range",
			Validate: func(value interface{}) error {
				if hops, ok := value.(int); ok {
					if hops <= 0 || hops > 255 {
						return fmt.Errorf("max_hops must be between 1 and 255")
					}
				}
				return nil
			},
			Message: "Max hops must be between 1 and 255",
		},
	}
	
	// Theme validation
	v.rules["ui.theme"] = []ValidationRule{
		{
			Name: "valid_theme",
			Validate: func(value interface{}) error {
				if theme, ok := value.(string); ok {
					validThemes := []string{"default", "dark", "light", "minimal"}
					for _, valid := range validThemes {
						if theme == valid {
							return nil
						}
					}
					return fmt.Errorf("theme must be one of: %v", validThemes)
				}
				return nil
			},
			Message: "Theme must be one of: default, dark, light, minimal",
		},
	}
}

// ValidateField validates a specific configuration field
func (v *Validator) ValidateField(key string, value interface{}) error {
	if rules, exists := v.rules[key]; exists {
		for _, rule := range rules {
			if err := rule.Validate(value); err != nil {
				return fmt.Errorf("%s: %s", rule.Message, err.Error())
			}
		}
	}
	return nil
}

// Validate validates the configuration
func (v *Validator) Validate(config *domain.Config) error {
	if err := v.validateNetworkConfig(&config.Network); err != nil {
		return fmt.Errorf("network config validation failed: %w", err)
	}
	
	if err := v.validateUIConfig(&config.UI); err != nil {
		return fmt.Errorf("UI config validation failed: %w", err)
	}
	
	if err := v.validateExportConfig(&config.Export); err != nil {
		return fmt.Errorf("export config validation failed: %w", err)
	}
	
	return nil
}

// validateNetworkConfig validates network configuration
func (v *Validator) validateNetworkConfig(config *domain.NetworkConfig) error {
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	if config.MaxHops <= 0 || config.MaxHops > 255 {
		return fmt.Errorf("max_hops must be between 1 and 255")
	}
	
	if config.PacketSize <= 0 || config.PacketSize > 65507 {
		return fmt.Errorf("packet_size must be between 1 and 65507")
	}
	
	if config.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be positive")
	}
	
	if config.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts must be non-negative")
	}
	
	if config.RetryDelay < 0 {
		return fmt.Errorf("retry_delay must be non-negative")
	}
	
	if len(config.DNSServers) == 0 {
		return fmt.Errorf("at least one DNS server must be configured")
	}
	
	return nil
}

// validateUIConfig validates UI configuration
func (v *Validator) validateUIConfig(config *domain.UIConfig) error {
	if config.AnimationSpeed < 0 {
		return fmt.Errorf("animation_speed must be non-negative")
	}
	
	if config.RefreshInterval <= 0 {
		return fmt.Errorf("refresh_interval must be positive")
	}
	
	validThemes := []string{"default", "dark", "light", "minimal"}
	if !contains(validThemes, config.Theme) {
		return fmt.Errorf("theme must be one of: %v", validThemes)
	}
	
	validColorModes := []string{"auto", "always", "never"}
	if !contains(validColorModes, config.ColorMode) {
		return fmt.Errorf("color_mode must be one of: %v", validColorModes)
	}
	
	return nil
}

// validateExportConfig validates export configuration
func (v *Validator) validateExportConfig(config *domain.ExportConfig) error {
	if config.DefaultFormat < 0 || config.DefaultFormat > domain.ExportFormatText {
		return fmt.Errorf("invalid default_format")
	}
	
	if config.OutputDirectory == "" {
		return fmt.Errorf("output_directory cannot be empty")
	}
	
	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}