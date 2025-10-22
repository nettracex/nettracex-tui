// Package tui contains the terminal user interface components using Bubble Tea
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	configpkg "github.com/nettracex/nettracex-tui/internal/config"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// AppState represents the current state of the application
type AppState int

const (
	StateMainMenu AppState = iota
	StateNavigation
	StateDiagnostic
	StateSettings
	StateHelp
	StateExit
)

// MainModel represents the root application model
type MainModel struct {
	state         AppState
	navigation    *NavigationModel
	helpView      *HelpModel
	configView    *configpkg.ConfigUIModel
	activeView    tea.Model
	plugins       domain.PluginRegistry
	config        *domain.Config
	configManager *configpkg.Manager
	theme         domain.Theme
	width         int
	height        int
	keyMap        KeyMap
	quitting      bool
}

// KeyMap defines keyboard shortcuts for the application
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
	Tab      key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("PgDown", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "ctrl+a"),
			key.WithHelp("Home", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "ctrl+e"),
			key.WithHelp("End", "go to bottom"),
		),
	}
}

// NewMainModel creates a new main application model
func NewMainModel(plugins domain.PluginRegistry, config *domain.Config, configManager *configpkg.Manager, theme domain.Theme) *MainModel {
	nav := NewNavigationModel()
	help := NewHelpModel()
	configUI := configpkg.NewConfigUIModel(configManager)
	
	return &MainModel{
		state:         StateMainMenu,
		navigation:    nav,
		helpView:      help,
		configView:    configUI,
		activeView:    nav,
		plugins:       plugins,
		config:        config,
		configManager: configManager,
		theme:         theme,
		keyMap:        DefaultKeyMap(),
		quitting:      false,
	}
}

// Init implements tea.Model
func (m *MainModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// Update implements tea.Model
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update navigation model size
		if m.navigation != nil {
			m.navigation.SetSize(msg.Width, msg.Height)
		}
		
		// Update help view size
		if m.helpView != nil {
			m.helpView.SetSize(msg.Width, msg.Height)
		}
		
		// Update config view size
		if m.configView != nil {
			m.configView.SetSize(msg.Width, msg.Height)
		}
		
		// Update active view size
		if m.activeView != nil {
			if component, ok := m.activeView.(domain.TUIComponent); ok {
				component.SetSize(msg.Width, msg.Height)
			}
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			return m.handleBack()

		case key.Matches(msg, m.keyMap.Help):
			m.state = StateHelp
			m.activeView = m.helpView
			m.helpView.SetSize(m.width, m.height)
			m.helpView.Focus()
			return m, nil
		}

	case NavigationMsg:
		return m.handleNavigation(msg)
	}

	// Update the active view
	if m.activeView != nil {
		m.activeView, cmd = m.activeView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *MainModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Create the main layout
	header := m.renderHeader()
	content := m.renderContent()
	footer := m.renderFooter()

	// Calculate content height
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight

	// Style the content area
	contentStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Padding(0, 1)

	styledContent := contentStyle.Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, header, styledContent, footer)
}

// renderHeader renders the application header
func (m *MainModel) renderHeader() string {
	title := "NetTraceX - Network Diagnostic Toolkit"
	
	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	return headerStyle.Render(title)
}

// renderContent renders the main content area
func (m *MainModel) renderContent() string {
	if m.activeView == nil {
		return "No active view"
	}
	return m.activeView.View()
}

// renderFooter renders the application footer with key bindings
func (m *MainModel) renderFooter() string {
	var keys []string
	
	switch m.state {
	case StateMainMenu, StateNavigation:
		keys = []string{
			"↑/↓: navigate",
			"PgUp/PgDown: page",
			"Home/End: jump",
			"enter: select",
			"?: help",
			"q: quit",
		}
	case StateHelp:
		keys = []string{
			"↑/↓: scroll",
			"PgUp/PgDown: page",
			"Home/End: jump",
			"esc: back",
			"q: quit",
		}
	default:
		keys = []string{
			"esc: back",
			"?: help",
			"q: quit",
		}
	}

	footerStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))

	return footerStyle.Render(strings.Join(keys, " • "))
}

