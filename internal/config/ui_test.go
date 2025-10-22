package config

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigUIModel(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	
	assert.NotNil(t, model)
	assert.Equal(t, stateSelectingSection, model.state)
	assert.NotNil(t, model.manager)
	assert.NotNil(t, model.sections)
	assert.NotNil(t, model.settings)
	assert.NotNil(t, model.editor)
	
	// Verify sections are loaded
	assert.Greater(t, len(model.sections.Items()), 0)
}

func TestConfigUIModelInit(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	cmd := model.Init()
	
	assert.Nil(t, cmd)
}

func TestConfigUIModelWindowResize(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	
	// Test window resize
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, cmd := model.Update(msg)
	
	assert.Nil(t, cmd)
	configModel := updatedModel.(*ConfigUIModel)
	assert.Equal(t, 100, configModel.width)
	assert.Equal(t, 50, configModel.height)
}

func TestConfigUIModelSectionNavigation(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	model.width = 100
	model.height = 50
	
	// Test entering a section
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	
	configModel := updatedModel.(*ConfigUIModel)
	assert.Equal(t, stateSelectingSetting, configModel.state)
	
	// Test going back to sections
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = configModel.Update(escMsg)
	
	configModel = updatedModel.(*ConfigUIModel)
	assert.Equal(t, stateSelectingSection, configModel.state)
}

func TestConfigUIModelSettingEditing(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	model.width = 100
	model.height = 50
	
	// Navigate to settings
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	configModel := updatedModel.(*ConfigUIModel)
	
	// Start editing a setting
	updatedModel, _ = configModel.Update(enterMsg)
	configModel = updatedModel.(*ConfigUIModel)
	assert.Equal(t, stateEditingValue, configModel.state)
	assert.NotEmpty(t, configModel.currentKey)
	
	// Cancel editing
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = configModel.Update(escMsg)
	configModel = updatedModel.(*ConfigUIModel)
	assert.Equal(t, stateSelectingSetting, configModel.state)
	assert.Empty(t, configModel.currentKey)
}

func TestConfigUIModelView(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	model.width = 100
	model.height = 50
	
	// Test view in different states
	view := model.View()
	assert.Contains(t, view, "NetTraceX Configuration")
	
	// Navigate to settings and test view
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := model.Update(enterMsg)
	configModel := updatedModel.(*ConfigUIModel)
	
	view = configModel.View()
	assert.Contains(t, view, "Back to sections")
	
	// Start editing and test view
	updatedModel, _ = configModel.Update(enterMsg)
	configModel = updatedModel.(*ConfigUIModel)
	
	view = configModel.View()
	assert.Contains(t, view, "Editing:")
}

func TestConfigUIModelParseValue(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	
	// Test duration parsing
	value, err := model.parseValue("network.timeout", "45s")
	assert.NoError(t, err)
	assert.Equal(t, 45*time.Second, value)
	
	// Test integer parsing
	value, err = model.parseValue("network.max_hops", "25")
	assert.NoError(t, err)
	assert.Equal(t, 25, value)
	
	// Test boolean parsing
	value, err = model.parseValue("ui.auto_refresh", "true")
	assert.NoError(t, err)
	assert.Equal(t, true, value)
	
	// Test export format parsing
	value, err = model.parseValue("export.default_format", "CSV")
	assert.NoError(t, err)
	assert.Equal(t, domain.ExportFormatCSV, value)
	
	// Test string array parsing
	value, err = model.parseValue("plugins.enabled_plugins", "plugin1, plugin2, plugin3")
	assert.NoError(t, err)
	expected := []string{"plugin1", "plugin2", "plugin3"}
	assert.Equal(t, expected, value)
	
	// Test empty string array
	value, err = model.parseValue("plugins.enabled_plugins", "")
	assert.NoError(t, err)
	assert.Equal(t, []string{}, value)
	
	// Test string parsing (default case)
	value, err = model.parseValue("network.user_agent", "NetTraceX/2.0")
	assert.NoError(t, err)
	assert.Equal(t, "NetTraceX/2.0", value)
	
	// Test invalid duration
	_, err = model.parseValue("network.timeout", "invalid")
	assert.Error(t, err)
	
	// Test invalid integer
	_, err = model.parseValue("network.max_hops", "invalid")
	assert.Error(t, err)
	
	// Test invalid boolean
	_, err = model.parseValue("ui.auto_refresh", "invalid")
	assert.Error(t, err)
	
	// Test invalid export format
	_, err = model.parseValue("export.default_format", "invalid")
	assert.Error(t, err)
}

