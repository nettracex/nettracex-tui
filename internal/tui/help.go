// Package tui contains help view components
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// HelpModel displays help information and keyboard shortcuts using viewport for smooth scrolling
type HelpModel struct {
	width      int
	height     int
	theme      domain.Theme
	keyMap     KeyMap
	focused    bool
	viewport   *ScrollableView
	ready      bool
	content    string
}

// NewHelpModel creates a new help model
func NewHelpModel() *HelpModel {
	viewport := NewScrollableView()
	
	return &HelpModel{
		keyMap:   DefaultKeyMap(),
		focused:  true,
		viewport: viewport,
		ready:    false,
	}
}

// Init implements tea.Model
func (m *HelpModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyMap.Back), key.Matches(msg, m.keyMap.Help):
			// Send back navigation message
			return m, func() tea.Msg {
				return NavigationMsg{
					Action: NavigationActionBack,
				}
			}
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		default:
			// Delegate scrolling to viewport
			_, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		headerHeight := 2 // Space for title
		footerHeight := 2 // Space for help text
		verticalMarginHeight := headerHeight + footerHeight

		// Ensure minimum content height
		contentHeight := msg.Height - verticalMarginHeight
		if contentHeight < 1 {
			contentHeight = 1
		}

		if !m.ready {
			// Initialize help content and viewport
			m.initializeHelpContent()
			m.viewport.SetSize(msg.Width, contentHeight)
			m.ready = true
		} else {
			m.viewport.SetSize(msg.Width, contentHeight)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *HelpModel) View() string {
	if !m.ready {
		// Initialize with default size if not ready
		if m.width > 0 && m.height > 0 {
			headerHeight := 2
			footerHeight := 2
			verticalMarginHeight := headerHeight + footerHeight
			
			// Ensure minimum content height
			contentHeight := m.height - verticalMarginHeight
			if contentHeight < 1 {
				contentHeight = 1
			}
			
			m.initializeHelpContent()
			m.viewport.SetSize(m.width, contentHeight)
			m.ready = true
		} else {
			return "\n  Initializing help..."
		}
	}

	// Header
	header := m.headerView()
	
	// Footer
	footer := m.footerView()

	// Combine header, viewport content, and footer
	return header + "\n" + m.viewport.View() + "\n" + footer
}

// initializeHelpContent sets up the help content as plain text for smooth scrolling
func (m *HelpModel) initializeHelpContent() {
	// Generate help content as formatted text
	m.content = m.generateHelpContent()
	
	// Set the content in the viewport
	m.viewport.SetContent(m.content)
	
	// Set theme for consistent styling
	m.viewport.SetTheme(m.theme)
}

// SetSize implements domain.TUIComponent
func (m *HelpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	// Update scroll pager size if ready
	if m.ready {
		headerHeight := 2
		footerHeight := 2
		verticalMarginHeight := headerHeight + footerHeight
		
		// Ensure minimum content height
		contentHeight := height - verticalMarginHeight
		if contentHeight < 1 {
			contentHeight = 1
		}
		
		m.viewport.SetSize(width, contentHeight)
	}
}

// SetTheme implements domain.TUIComponent
func (m *HelpModel) SetTheme(theme domain.Theme) {
	m.theme = theme
	// Update viewport theme
	if m.viewport != nil {
		m.viewport.SetTheme(theme)
	}
}

// Focus implements domain.TUIComponent
func (m *HelpModel) Focus() {
	m.focused = true
	// Focus the viewport
	if m.viewport != nil {
		m.viewport.Focus()
	}
}

// Blur implements domain.TUIComponent
func (m *HelpModel) Blur() {
	m.focused = false
	// Blur the viewport
	if m.viewport != nil {
		m.viewport.Blur()
	}
}



// headerView renders the help header
func (m *HelpModel) headerView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Align(lipgloss.Center).
		Width(m.width)

	return titleStyle.Render("NetTraceX Help")
}

// footerView renders the help footer
func (m *HelpModel) footerView() string {
	// Calculate scroll percentage based on viewport position
	scrollPercent := 0.0
	if m.viewport != nil {
		scrollPercent = m.viewport.GetScrollPercent() * 100
	}
	
	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(fmt.Sprintf("%.0f%% • Press Esc or ? to close • Use ↑/↓ PgUp/PgDown to scroll", scrollPercent))
	
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

// generateHelpContent creates formatted help content as a string
func (m *HelpModel) generateHelpContent() string {
	var content strings.Builder
	
	// Navigation & Scrolling section
	content.WriteString(m.renderHelpSection("Navigation & Scrolling", []HelpItem{
		NewHelpItem("↑/↓ or j/k", "Navigate up/down in menus or scroll content"),
		NewHelpItem("←/→ or h/l", "Navigate left/right (when applicable)"),
		NewHelpItem("PgUp/PgDown", "Scroll page up/down in help and results"),
		NewHelpItem("Home/End", "Jump to top/bottom of scrollable content"),
		NewHelpItem("Enter", "Select menu item or execute action"),
		NewHelpItem("Esc", "Return to tool input"),
		NewHelpItem("Tab", "Switch between input fields"),
	}))
	
	// Tool Operations section
	content.WriteString(m.renderHelpSection("Tool Operations", []HelpItem{
		NewHelpItem("Enter", "Execute diagnostic tool with current parameters"),
		NewHelpItem("f/t/r", "Switch between formatted/table/raw result views"),
		NewHelpItem("Tab", "Cycle through result view modes"),
		NewHelpItem("s", "Save configuration (in settings)"),
		NewHelpItem("e", "Export results (when available)"),
	}))
	
	// Tips & Examples section
	content.WriteString(m.renderHelpSection("Tips & Examples", []HelpItem{
		NewHelpItem("Domain examples", "google.com, github.io, example.dev, lavan.dev"),
		NewHelpItem("IP examples", "8.8.8.8, 1.1.1.1, 192.168.1.1"),
		NewHelpItem("Ping counts", "Use 1-100 for ping count (default: 4)"),
		NewHelpItem("DNS records", "A, AAAA, MX, TXT, CNAME, NS supported"),
		NewHelpItem("SSL ports", "443 (HTTPS), 993 (IMAPS), 995 (POP3S)"),
		NewHelpItem("WHOIS queries", "Works with domains and IP addresses"),
		NewHelpItem("Traceroute", "Shows network path with hop details"),
	}))
	
	// Troubleshooting section
	content.WriteString(m.renderHelpSection("Troubleshooting", []HelpItem{
		NewHelpItem("No results", "Check network connection and query format"),
		NewHelpItem("Timeout errors", "Try again or check if host is reachable"),
		NewHelpItem("WHOIS no data", "Some domains may have privacy protection"),
		NewHelpItem("DNS failures", "Verify domain exists and DNS servers work"),
		NewHelpItem("SSL errors", "Check if port supports SSL/TLS"),
		NewHelpItem("Long results", "Use ↑/↓ or PgUp/PgDown to scroll"),
	}))
	
	return content.String()
}

// renderHelpSection renders a help section with title and items
func (m *HelpModel) renderHelpSection(title string, items []HelpItem) string {
	var content strings.Builder
	
	// Section title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")
	
	// Section items
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Width(20).
		Align(lipgloss.Left)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	for _, item := range items {
		key := keyStyle.Render(item.Key)
		value := valueStyle.Render(item.Description)
		content.WriteString("  " + key + " " + value + "\n")
	}
	
	content.WriteString("\n") // Add spacing after section
	return content.String()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

