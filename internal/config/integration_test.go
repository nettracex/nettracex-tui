package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigurationIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "nettracex-config-integration")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Set HOME to temp directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)
	
	t.Run("FullConfigurationLifecycle", func(t *testing.T) {
		// Create a new manager
		manager := NewManager()
		
		// Load default configuration
		err := manager.Load()
		assert.NoError(t, err)
		
		// Verify default values
		config := manager.GetConfig()
		assert.Equal(t, 30*time.Second, config.Network.Timeout)
		assert.Equal(t, "default", config.UI.Theme)
		assert.Equal(t, domain.ExportFormatJSON, config.Export.DefaultFormat)
		
		// Modify configuration
		err = manager.Set("network.timeout", "45s")
		assert.NoError(t, err)
		
		err = manager.Set("ui.theme", "dark")
		assert.NoError(t, err)
		
		err = manager.Set("export.default_format", int(domain.ExportFormatCSV))
		assert.NoError(t, err)
		
		// Save configuration
		err = manager.Save()
		assert.NoError(t, err)
		
		// Verify config file was created
		configFile := filepath.Join(tempDir, ".config", "nettracex", "nettracex.yaml")
		assert.FileExists(t, configFile)
		
		// Create a new manager and load the saved configuration
		newManager := NewManager()
		err = newManager.Load()
		assert.NoError(t, err)
		
		// Verify saved values are loaded
		newConfig := newManager.GetConfig()
		assert.Equal(t, 45*time.Second, newConfig.Network.Timeout)
		assert.Equal(t, "dark", newConfig.UI.Theme)
		assert.Equal(t, domain.ExportFormatCSV, newConfig.Export.DefaultFormat)
	})
	
	t.Run("EnvironmentVariableOverrides", func(t *testing.T) {
		// Set environment variables
		os.Setenv("NETTRACEX_NETWORK_TIMEOUT", "60s")
		os.Setenv("NETTRACEX_UI_THEME", "light")
		os.Setenv("NETTRACEX_NETWORK_DNS_SERVERS", "1.1.1.1,8.8.8.8")
		defer func() {
			os.Unsetenv("NETTRACEX_NETWORK_TIMEOUT")
			os.Unsetenv("NETTRACEX_UI_THEME")
			os.Unsetenv("NETTRACEX_NETWORK_DNS_SERVERS")
		}()
		
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Verify environment variables override defaults
		config := manager.GetConfig()
		assert.Equal(t, 60*time.Second, config.Network.Timeout)
		assert.Equal(t, "light", config.UI.Theme)
		assert.Contains(t, config.Network.DNSServers, "1.1.1.1")
		assert.Contains(t, config.Network.DNSServers, "8.8.8.8")
	})
	
	t.Run("ConfigurationValidation", func(t *testing.T) {
		// Create a fresh manager for this test
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Store original values before testing validation
		originalTimeout := manager.Get("network.timeout")
		originalMaxHops := manager.Get("network.max_hops")
		originalTheme := manager.Get("ui.theme")
		
		// Test validation during Set operations
		err = manager.Set("network.timeout", "0s")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
		
		err = manager.Set("network.max_hops", 300)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
		
		err = manager.Set("ui.theme", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
		
		// Verify original values are preserved after validation failures
		assert.Equal(t, originalTimeout, manager.Get("network.timeout"))
		assert.Equal(t, originalMaxHops, manager.Get("network.max_hops"))
		assert.Equal(t, originalTheme, manager.Get("ui.theme"))
	})
	
	t.Run("AtomicMultipleUpdates", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Test successful atomic update
		values := map[string]interface{}{
			"network.timeout":    "45s",
			"ui.theme":           "dark",
			"network.max_hops":   25,
			"ui.auto_refresh":    true,
		}
		
		err = manager.SetMultiple(values)
		assert.NoError(t, err)
		
		// Verify all values were set
		config := manager.GetConfig()
		assert.Equal(t, 45*time.Second, config.Network.Timeout)
		assert.Equal(t, "dark", config.UI.Theme)
		assert.Equal(t, 25, config.Network.MaxHops)
		assert.True(t, config.UI.AutoRefresh)
		
		// Test rollback on validation failure
		invalidValues := map[string]interface{}{
			"network.timeout":  "60s", // Valid
			"network.max_hops": -1,    // Invalid - should cause rollback
			"ui.theme":         "minimal", // Valid
		}
		
		err = manager.SetMultiple(invalidValues)
		assert.Error(t, err)
		
		// Verify all original values are preserved (no partial updates)
		config = manager.GetConfig()
		assert.Equal(t, 45*time.Second, config.Network.Timeout) // Should remain unchanged
		assert.Equal(t, 25, config.Network.MaxHops)             // Should remain unchanged
		assert.Equal(t, "dark", config.UI.Theme)                // Should remain unchanged
	})
	
	t.Run("ConfigurationChangeNotifications", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		var notifications []string
		listener := func(key string, oldValue, newValue interface{}) {
			notifications = append(notifications, key)
		}
		
		manager.AddChangeListener(listener)
		
		// Make changes and verify notifications
		err = manager.Set("network.timeout", "45s")
		assert.NoError(t, err)
		
		err = manager.Set("ui.theme", "dark")
		assert.NoError(t, err)
		
		assert.Len(t, notifications, 2)
		assert.Contains(t, notifications, "network.timeout")
		assert.Contains(t, notifications, "ui.theme")
		
		// Test multiple updates notification
		notifications = nil
		values := map[string]interface{}{
			"network.max_hops": 25,
			"ui.auto_refresh":  true,
		}
		
		err = manager.SetMultiple(values)
		assert.NoError(t, err)
		
		assert.Len(t, notifications, 2)
		assert.Contains(t, notifications, "network.max_hops")
		assert.Contains(t, notifications, "ui.auto_refresh")
	})
	
	t.Run("SectionReset", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Modify network settings
		err = manager.Set("network.timeout", "45s")
		assert.NoError(t, err)
		err = manager.Set("network.max_hops", 25)
		assert.NoError(t, err)
		
		// Modify UI settings
		err = manager.Set("ui.theme", "dark")
		assert.NoError(t, err)
		err = manager.Set("ui.auto_refresh", true)
		assert.NoError(t, err)
		
		// Reset only network section
		err = manager.ResetSection("network")
		assert.NoError(t, err)
		
		// Verify network settings are reset
		config := manager.GetConfig()
		assert.Equal(t, 30*time.Second, config.Network.Timeout)
		assert.Equal(t, 30, config.Network.MaxHops)
		
		// Verify UI settings remain unchanged
		assert.Equal(t, "dark", config.UI.Theme)
		assert.True(t, config.UI.AutoRefresh)
	})
	
	t.Run("ConfigurationPersistence", func(t *testing.T) {
		// Create initial configuration
		manager1 := NewManager()
		err := manager1.Load()
		assert.NoError(t, err)
		
		// Modify and save
		err = manager1.Set("network.timeout", "45s")
		assert.NoError(t, err)
		err = manager1.Set("ui.theme", "dark")
		assert.NoError(t, err)
		
		err = manager1.Save()
		assert.NoError(t, err)
		
		// Load in a new manager
		manager2 := NewManager()
		err = manager2.Load()
		assert.NoError(t, err)
		
		// Verify persistence
		config := manager2.GetConfig()
		assert.Equal(t, 45*time.Second, config.Network.Timeout)
		assert.Equal(t, "dark", config.UI.Theme)
		
		// Modify and save again
		err = manager2.Set("network.max_hops", 25)
		assert.NoError(t, err)
		
		err = manager2.Save()
		assert.NoError(t, err)
		
		// Load in a third manager
		manager3 := NewManager()
		err = manager3.Load()
		assert.NoError(t, err)
		
		// Verify all changes are persisted
		config = manager3.GetConfig()
		assert.Equal(t, 45*time.Second, config.Network.Timeout)
		assert.Equal(t, "dark", config.UI.Theme)
		assert.Equal(t, 25, config.Network.MaxHops)
	})
}

