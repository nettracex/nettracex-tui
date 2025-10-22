package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.NotNil(t, manager.viper)
}

func TestManagerLoad(t *testing.T) {
	manager := NewManager()
	
	// Test loading with no config file (should use defaults)
	err := manager.Load()
	assert.NoError(t, err)
	
	// Verify default values are loaded
	config := manager.GetConfig()
	assert.Equal(t, 30*time.Second, config.Network.Timeout)
	assert.Equal(t, 30, config.Network.MaxHops)
	assert.Equal(t, 64, config.Network.PacketSize)
	assert.Equal(t, "NetTraceX/1.0", config.Network.UserAgent)
	assert.Equal(t, 10, config.Network.MaxConcurrency)
	assert.Equal(t, 3, config.Network.RetryAttempts)
	assert.Equal(t, time.Second, config.Network.RetryDelay)
	
	assert.Equal(t, "default", config.UI.Theme)
	assert.Equal(t, 250*time.Millisecond, config.UI.AnimationSpeed)
	assert.False(t, config.UI.AutoRefresh)
	assert.Equal(t, 5*time.Second, config.UI.RefreshInterval)
	assert.True(t, config.UI.ShowHelp)
	assert.Equal(t, "auto", config.UI.ColorMode)
	
	assert.Equal(t, domain.ExportFormatJSON, config.Export.DefaultFormat)
	assert.Equal(t, "./output", config.Export.OutputDirectory)
	assert.True(t, config.Export.IncludeMetadata)
	assert.False(t, config.Export.Compression)
	
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
	assert.Equal(t, "stdout", config.Logging.Output)
}

func TestManagerGetSet(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Test Get
	timeout := manager.Get("network.timeout")
	assert.Equal(t, "30s", timeout)
	
	theme := manager.Get("ui.theme")
	assert.Equal(t, "default", theme)
	
	// Test Set
	err = manager.Set("network.timeout", "60s")
	assert.NoError(t, err)
	
	newTimeout := manager.Get("network.timeout")
	assert.Equal(t, "60s", newTimeout)
	
	// Verify the config struct is updated
	config := manager.GetConfig()
	assert.Equal(t, 60*time.Second, config.Network.Timeout)
}

func TestManagerGetNetworkConfig(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	networkConfig := manager.GetNetworkConfig()
	assert.Equal(t, 30*time.Second, networkConfig.Timeout)
	assert.Equal(t, 30, networkConfig.MaxHops)
	assert.Equal(t, 64, networkConfig.PacketSize)
	assert.Len(t, networkConfig.DNSServers, 3)
	assert.Contains(t, networkConfig.DNSServers, "8.8.8.8")
	assert.Contains(t, networkConfig.DNSServers, "8.8.4.4")
	assert.Contains(t, networkConfig.DNSServers, "1.1.1.1")
}

func TestManagerGetUIConfig(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	uiConfig := manager.GetUIConfig()
	assert.Equal(t, "default", uiConfig.Theme)
	assert.Equal(t, 250*time.Millisecond, uiConfig.AnimationSpeed)
	assert.False(t, uiConfig.AutoRefresh)
	assert.Equal(t, 5*time.Second, uiConfig.RefreshInterval)
	assert.True(t, uiConfig.ShowHelp)
	assert.Equal(t, "auto", uiConfig.ColorMode)
	
	// Test key bindings
	assert.NotEmpty(t, uiConfig.KeyBindings)
	assert.Equal(t, "q", uiConfig.KeyBindings["quit"])
	assert.Equal(t, "?", uiConfig.KeyBindings["help"])
	assert.Equal(t, "esc", uiConfig.KeyBindings["back"])
}

func TestManagerSave(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "nettracex-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Set HOME to temp directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	
	manager := NewManager()
	err = manager.Load()
	assert.NoError(t, err)
	
	// Modify some configuration
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	
	err = manager.Set("ui.theme", "dark")
	assert.NoError(t, err)
	
	// Save configuration
	err = manager.Save()
	assert.NoError(t, err)
	
	// Verify config file was created
	configFile := filepath.Join(tempDir, ".config", "nettracex", "nettracex.yaml")
	assert.FileExists(t, configFile)
	
	// Load configuration in a new manager to verify persistence
	newManager := NewManager()
	err = newManager.Load()
	assert.NoError(t, err)
	
	assert.Equal(t, "45s", newManager.Get("network.timeout"))
	assert.Equal(t, "dark", newManager.Get("ui.theme"))
}

