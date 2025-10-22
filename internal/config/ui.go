// Package config provides configuration UI components
package config

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// ConfigUIModel represents the configuration UI model
type ConfigUIModel struct {
	manager     *Manager
	state       configUIState
	sections    list.Model
	settings    list.Model
	editor      textinput.Model
	currentKey  string
	width       int
	height      int
	styles      configUIStyles
	keyMap      configUIKeyMap
	message     string
	messageType messageType
}

type configUIState int

const (
	stateSelectingSection configUIState = iota
	stateSelectingSetting
	stateEditingValue
)

type messageType int

const (
	messageTypeNone messageType = iota
	messageTypeSuccess
	messageTypeError
	messageTypeInfo
)

type configUIKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Escape   key.Binding
	Save     key.Binding
	Reset    key.Binding
	Help     key.Binding
	Quit     key.Binding
}

type configUIStyles struct {
	titleStyle       lipgloss.Style
	sectionStyle     lipgloss.Style
	settingStyle     lipgloss.Style
	valueStyle       lipgloss.Style
	selectedStyle    lipgloss.Style
	errorStyle       lipgloss.Style
	successStyle     lipgloss.Style
	infoStyle        lipgloss.Style
	helpStyle        lipgloss.Style
	borderStyle      lipgloss.Style
}

// ConfigSection represents a configuration section for the UI
type ConfigSection struct {
	Name        string
	Description string
	Settings    []ConfigSetting
}

// ConfigSetting represents a single configuration setting
type ConfigSetting struct {
	Key         string
	Name        string
	Description string
	Value       interface{}
	Type        string
	Options     []string // For enum-like settings
}

// ConfigSettingDelegate is a custom list delegate for configuration settings
type ConfigSettingDelegate struct {
	styles configUIStyles
}

// NewConfigSettingDelegate creates a new configuration setting delegate
func NewConfigSettingDelegate(styles configUIStyles) *ConfigSettingDelegate {
	return &ConfigSettingDelegate{
		styles: styles,
	}
}

// Height returns the height of a list item
func (d *ConfigSettingDelegate) Height() int {
	return 2 // Two lines: setting name + current value
}

// Spacing returns the spacing between list items
func (d *ConfigSettingDelegate) Spacing() int {
	return 1
}

// Update handles updates for the delegate
func (d *ConfigSettingDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders a configuration setting with its current value
func (d *ConfigSettingDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	setting, ok := item.(ConfigSetting)
	if !ok {
		return
	}

	// Check if this item is selected
	isSelected := index == m.Index()
	
	// Setting name style
	nameStyle := d.styles.settingStyle
	if isSelected {
		nameStyle = d.styles.selectedStyle
	}
	
	// Current value style
	valueStyle := d.styles.valueStyle
	if isSelected {
		valueStyle = d.styles.selectedStyle
	}
	
	// Format the current value for display
	currentValue := fmt.Sprintf("%v", setting.Value)
	if len(currentValue) > 50 {
		currentValue = currentValue[:47] + "..."
	}
	
	// Render setting name and description
	settingLine := nameStyle.Render(fmt.Sprintf("  %s", setting.Name))
	if setting.Description != "" {
		settingLine += " - " + nameStyle.Copy().Faint(true).Render(setting.Description)
	}
	
	// Render current value
	valueLine := valueStyle.Render(fmt.Sprintf("    Current: %s", currentValue))
	
	// Write both lines
	fmt.Fprint(w, settingLine)
	fmt.Fprint(w, "\n")
	fmt.Fprint(w, valueLine)
}

// NewConfigUIModel creates a new configuration UI model
func NewConfigUIModel(manager *Manager) *ConfigUIModel {
	// Initialize key bindings
	keyMap := configUIKeyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "back")),
		Right:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "select")),
		Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "edit/confirm")),
		Escape: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back/cancel")),
		Save:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save config")),
		Reset:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset section")),
		Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:   key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}

	// Initialize styles
	styles := configUIStyles{
		titleStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
		sectionStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
		settingStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		valueStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		selectedStyle: lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("230")),
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		successStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("46")),
		infoStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("33")),
		helpStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		borderStyle:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")),
	}

	// Create sections list
	sections := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	sections.Title = "Configuration Sections"
	sections.SetShowStatusBar(false)
	sections.SetFilteringEnabled(false)

	// Create settings list with custom delegate to show current values
	settingsDelegate := NewConfigSettingDelegate(styles)
	settings := list.New([]list.Item{}, settingsDelegate, 0, 0)
	settings.Title = "Settings"
	settings.SetShowStatusBar(false)
	settings.SetFilteringEnabled(false)

	// Create text input for editing values
	editor := textinput.New()
	editor.Placeholder = "Enter value..."
	editor.CharLimit = 256

	model := &ConfigUIModel{
		manager:  manager,
		state:    stateSelectingSection,
		sections: sections,
		settings: settings,
		editor:   editor,
		keyMap:   keyMap,
		styles:   styles,
	}

	model.loadSections()
	return model
}