func TestConfigUIModelGetSettings(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	config := manager.GetConfig()
	
	// Test network settings
	networkSettings := model.getNetworkSettings(config.Network)
	assert.Greater(t, len(networkSettings), 0)
	
	// Verify specific settings exist
	var timeoutSetting *ConfigSetting
	for _, setting := range networkSettings {
		if setting.Key == "network.timeout" {
			timeoutSetting = &setting
			break
		}
	}
	assert.NotNil(t, timeoutSetting)
	assert.Equal(t, "Timeout", timeoutSetting.Name)
	assert.Equal(t, "duration", timeoutSetting.Type)
	
	// Test UI settings
	uiSettings := model.getUISettings(config.UI)
	assert.Greater(t, len(uiSettings), 0)
	
	// Verify theme setting
	var themeSetting *ConfigSetting
	for _, setting := range uiSettings {
		if setting.Key == "ui.theme" {
			themeSetting = &setting
			break
		}
	}
	assert.NotNil(t, themeSetting)
	assert.Equal(t, "Theme", themeSetting.Name)
	assert.Equal(t, "enum", themeSetting.Type)
	assert.Contains(t, themeSetting.Options, "default")
	assert.Contains(t, themeSetting.Options, "dark")
	
	// Test plugin settings
	pluginSettings := model.getPluginSettings(config.Plugins)
	assert.Greater(t, len(pluginSettings), 0)
	
	// Test export settings
	exportSettings := model.getExportSettings(config.Export)
	assert.Greater(t, len(exportSettings), 0)
	
	// Test logging settings
	loggingSettings := model.getLoggingSettings(config.Logging)
	assert.Greater(t, len(loggingSettings), 0)
}

func TestConfigUIModelSetMessage(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	
	// Test setting different message types
	model.setMessage("Success message", messageTypeSuccess)
	assert.Equal(t, "Success message", model.message)
	assert.Equal(t, messageTypeSuccess, model.messageType)
	
	model.setMessage("Error message", messageTypeError)
	assert.Equal(t, "Error message", model.message)
	assert.Equal(t, messageTypeError, model.messageType)
	
	model.setMessage("Info message", messageTypeInfo)
	assert.Equal(t, "Info message", model.message)
	assert.Equal(t, messageTypeInfo, model.messageType)
}

func TestConfigSectionFilterValue(t *testing.T) {
	section := ConfigSection{
		Name:        "Network",
		Description: "Network settings",
	}
	
	assert.Equal(t, "Network", section.FilterValue())
}

func TestConfigSettingFilterValue(t *testing.T) {
	setting := ConfigSetting{
		Key:         "network.timeout",
		Name:        "Timeout",
		Description: "Network timeout",
		Value:       "30s",
		Type:        "duration",
	}
	
	assert.Equal(t, "Timeout", setting.FilterValue())
}

func TestConfigUIModelSaveConfiguration(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	model.width = 100
	model.height = 50
	
	// Test save key in section selection state
	saveMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updatedModel, _ := model.Update(saveMsg)
	
	configModel := updatedModel.(*ConfigUIModel)
	// Should show success message (assuming save succeeds with in-memory config)
	assert.Contains(t, configModel.message, "saved")
}

func TestConfigUIModelResetSection(t *testing.T) {
	manager := NewManager()
	err := manager.Load()
	assert.NoError(t, err)
	
	// Modify a value first
	err = manager.Set("network.timeout", "45s")
	assert.NoError(t, err)
	
	model := NewConfigUIModel(manager)
	model.width = 100
	model.height = 50
	
	// Test reset key in section selection state
	resetMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	updatedModel, _ := model.Update(resetMsg)
	
	configModel := updatedModel.(*ConfigUIModel)
	// Should show success message
	assert.Contains(t, configModel.message, "reset")
	
	// Verify the value was actually reset
	assert.Equal(t, "30s", manager.Get("network.timeout"))
}