func TestValidatorValidateNetworkConfig(t *testing.T) {
	validator := NewValidator()
	
	// Test valid network config
	validConfig := &domain.NetworkConfig{
		Timeout:        30 * time.Second,
		MaxHops:        30,
		PacketSize:     64,
		DNSServers:     []string{"8.8.8.8"},
		UserAgent:      "NetTraceX/1.0",
		MaxConcurrency: 10,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
	}
	
	err := validator.validateNetworkConfig(validConfig)
	assert.NoError(t, err)
	
	// Test invalid timeout
	invalidConfig := *validConfig
	invalidConfig.Timeout = 0
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be positive")
	
	// Test invalid max hops (too small)
	invalidConfig = *validConfig
	invalidConfig.MaxHops = 0
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_hops must be between 1 and 255")
	
	// Test invalid max hops (too large)
	invalidConfig = *validConfig
	invalidConfig.MaxHops = 300
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_hops must be between 1 and 255")
	
	// Test invalid packet size (too small)
	invalidConfig = *validConfig
	invalidConfig.PacketSize = 0
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packet_size must be between 1 and 65507")
	
	// Test invalid packet size (too large)
	invalidConfig = *validConfig
	invalidConfig.PacketSize = 70000
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "packet_size must be between 1 and 65507")
	
	// Test invalid max concurrency
	invalidConfig = *validConfig
	invalidConfig.MaxConcurrency = 0
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_concurrency must be positive")
	
	// Test invalid retry attempts
	invalidConfig = *validConfig
	invalidConfig.RetryAttempts = -1
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry_attempts must be non-negative")
	
	// Test invalid retry delay
	invalidConfig = *validConfig
	invalidConfig.RetryDelay = -time.Second
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry_delay must be non-negative")
	
	// Test empty DNS servers
	invalidConfig = *validConfig
	invalidConfig.DNSServers = []string{}
	err = validator.validateNetworkConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one DNS server must be configured")
}

func TestValidatorValidateUIConfig(t *testing.T) {
	validator := NewValidator()
	
	// Test valid UI config
	validConfig := &domain.UIConfig{
		Theme:           "default",
		AnimationSpeed:  250 * time.Millisecond,
		KeyBindings:     map[string]string{"quit": "q"},
		AutoRefresh:     false,
		RefreshInterval: 5 * time.Second,
		ShowHelp:        true,
		ColorMode:       "auto",
	}
	
	err := validator.validateUIConfig(validConfig)
	assert.NoError(t, err)
	
	// Test invalid animation speed
	invalidConfig := *validConfig
	invalidConfig.AnimationSpeed = -time.Millisecond
	err = validator.validateUIConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "animation_speed must be non-negative")
	
	// Test invalid refresh interval
	invalidConfig = *validConfig
	invalidConfig.RefreshInterval = 0
	err = validator.validateUIConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refresh_interval must be positive")
	
	// Test invalid theme
	invalidConfig = *validConfig
	invalidConfig.Theme = "invalid"
	err = validator.validateUIConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "theme must be one of")
	
	// Test invalid color mode
	invalidConfig = *validConfig
	invalidConfig.ColorMode = "invalid"
	err = validator.validateUIConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "color_mode must be one of")
}