// loadSections loads configuration sections into the UI with current values
func (m *ConfigUIModel) loadSections() {
	config := m.manager.GetConfig()
	
	sections := []list.Item{
		ConfigSection{
			Name:        "Network",
			Description: "Network operation settings",
			Settings:    m.getNetworkSettings(config.Network),
		},
		ConfigSection{
			Name:        "UI",
			Description: "User interface preferences",
			Settings:    m.getUISettings(config.UI),
		},
		ConfigSection{
			Name:        "Plugins",
			Description: "Plugin configuration",
			Settings:    m.getPluginSettings(config.Plugins),
		},
		ConfigSection{
			Name:        "Export",
			Description: "Export and save settings",
			Settings:    m.getExportSettings(config.Export),
		},
		ConfigSection{
			Name:        "Logging",
			Description: "Logging configuration",
			Settings:    m.getLoggingSettings(config.Logging),
		},
	}
	
	m.sections.SetItems(sections)
}

// getNetworkSettings returns network configuration settings
func (m *ConfigUIModel) getNetworkSettings(config domain.NetworkConfig) []ConfigSetting {
	return []ConfigSetting{
		{
			Key:         "network.timeout",
			Name:        "Timeout",
			Description: "Network operation timeout",
			Value:       config.Timeout.String(),
			Type:        "duration",
		},
		{
			Key:         "network.max_hops",
			Name:        "Max Hops",
			Description: "Maximum number of hops for traceroute",
			Value:       config.MaxHops,
			Type:        "int",
		},
		{
			Key:         "network.packet_size",
			Name:        "Packet Size",
			Description: "Default packet size in bytes",
			Value:       config.PacketSize,
			Type:        "int",
		},
		{
			Key:         "network.user_agent",
			Name:        "User Agent",
			Description: "User agent string for HTTP requests",
			Value:       config.UserAgent,
			Type:        "string",
		},
		{
			Key:         "network.max_concurrency",
			Name:        "Max Concurrency",
			Description: "Maximum concurrent network operations",
			Value:       config.MaxConcurrency,
			Type:        "int",
		},
		{
			Key:         "network.retry_attempts",
			Name:        "Retry Attempts",
			Description: "Number of retry attempts for failed operations",
			Value:       config.RetryAttempts,
			Type:        "int",
		},
		{
			Key:         "network.retry_delay",
			Name:        "Retry Delay",
			Description: "Delay between retry attempts",
			Value:       config.RetryDelay.String(),
			Type:        "duration",
		},
	}
}

