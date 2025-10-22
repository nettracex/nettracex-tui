// Package tui contains navigation components for the TUI
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// NavigationAction represents different navigation actions
type NavigationAction int

const (
	NavigationActionSelect NavigationAction = iota
	NavigationActionBack
	NavigationActionUp
	NavigationActionDown
)

// NavigationMsg represents a navigation message
type NavigationMsg struct {
	Action NavigationAction
	Data   interface{}
}

// NavigationItem represents a menu item
type NavigationItem struct {
	ID          string
	Title       string
	Description string
	Icon        string
	Enabled     bool
}

// Render implements ScrollableItem interface
// Renders the navigation item with proper selection styling and theme application
func (n NavigationItem) Render(width int, selected bool, theme domain.Theme) string {
	// Get icon, default to bullet if empty
	icon := n.Icon
	if icon == "" {
		icon = "•"
	}
	
	// Build title with disabled indicator if needed
	title := n.Title
	if !n.Enabled {
		title += " (disabled)"
	}
	
	// Create the main item text
	itemText := icon + " " + title
	
	// Add description if available
	if n.Description != "" {
		// Get theme-aware style for description
		var descStyle lipgloss.Style
		if theme != nil {
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.GetColor("muted"))).
				Italic(true)
		} else {
			// Fallback styling
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)
		}
		itemText += "\n  " + descStyle.Render(n.Description)
	}
	
	// Apply selection and enabled state styling
	style := n.getItemStyle(width, selected, n.Enabled, theme)
	
	return style.Render(itemText)
}

// GetHeight implements ScrollableItem interface
// Returns the number of lines this item occupies when rendered
func (n NavigationItem) GetHeight() int {
	height := 1 // Icon + title line
	if n.Description != "" {
		height++ // Description line
	}
	height++ // Spacing line
	return height
}

// IsSelectable implements ScrollableItem interface
// Returns true if this item can be selected by the user
func (n NavigationItem) IsSelectable() bool {
	return n.Enabled
}

// GetID implements ScrollableItem interface
// Returns a unique identifier for this item
func (n NavigationItem) GetID() string {
	return n.ID
}

// getItemStyle returns the appropriate style for a navigation item
// This is a helper method that applies theme-aware styling based on selection and enabled state
func (n NavigationItem) getItemStyle(width int, selected, enabled bool, theme domain.Theme) lipgloss.Style {
	style := lipgloss.NewStyle().
		Padding(0, 2).
		Width(width - 4)

	if theme != nil {
		// Use theme-aware styling
		if !enabled {
			style = style.Foreground(lipgloss.Color(theme.GetColor("muted")))
		} else if selected {
			style = style.
				Background(lipgloss.Color(theme.GetColor("primary"))).
				Foreground(lipgloss.Color(theme.GetColor("highlight"))).
				Bold(true)
		} else {
			style = style.Foreground(lipgloss.Color(theme.GetColor("foreground")))
		}
	} else {
		// Fallback styling when no theme is available
		if !enabled {
			style = style.Foreground(lipgloss.Color("240"))
		} else if selected {
			style = style.
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Bold(true)
		} else {
			style = style.Foreground(lipgloss.Color("252"))
		}
	}

	return style
}

// NavigationModel handles menu and navigation
type NavigationModel struct {
	breadcrumbs []string
	theme       domain.Theme
	focused     bool
	keyMap      KeyMap
	scrollPager *StandardScrollPager
}

// NewNavigationModel creates a new navigation model
func NewNavigationModel() *NavigationModel {
	items := []NavigationItem{
		{
			ID:          "whois",
			Title:       "WHOIS Lookup",
			Description: "Domain and IP registration information",
			Icon:        "🔍",
			Enabled:     true,
		},
		{
			ID:          "ping",
			Title:       "Ping Test",
			Description: "Test connectivity and measure latency",
			Icon:        "📡",
			Enabled:     true,
		},
		{
			ID:          "traceroute",
			Title:       "Traceroute",
			Description: "Trace network path to destination",
			Icon:        "🗺️",
			Enabled:     true,
		},
		{
			ID:          "dns",
			Title:       "DNS Lookup",
			Description: "Query DNS records for domains",
			Icon:        "🌐",
			Enabled:     true,
		},
		{
			ID:          "ssl",
			Title:       "SSL Certificate Check",
			Description: "Verify SSL certificate validity",
			Icon:        "🔒",
			Enabled:     true,
		},
		{
			ID:          "settings",
			Title:       "Settings",
			Description: "Configure application preferences",
			Icon:        "⚙️",
			Enabled:     true,
		},
	}

	scrollPager := NewStandardScrollPager()
	
	// Convert NavigationItems to ScrollableItems
	scrollableItems := make([]ScrollableItem, len(items))
	for i, item := range items {
		scrollableItems[i] = item
	}
	scrollPager.SetItems(scrollableItems)
	scrollPager.SetShowScrollIndicators(true)

	return &NavigationModel{
		focused:     true,
		keyMap:      DefaultKeyMap(),
		scrollPager: scrollPager,
	}
}