func TestValidatorValidateExportConfig(t *testing.T) {
	validator := NewValidator()
	
	// Test valid export config
	validConfig := &domain.ExportConfig{
		DefaultFormat:   domain.ExportFormatJSON,
		OutputDirectory: "./output",
		IncludeMetadata: true,
		Compression:     false,
	}
	
	err := validator.validateExportConfig(validConfig)
	assert.NoError(t, err)
	
	// Test invalid default format
	invalidConfig := *validConfig
	invalidConfig.DefaultFormat = domain.ExportFormat(999)
	err = validator.validateExportConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid default_format")
	
	// Test empty output directory
	invalidConfig = *validConfig
	invalidConfig.OutputDirectory = ""
	err = validator.validateExportConfig(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output_directory cannot be empty")
}

func TestValidatorValidateCompleteConfig(t *testing.T) {
	validator := NewValidator()
	
	// Test valid complete config
	validConfig := &domain.Config{
		Network: domain.NetworkConfig{
			Timeout:        30 * time.Second,
			MaxHops:        30,
			PacketSize:     64,
			DNSServers:     []string{"8.8.8.8"},
			UserAgent:      "NetTraceX/1.0",
			MaxConcurrency: 10,
			RetryAttempts:  3,
			RetryDelay:     time.Second,
		},
		UI: domain.UIConfig{
			Theme:           "default",
			AnimationSpeed:  250 * time.Millisecond,
			KeyBindings:     map[string]string{"quit": "q"},
			AutoRefresh:     false,
			RefreshInterval: 5 * time.Second,
			ShowHelp:        true,
			ColorMode:       "auto",
		},
		Export: domain.ExportConfig{
			DefaultFormat:   domain.ExportFormatJSON,
			OutputDirectory: "./output",
			IncludeMetadata: true,
			Compression:     false,
		},
	}
	
	err := validator.Validate(validConfig)
	assert.NoError(t, err)
	
	// Test config with invalid network settings
	invalidConfig := *validConfig
	invalidConfig.Network.Timeout = 0
	err = validator.Validate(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network config validation failed")
	
	// Test config with invalid UI settings
	invalidConfig = *validConfig
	invalidConfig.UI.Theme = "invalid"
	err = validator.Validate(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UI config validation failed")
	
	// Test config with invalid export settings
	invalidConfig = *validConfig
	invalidConfig.Export.OutputDirectory = ""
	err = validator.Validate(&invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "export config validation failed")
}

func TestManagerValidation(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Test validation passes with default config
	err = manager.Validate()
	assert.NoError(t, err)
	
	// Test validation fails with invalid setting
	err = manager.Set("network.timeout", "0s")
	assert.Error(t, err) // Should fail validation during Set
}

func TestManagerEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("NETTRACEX_NETWORK_TIMEOUT", "45s")
	os.Setenv("NETTRACEX_UI_THEME", "dark")
	os.Setenv("NETTRACEX_NETWORK_MAX_HOPS", "25")
	defer func() {
		os.Unsetenv("NETTRACEX_NETWORK_TIMEOUT")
		os.Unsetenv("NETTRACEX_UI_THEME")
		os.Unsetenv("NETTRACEX_NETWORK_MAX_HOPS")
	}()
	
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Verify environment variables override defaults
	assert.Equal(t, "45s", manager.Get("network.timeout"))
	assert.Equal(t, "dark", manager.Get("ui.theme"))
	assert.Equal(t, "25", manager.Get("network.max_hops")) // Environment variables are strings
	
	// Verify the config struct is updated (viper handles type conversion)
	config := manager.GetConfig()
	assert.Equal(t, 45*time.Second, config.Network.Timeout)
	assert.Equal(t, "dark", config.UI.Theme)
	assert.Equal(t, 25, config.Network.MaxHops)
}

func TestManagerLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "nettracex-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	configFile := filepath.Join(tempDir, "test-config.yaml")
	configContent := `
network:
  timeout: 60s
  max_hops: 20
ui:
  theme: light
  auto_refresh: true
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	assert.NoError(t, err)
	
	manager := NewManager()
	err = manager.LoadFromFile(configFile)
	assert.NoError(t, err)
	
	// Verify values from file
	assert.Equal(t, "60s", manager.Get("network.timeout"))
	assert.Equal(t, 20, manager.Get("network.max_hops"))
	assert.Equal(t, "light", manager.Get("ui.theme"))
	assert.Equal(t, true, manager.Get("ui.auto_refresh"))
	
	// Verify config file path is stored
	assert.Equal(t, configFile, manager.GetConfigFile())
}

func TestManagerSetMultiple(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Test successful multiple set
	values := map[string]interface{}{
		"network.timeout":  "45s",
		"ui.theme":         "dark",
		"network.max_hops": 25,
	}
	
	err = manager.SetMultiple(values)
	assert.NoError(t, err)
	
	// Verify all values were set
	assert.Equal(t, "45s", manager.Get("network.timeout"))
	assert.Equal(t, "dark", manager.Get("ui.theme"))
	assert.Equal(t, 25, manager.Get("network.max_hops"))
	
	// Test rollback on validation failure
	invalidValues := map[string]interface{}{
		"network.timeout":  "60s",  // Valid
		"network.max_hops": -1,     // Invalid - should cause rollback
	}
	
	err = manager.SetMultiple(invalidValues)
	assert.Error(t, err)
	
	// Verify original values are preserved
	assert.Equal(t, "45s", manager.Get("network.timeout"))
	assert.Equal(t, 25, manager.Get("network.max_hops"))
}

func TestManagerChangeListeners(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	var notifications []string
	listener := func(key string, oldValue, newValue interface{}) {
		notifications = append(notifications, fmt.Sprintf("%s: %v -> %v", key, oldValue, newValue))
	}
	
	manager.AddChangeListener(listener)
	
	// Make some changes
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	
	err = manager.Set("ui.theme", "dark")
	assert.NoError(t, err)
	
	// Verify notifications were sent
	assert.Len(t, notifications, 2)
	assert.Contains(t, notifications[0], "network.timeout")
	assert.Contains(t, notifications[1], "ui.theme")
	
	// Remove listener and verify no more notifications
	manager.RemoveChangeListener(listener)
	notifications = nil
	
	err = manager.Set("network.max_hops", 25)
	assert.NoError(t, err)
	assert.Len(t, notifications, 0)
}

func TestManagerReset(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Change some values
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	err = manager.Set("ui.theme", "dark")
	assert.NoError(t, err)
	
	// Verify changes
	assert.Equal(t, "45s", manager.Get("network.timeout"))
	assert.Equal(t, "dark", manager.Get("ui.theme"))
	
	// Reset configuration
	err = manager.Reset()
	assert.NoError(t, err)
	
	// Verify defaults are restored
	assert.Equal(t, "30s", manager.Get("network.timeout"))
	assert.Equal(t, "default", manager.Get("ui.theme"))
}

func TestManagerResetSection(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Change network values
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	err = manager.Set("network.max_hops", 25)
	assert.NoError(t, err)
	
	// Change UI values
	err = manager.Set("ui.theme", "dark")
	assert.NoError(t, err)
	
	// Reset only network section
	err = manager.ResetSection("network")
	assert.NoError(t, err)
	
	// Verify network values are reset but UI values remain
	assert.Equal(t, "30s", manager.Get("network.timeout"))
	assert.Equal(t, 30, manager.Get("network.max_hops"))
	assert.Equal(t, "dark", manager.Get("ui.theme")) // Should remain unchanged
	
	// Test invalid section
	err = manager.ResetSection("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown configuration section")
}

func TestManagerSaveAs(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "nettracex-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	manager := NewManager()
	err = manager.Load()
	assert.NoError(t, err)
	
	// Modify some configuration
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	
	// Save to specific file
	configFile := filepath.Join(tempDir, "custom-config.yaml")
	err = manager.SaveAs(configFile)
	assert.NoError(t, err)
	
	// Verify file was created
	assert.FileExists(t, configFile)
	
	// Verify config file path is updated
	assert.Equal(t, configFile, manager.GetConfigFile())
	
	// Load in a new manager to verify persistence
	newManager := NewManager()
	err = newManager.LoadFromFile(configFile)
	assert.NoError(t, err)
	
	assert.Equal(t, "45s", newManager.Get("network.timeout"))
}

func TestManagerGetConfigSections(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Test getting individual config sections
	networkConfig := manager.GetNetworkConfig()
	assert.Equal(t, 30*time.Second, networkConfig.Timeout)
	
	uiConfig := manager.GetUIConfig()
	assert.Equal(t, "default", uiConfig.Theme)
	
	pluginConfig := manager.GetPluginConfig()
	assert.Empty(t, pluginConfig.EnabledPlugins)
	
	exportConfig := manager.GetExportConfig()
	assert.Equal(t, domain.ExportFormatJSON, exportConfig.DefaultFormat)
	
	loggingConfig := manager.GetLoggingConfig()
	assert.Equal(t, "info", loggingConfig.Level)
}

func TestValidatorFieldValidation(t *testing.T) {
	validator := NewValidator()
	
	// Test network timeout validation
	err := validator.ValidateField("network.timeout", 30*time.Second)
	assert.NoError(t, err)
	
	err = validator.ValidateField("network.timeout", -time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be positive")
	
	err = validator.ValidateField("network.timeout", 10*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "should not exceed 5 minutes")
	
	// Test max hops validation
	err = validator.ValidateField("network.max_hops", 30)
	assert.NoError(t, err)
	
	err = validator.ValidateField("network.max_hops", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 1 and 255")
	
	err = validator.ValidateField("network.max_hops", 300)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be between 1 and 255")
	
	// Test theme validation
	err = validator.ValidateField("ui.theme", "default")
	assert.NoError(t, err)
	
	err = validator.ValidateField("ui.theme", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestManagerValidationOnSet(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Test that invalid values are rejected
	err = manager.Set("network.timeout", "0s")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
	
	// Verify original value is preserved
	assert.Equal(t, "30s", manager.Get("network.timeout"))
	
	// Test that valid values are accepted
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	assert.Equal(t, "45s", manager.Get("network.timeout"))
}

func TestConfigurationManagerInterfaceCompliance(t *testing.T) {
	// Test that Manager implements the ConfigurationManager interface
	var _ domain.ConfigurationManager = (*Manager)(nil)
}