// getUISettings returns UI configuration settings
func (m *ConfigUIModel) getUISettings(config domain.UIConfig) []ConfigSetting {
	return []ConfigSetting{
		{
			Key:         "ui.theme",
			Name:        "Theme",
			Description: "UI color theme",
			Value:       config.Theme,
			Type:        "enum",
			Options:     []string{"default", "dark", "light", "minimal"},
		},
		{
			Key:         "ui.animation_speed",
			Name:        "Animation Speed",
			Description: "Speed of UI animations",
			Value:       config.AnimationSpeed.String(),
			Type:        "duration",
		},
		{
			Key:         "ui.auto_refresh",
			Name:        "Auto Refresh",
			Description: "Enable automatic refresh of results",
			Value:       config.AutoRefresh,
			Type:        "bool",
		},
		{
			Key:         "ui.refresh_interval",
			Name:        "Refresh Interval",
			Description: "Interval for automatic refresh",
			Value:       config.RefreshInterval.String(),
			Type:        "duration",
		},
		{
			Key:         "ui.show_help",
			Name:        "Show Help",
			Description: "Show help text in UI",
			Value:       config.ShowHelp,
			Type:        "bool",
		},
		{
			Key:         "ui.color_mode",
			Name:        "Color Mode",
			Description: "Color output mode",
			Value:       config.ColorMode,
			Type:        "enum",
			Options:     []string{"auto", "always", "never"},
		},
	}
}

// getPluginSettings returns plugin configuration settings
func (m *ConfigUIModel) getPluginSettings(config domain.PluginConfig) []ConfigSetting {
	return []ConfigSetting{
		{
			Key:         "plugins.enabled_plugins",
			Name:        "Enabled Plugins",
			Description: "List of enabled plugins",
			Value:       strings.Join(config.EnabledPlugins, ", "),
			Type:        "string_array",
		},
		{
			Key:         "plugins.disabled_plugins",
			Name:        "Disabled Plugins",
			Description: "List of disabled plugins",
			Value:       strings.Join(config.DisabledPlugins, ", "),
			Type:        "string_array",
		},
		{
			Key:         "plugins.plugin_paths",
			Name:        "Plugin Paths",
			Description: "Directories to search for plugins",
			Value:       strings.Join(config.PluginPaths, ", "),
			Type:        "string_array",
		},
	}
}

// getExportSettings returns export configuration settings
func (m *ConfigUIModel) getExportSettings(config domain.ExportConfig) []ConfigSetting {
	formatNames := []string{"JSON", "CSV", "Text"}
	return []ConfigSetting{
		{
			Key:         "export.default_format",
			Name:        "Default Format",
			Description: "Default export format",
			Value:       formatNames[config.DefaultFormat],
			Type:        "enum",
			Options:     formatNames,
		},
		{
			Key:         "export.output_directory",
			Name:        "Output Directory",
			Description: "Default directory for exported files",
			Value:       config.OutputDirectory,
			Type:        "string",
		},
		{
			Key:         "export.include_metadata",
			Name:        "Include Metadata",
			Description: "Include metadata in exported files",
			Value:       config.IncludeMetadata,
			Type:        "bool",
		},
		{
			Key:         "export.compression",
			Name:        "Compression",
			Description: "Enable compression for exported files",
			Value:       config.Compression,
			Type:        "bool",
		},
	}
}

// getLoggingSettings returns logging configuration settings
func (m *ConfigUIModel) getLoggingSettings(config domain.LoggingConfig) []ConfigSetting {
	return []ConfigSetting{
		{
			Key:         "logging.level",
			Name:        "Log Level",
			Description: "Minimum log level to output",
			Value:       config.Level,
			Type:        "enum",
			Options:     []string{"debug", "info", "warn", "error", "fatal"},
		},
		{
			Key:         "logging.format",
			Name:        "Log Format",
			Description: "Log output format",
			Value:       config.Format,
			Type:        "enum",
			Options:     []string{"text", "json"},
		},
		{
			Key:         "logging.output",
			Name:        "Log Output",
			Description: "Log output destination",
			Value:       config.Output,
			Type:        "enum",
			Options:     []string{"stdout", "stderr", "file"},
		},
		{
			Key:         "logging.max_size",
			Name:        "Max Size (MB)",
			Description: "Maximum log file size in MB",
			Value:       config.MaxSize,
			Type:        "int",
		},
		{
			Key:         "logging.max_backups",
			Name:        "Max Backups",
			Description: "Maximum number of log file backups",
			Value:       config.MaxBackups,
			Type:        "int",
		},
		{
			Key:         "logging.max_age",
			Name:        "Max Age (days)",
			Description: "Maximum age of log files in days",
			Value:       config.MaxAge,
			Type:        "int",
		},
	}
}