// handleBack handles the back navigation
func (m *MainModel) handleBack() (*MainModel, tea.Cmd) {
	switch m.state {
	case StateMainMenu:
		m.quitting = true
		return m, tea.Quit
	case StateDiagnostic, StateSettings, StateHelp:
		m.state = StateMainMenu
		m.activeView = m.navigation
		m.navigation.Focus()
		if m.helpView != nil {
			m.helpView.Blur()
		}
		return m, nil
	default:
		m.state = StateMainMenu
		m.activeView = m.navigation
		m.navigation.Focus()
		return m, nil
	}
}

// handleNavigation handles navigation messages
func (m *MainModel) handleNavigation(msg NavigationMsg) (*MainModel, tea.Cmd) {
	switch msg.Action {
	case NavigationActionSelect:
		// Handle menu selection
		if item, ok := msg.Data.(NavigationItem); ok {
			return m.selectNavigationItem(item)
		}
	case NavigationActionBack:
		return m.handleBack()
	}
	return m, nil
}

// selectNavigationItem handles navigation item selection
func (m *MainModel) selectNavigationItem(item NavigationItem) (*MainModel, tea.Cmd) {
	switch item.ID {
	case "whois":
		m.state = StateDiagnostic
		if tool, exists := m.plugins.Get("whois"); exists {
			diagnosticView := NewDiagnosticViewModel(tool)
			diagnosticView.SetSize(m.width, m.height)
			diagnosticView.SetTheme(m.theme)
			m.activeView = diagnosticView
		}
		return m, nil
	case "ping":
		m.state = StateDiagnostic
		if tool, exists := m.plugins.Get("ping"); exists {
			diagnosticView := NewDiagnosticViewModel(tool)
			diagnosticView.SetSize(m.width, m.height)
			diagnosticView.SetTheme(m.theme)
			m.activeView = diagnosticView
		}
		return m, nil
	case "traceroute":
		m.state = StateDiagnostic
		if tool, exists := m.plugins.Get("traceroute"); exists {
			diagnosticView := NewDiagnosticViewModel(tool)
			diagnosticView.SetSize(m.width, m.height)
			diagnosticView.SetTheme(m.theme)
			m.activeView = diagnosticView
		}
		return m, nil
	case "dns":
		m.state = StateDiagnostic
		if tool, exists := m.plugins.Get("dns"); exists {
			diagnosticView := NewDiagnosticViewModel(tool)
			diagnosticView.SetSize(m.width, m.height)
			diagnosticView.SetTheme(m.theme)
			m.activeView = diagnosticView
		}
		return m, nil
	case "ssl":
		m.state = StateDiagnostic
		if tool, exists := m.plugins.Get("ssl"); exists {
			diagnosticView := NewDiagnosticViewModel(tool)
			diagnosticView.SetSize(m.width, m.height)
			diagnosticView.SetTheme(m.theme)
			m.activeView = diagnosticView
		}
		return m, nil
	case "settings":
		m.state = StateSettings
		m.activeView = m.configView
		m.configView.SetSize(m.width, m.height)
		m.configView.SetTheme(m.theme)
		m.configView.Focus()
		return m, nil
	default:
		return m, nil
	}
}

// SetSize implements domain.TUIComponent
func (m *MainModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	if m.navigation != nil {
		m.navigation.SetSize(width, height)
	}
	
	if m.helpView != nil {
		m.helpView.SetSize(width, height)
	}
	
	if m.configView != nil {
		m.configView.SetSize(width, height)
	}
	
	if m.activeView != nil {
		if component, ok := m.activeView.(domain.TUIComponent); ok {
			component.SetSize(width, height)
		}
	}
}

// SetTheme implements domain.TUIComponent
func (m *MainModel) SetTheme(theme domain.Theme) {
	m.theme = theme
	
	if m.navigation != nil {
		m.navigation.SetTheme(theme)
	}
	
	if m.helpView != nil {
		m.helpView.SetTheme(theme)
	}
	
	if m.configView != nil {
		m.configView.SetTheme(theme)
	}
	
	if m.activeView != nil {
		if component, ok := m.activeView.(domain.TUIComponent); ok {
			component.SetTheme(theme)
		}
	}
}

// Focus implements domain.TUIComponent
func (m *MainModel) Focus() {
	if m.activeView != nil {
		if component, ok := m.activeView.(domain.TUIComponent); ok {
			component.Focus()
		}
	}
}

// Blur implements domain.TUIComponent
func (m *MainModel) Blur() {
	if m.activeView != nil {
		if component, ok := m.activeView.(domain.TUIComponent); ok {
			component.Blur()
		}
	}
}