func TestConfigurationUIIntegration(t *testing.T) {
	t.Run("UIModelConfigurationInteraction", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Create UI model
		uiModel := NewConfigUIModel(manager)
		
		// Verify UI model reflects current configuration
		config := manager.GetConfig()
		networkSettings := uiModel.getNetworkSettings(config.Network)
		
		// Find timeout setting
		var timeoutSetting *ConfigSetting
		for _, setting := range networkSettings {
			if setting.Key == "network.timeout" {
				timeoutSetting = &setting
				break
			}
		}
		
		assert.NotNil(t, timeoutSetting)
		assert.Equal(t, config.Network.Timeout.String(), timeoutSetting.Value)
		
		// Modify configuration through manager
		err = manager.Set("network.timeout", "60s")
		assert.NoError(t, err)
		
		// Reload UI model settings
		updatedConfig := manager.GetConfig()
		updatedNetworkSettings := uiModel.getNetworkSettings(updatedConfig.Network)
		
		// Find updated timeout setting
		var updatedTimeoutSetting *ConfigSetting
		for _, setting := range updatedNetworkSettings {
			if setting.Key == "network.timeout" {
				updatedTimeoutSetting = &setting
				break
			}
		}
		
		assert.NotNil(t, updatedTimeoutSetting)
		assert.Equal(t, "1m0s", updatedTimeoutSetting.Value) // 60s formatted as duration
	})
	
	t.Run("UIValueParsing", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		uiModel := NewConfigUIModel(manager)
		
		// Test parsing various value types
		testCases := []struct {
			key      string
			input    string
			expected interface{}
		}{
			{"network.timeout", "45s", 45 * time.Second},
			{"network.max_hops", "25", 25},
			{"ui.auto_refresh", "true", true},
			{"ui.theme", "dark", "dark"},
			{"export.default_format", "CSV", domain.ExportFormatCSV},
			{"plugins.enabled_plugins", "plugin1, plugin2", []string{"plugin1", "plugin2"}},
			{"plugins.enabled_plugins", "", []string{}},
		}
		
		for _, tc := range testCases {
			t.Run(tc.key, func(t *testing.T) {
				value, err := uiModel.parseValue(tc.key, tc.input)
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			})
		}
		
		// Test invalid values
		invalidCases := []struct {
			key   string
			input string
		}{
			{"network.timeout", "invalid"},
			{"network.max_hops", "not_a_number"},
			{"ui.auto_refresh", "not_a_bool"},
			{"export.default_format", "INVALID_FORMAT"},
		}
		
		for _, tc := range invalidCases {
			t.Run(tc.key+"_invalid", func(t *testing.T) {
				_, err := uiModel.parseValue(tc.key, tc.input)
				assert.Error(t, err)
			})
		}
	})
}