// Init implements tea.Model
func (m *ConfigUIModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *ConfigUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sections.SetSize(msg.Width/2-2, msg.Height-6)
		m.settings.SetSize(msg.Width/2-2, msg.Height-6)
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateSelectingSection:
			switch {
			case key.Matches(msg, m.keyMap.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keyMap.Enter), key.Matches(msg, m.keyMap.Right):
				if section, ok := m.sections.SelectedItem().(ConfigSection); ok {
					m.loadSettings(section.Settings)
					m.state = stateSelectingSetting
				}
			case key.Matches(msg, m.keyMap.Save):
				if err := m.manager.Save(); err != nil {
					m.setMessage("Failed to save configuration: "+err.Error(), messageTypeError)
				} else {
					m.setMessage("Configuration saved successfully", messageTypeSuccess)
				}
			case key.Matches(msg, m.keyMap.Reset):
				if section, ok := m.sections.SelectedItem().(ConfigSection); ok {
					sectionName := strings.ToLower(section.Name)
					if err := m.manager.ResetSection(sectionName); err != nil {
						m.setMessage("Failed to reset section: "+err.Error(), messageTypeError)
					} else {
						m.setMessage("Section reset to defaults", messageTypeSuccess)
						m.loadSections() // Reload sections to show updated values
						
						// Also reload the current section's settings if we're viewing them
						if m.state == stateSelectingSetting {
							config := m.manager.GetConfig()
							var freshSettings []ConfigSetting
							
							switch section.Name {
							case "Network":
								freshSettings = m.getNetworkSettings(config.Network)
							case "UI":
								freshSettings = m.getUISettings(config.UI)
							case "Plugins":
								freshSettings = m.getPluginSettings(config.Plugins)
							case "Export":
								freshSettings = m.getExportSettings(config.Export)
							case "Logging":
								freshSettings = m.getLoggingSettings(config.Logging)
							}
							
							m.loadSettings(freshSettings)
						}
					}
				}
			}
			m.sections, cmd = m.sections.Update(msg)
			cmds = append(cmds, cmd)

		case stateSelectingSetting:
			switch {
			case key.Matches(msg, m.keyMap.Escape), key.Matches(msg, m.keyMap.Left):
				m.state = stateSelectingSection
			case key.Matches(msg, m.keyMap.Enter), key.Matches(msg, m.keyMap.Right):
				if setting, ok := m.settings.SelectedItem().(ConfigSetting); ok {
					m.startEditing(setting)
				}
			case key.Matches(msg, m.keyMap.Save):
				if err := m.manager.Save(); err != nil {
					m.setMessage("Failed to save configuration: "+err.Error(), messageTypeError)
				} else {
					m.setMessage("Configuration saved successfully", messageTypeSuccess)
				}
			}
			m.settings, cmd = m.settings.Update(msg)
			cmds = append(cmds, cmd)

		case stateEditingValue:
			switch {
			case key.Matches(msg, m.keyMap.Escape):
				m.cancelEditing()
			case key.Matches(msg, m.keyMap.Enter):
				m.saveCurrentValue()
			}
			m.editor, cmd = m.editor.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *ConfigUIModel) View() string {
	if m.width == 0 {
		return "Loading configuration..."
	}

	var content strings.Builder
	
	// Title
	title := m.styles.titleStyle.Render("NetTraceX Configuration")
	content.WriteString(title + "\n\n")

	// Message
	if m.message != "" {
		var messageStyle lipgloss.Style
		switch m.messageType {
		case messageTypeSuccess:
			messageStyle = m.styles.successStyle
		case messageTypeError:
			messageStyle = m.styles.errorStyle
		case messageTypeInfo:
			messageStyle = m.styles.infoStyle
		}
		content.WriteString(messageStyle.Render(m.message) + "\n\n")
	}

	// Main content
	switch m.state {
	case stateSelectingSection:
		content.WriteString(m.renderSectionSelection())
	case stateSelectingSetting:
		content.WriteString(m.renderSettingSelection())
	case stateEditingValue:
		content.WriteString(m.renderValueEditor())
	}

	// Help
	content.WriteString("\n" + m.renderHelp())

	return content.String()
}