// Init implements tea.Model
func (m *NavigationModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *NavigationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	// Delegate scrolling to StandardScrollPager
	var cmd tea.Cmd
	updatedModel, scrollCmd := m.scrollPager.Update(msg)
	if pager, ok := updatedModel.(*StandardScrollPager); ok {
		m.scrollPager = pager
		cmd = scrollCmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Enter):
			selectedIndex := m.scrollPager.GetSelected()
			items := m.scrollPager.GetItems()
			if selectedIndex >= 0 && selectedIndex < len(items) {
				if navItem, ok := items[selectedIndex].(NavigationItem); ok && navItem.Enabled {
					return m, func() tea.Msg {
						return NavigationMsg{
							Action: NavigationActionSelect,
							Data:   navItem,
						}
					}
				}
			}

		case key.Matches(msg, m.keyMap.Back):
			return m, func() tea.Msg {
				return NavigationMsg{
					Action: NavigationActionBack,
					Data:   nil,
				}
			}
		}
	}

	return m, cmd
}

// View implements tea.Model
func (m *NavigationModel) View() string {
	// Add title section
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(1, 0)
	
	title := titleStyle.Render("Network Diagnostic Tools")
	
	// Render through StandardScrollPager
	content := m.scrollPager.View()
	
	// Combine title and scrollable content
	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}



// SetSize implements domain.TUIComponent
func (m *NavigationModel) SetSize(width, height int) {
	// Reserve space for title (3 lines: title + padding + empty line)
	contentHeight := height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	
	m.scrollPager.SetSize(width, contentHeight)
}

// SetTheme implements domain.TUIComponent
func (m *NavigationModel) SetTheme(theme domain.Theme) {
	m.theme = theme
	m.scrollPager.SetTheme(theme)
}

// Focus implements domain.TUIComponent
func (m *NavigationModel) Focus() {
	m.focused = true
	m.scrollPager.Focus()
}

// Blur implements domain.TUIComponent
func (m *NavigationModel) Blur() {
	m.focused = false
	m.scrollPager.Blur()
}

// GetSelected returns the currently selected item
func (m *NavigationModel) GetSelected() *NavigationItem {
	selectedIndex := m.scrollPager.GetSelected()
	items := m.scrollPager.GetItems()
	if selectedIndex >= 0 && selectedIndex < len(items) {
		if navItem, ok := items[selectedIndex].(NavigationItem); ok {
			return &navItem
		}
	}
	return nil
}

// SetSelected sets the selected item by index
func (m *NavigationModel) SetSelected(index int) {
	m.scrollPager.SetSelected(index)
}

// AddBreadcrumb adds a breadcrumb to the navigation
func (m *NavigationModel) AddBreadcrumb(crumb string) {
	m.breadcrumbs = append(m.breadcrumbs, crumb)
}

// PopBreadcrumb removes the last breadcrumb
func (m *NavigationModel) PopBreadcrumb() string {
	if len(m.breadcrumbs) == 0 {
		return ""
	}
	
	last := m.breadcrumbs[len(m.breadcrumbs)-1]
	m.breadcrumbs = m.breadcrumbs[:len(m.breadcrumbs)-1]
	return last
}

// GetBreadcrumbs returns the current breadcrumbs
func (m *NavigationModel) GetBreadcrumbs() []string {
	return append([]string{}, m.breadcrumbs...)
}

// EnableItem enables a menu item by ID
func (m *NavigationModel) EnableItem(id string) {
	items := m.scrollPager.GetItems()
	for i, item := range items {
		if navItem, ok := item.(NavigationItem); ok && navItem.ID == id {
			navItem.Enabled = true
			// Update the item in the scroll pager
			newItems := make([]ScrollableItem, len(items))
			copy(newItems, items)
			newItems[i] = navItem
			m.scrollPager.SetItems(newItems)
			break
		}
	}
}

// DisableItem disables a menu item by ID
func (m *NavigationModel) DisableItem(id string) {
	items := m.scrollPager.GetItems()
	for i, item := range items {
		if navItem, ok := item.(NavigationItem); ok && navItem.ID == id {
			navItem.Enabled = false
			// Update the item in the scroll pager
			newItems := make([]ScrollableItem, len(items))
			copy(newItems, items)
			newItems[i] = navItem
			m.scrollPager.SetItems(newItems)
			break
		}
	}
}

// AddItem adds a new navigation item
func (m *NavigationModel) AddItem(item NavigationItem) {
	m.scrollPager.AddItem(item)
}

// RemoveItem removes a navigation item by ID
func (m *NavigationModel) RemoveItem(id string) {
	items := m.scrollPager.GetItems()
	for i, item := range items {
		if navItem, ok := item.(NavigationItem); ok && navItem.ID == id {
			m.scrollPager.RemoveItem(i)
			break
		}
	}
}