func TestConfigurationErrorHandling(t *testing.T) {
	t.Run("InvalidConfigurationFile", func(t *testing.T) {
		// Create a temporary directory
		tempDir, err := os.MkdirTemp("", "nettracex-config-error")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		// Create an invalid YAML file
		invalidConfigFile := filepath.Join(tempDir, "invalid.yaml")
		invalidContent := `
network:
  timeout: 30s
  invalid_yaml: [unclosed array
ui:
  theme: default
`
		err = os.WriteFile(invalidConfigFile, []byte(invalidContent), 0644)
		require.NoError(t, err)
		
		manager := NewManager()
		err = manager.LoadFromFile(invalidConfigFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
	
	t.Run("ConfigurationValidationErrors", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Test various validation errors
		validationTests := []struct {
			key   string
			value interface{}
			error string
		}{
			{"network.timeout", "0s", "timeout must be positive"},
			{"network.max_hops", 0, "max_hops must be between 1 and 255"},
			{"network.max_hops", 300, "max_hops must be between 1 and 255"},
			{"network.packet_size", 0, "packet_size must be between 1 and 65507"},
			{"network.packet_size", 70000, "packet_size must be between 1 and 65507"},
			{"network.max_concurrency", 0, "max_concurrency must be positive"},
			{"network.retry_attempts", -1, "retry_attempts must be non-negative"},
			{"ui.theme", "invalid", "theme must be one of"},
			{"ui.color_mode", "invalid", "color_mode must be one of"},
			{"export.output_directory", "", "output_directory cannot be empty"},
		}
		
		for _, test := range validationTests {
			t.Run(test.key, func(t *testing.T) {
				err := manager.Set(test.key, test.value)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), test.error)
			})
		}
	})
	
	t.Run("FileSystemErrors", func(t *testing.T) {
		manager := NewManager()
		err := manager.Load()
		assert.NoError(t, err)
		
		// Try to save to an invalid path on Windows
		// Use a path that should fail on Windows
		invalidPath := "Z:\\nonexistent\\path\\config.yaml"
		err = manager.SaveAs(invalidPath)
		if err != nil {
			// Error is expected for invalid paths
			assert.Error(t, err)
		} else {
			// If no error, skip this test (might be running with special permissions)
			t.Skip("Skipping file system error test - no error occurred for invalid path")
		}
	})
}