// renderSectionSelection renders the section selection view
func (m *ConfigUIModel) renderSectionSelection() string {
	return m.styles.borderStyle.Width(m.width - 4).Render(m.sections.View())
}

// renderSettingSelection renders the setting selection view
func (m *ConfigUIModel) renderSettingSelection() string {
	var content strings.Builder
	
	// Back button hint
	content.WriteString(m.styles.helpStyle.Render("← Back to sections") + "\n\n")
	
	// Settings list
	content.WriteString(m.styles.borderStyle.Width(m.width - 4).Render(m.settings.View()))
	
	return content.String()
}

// renderValueEditor renders the value editing view
func (m *ConfigUIModel) renderValueEditor() string {
	var content strings.Builder
	
	content.WriteString(m.styles.helpStyle.Render("Editing: "+m.currentKey) + "\n\n")
	content.WriteString("New value:\n")
	content.WriteString(m.editor.View() + "\n\n")
	content.WriteString(m.styles.helpStyle.Render("Press Enter to save, Esc to cancel"))
	
	return content.String()
}

// renderHelp renders the help text
func (m *ConfigUIModel) renderHelp() string {
	var help strings.Builder
	
	switch m.state {
	case stateSelectingSection:
		help.WriteString("Enter/→: Select section • s: Save config • r: Reset section • q: Quit")
	case stateSelectingSetting:
		help.WriteString("Enter/→: Edit setting • ←/Esc: Back • s: Save config")
	case stateEditingValue:
		help.WriteString("Enter: Save • Esc: Cancel")
	}
	
	return m.styles.helpStyle.Render(help.String())
}

// loadSettings loads settings for the current section with current values
func (m *ConfigUIModel) loadSettings(settings []ConfigSetting) {
	items := make([]list.Item, len(settings))
	for i, setting := range settings {
		// Update the setting with current value from configuration manager
		currentValue := m.manager.Get(setting.Key)
		updatedSetting := setting
		updatedSetting.Value = currentValue
		items[i] = updatedSetting
	}
	m.settings.SetItems(items)
}

// startEditing starts editing a configuration value
func (m *ConfigUIModel) startEditing(setting ConfigSetting) {
	m.currentKey = setting.Key
	m.editor.SetValue(fmt.Sprintf("%v", setting.Value))
	m.editor.Focus()
	m.state = stateEditingValue
}

// cancelEditing cancels the current editing operation
func (m *ConfigUIModel) cancelEditing() {
	m.editor.Blur()
	m.editor.SetValue("")
	m.currentKey = ""
	m.state = stateSelectingSetting
}

// saveCurrentValue saves the currently edited value
func (m *ConfigUIModel) saveCurrentValue() {
	value := m.editor.Value()
	
	// Parse value based on the setting type
	parsedValue, err := m.parseValue(m.currentKey, value)
	if err != nil {
		m.setMessage("Invalid value: "+err.Error(), messageTypeError)
		return
	}
	
	// Set the configuration value
	if err := m.manager.Set(m.currentKey, parsedValue); err != nil {
		m.setMessage("Failed to set value: "+err.Error(), messageTypeError)
		return
	}
	
	m.setMessage("Value updated successfully", messageTypeSuccess)
	m.cancelEditing()
	
	// Reload the current section to show updated values
	if section, ok := m.sections.SelectedItem().(ConfigSection); ok {
		// Get fresh configuration and reload settings
		config := m.manager.GetConfig()
		var freshSettings []ConfigSetting
		
		switch section.Name {
		case "Network":
			freshSettings = m.getNetworkSettings(config.Network)
		case "UI":
			freshSettings = m.getUISettings(config.UI)
		case "Plugins":
			freshSettings = m.getPluginSettings(config.Plugins)
		case "Export":
			freshSettings = m.getExportSettings(config.Export)
		case "Logging":
			freshSettings = m.getLoggingSettings(config.Logging)
		}
		
		m.loadSettings(freshSettings)
	}
}

// parseValue parses a string value based on the configuration key
func (m *ConfigUIModel) parseValue(key, value string) (interface{}, error) {
	switch {
	case strings.Contains(key, "timeout") || strings.Contains(key, "delay") || strings.Contains(key, "interval") || strings.Contains(key, "speed"):
		return time.ParseDuration(value)
	case key == "network.max_hops" || key == "network.packet_size" || key == "network.max_concurrency" || key == "network.retry_attempts" || 
		 key == "logging.max_size" || key == "logging.max_backups" || key == "logging.max_age":
		return strconv.Atoi(value)
	case strings.Contains(key, "auto_refresh") || strings.Contains(key, "show_help") || strings.Contains(key, "metadata") || strings.Contains(key, "compression"):
		return strconv.ParseBool(value)
	case strings.Contains(key, "default_format"):
		// Handle export format enum
		switch strings.ToLower(value) {
		case "json":
			return domain.ExportFormatJSON, nil
		case "csv":
			return domain.ExportFormatCSV, nil
		case "text":
			return domain.ExportFormatText, nil
		default:
			return nil, fmt.Errorf("invalid export format: %s", value)
		}
	case strings.Contains(key, "_plugins") || strings.Contains(key, "_paths") || strings.Contains(key, "dns_servers"):
		// Handle string arrays
		if value == "" {
			return []string{}, nil
		}
		parts := strings.Split(value, ",")
		for i, part := range parts {
			parts[i] = strings.TrimSpace(part)
		}
		return parts, nil
	default:
		return value, nil
	}
}

// setMessage sets a status message
func (m *ConfigUIModel) setMessage(message string, msgType messageType) {
	m.message = message
	m.messageType = msgType
}

// FilterValue implements list.Item for ConfigSection
func (c ConfigSection) FilterValue() string {
	return c.Name
}

// FilterValue implements list.Item for ConfigSetting
func (c ConfigSetting) FilterValue() string {
	return c.Name
}

// SetSize implements domain.TUIComponent
func (m *ConfigUIModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.sections.SetSize(width/2-2, height-6)
	m.settings.SetSize(width/2-2, height-6)
}

// SetTheme implements domain.TUIComponent
func (m *ConfigUIModel) SetTheme(theme domain.Theme) {
	// Update styles based on theme if needed
	if theme != nil {
		m.styles.titleStyle = m.styles.titleStyle.Foreground(lipgloss.Color(theme.GetColor("primary")))
		m.styles.selectedStyle = m.styles.selectedStyle.Background(lipgloss.Color(theme.GetColor("primary")))
		m.styles.errorStyle = m.styles.errorStyle.Foreground(lipgloss.Color(theme.GetColor("error")))
		m.styles.successStyle = m.styles.successStyle.Foreground(lipgloss.Color(theme.GetColor("success")))
	}
}

// Focus implements domain.TUIComponent
func (m *ConfigUIModel) Focus() {
	// Focus is handled internally based on state
}

// Blur implements domain.TUIComponent
func (m *ConfigUIModel) Blur() {
	// Blur is handled internally based on state
}