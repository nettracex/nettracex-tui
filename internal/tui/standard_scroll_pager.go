// Package tui contains the StandardScrollPager implementation
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// StandardScrollPager provides unified scrolling behavior for all TUI models
type StandardScrollPager struct {
	content *ScrollableContent
	width   int
	height  int
	focused bool
}

// NewStandardScrollPager creates a new StandardScrollPager
func NewStandardScrollPager() *StandardScrollPager {
	return &StandardScrollPager{
		content: NewScrollableContent(),
		focused: true,
	}
}

// Init implements tea.Model
func (p *StandardScrollPager) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (p *StandardScrollPager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !p.focused {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, p.content.KeyMap.Up):
			p.MoveUp()
		case key.Matches(msg, p.content.KeyMap.Down):
			p.MoveDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup", "ctrl+b"))):
			p.PageUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown", "ctrl+f"))):
			p.PageDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("home", "ctrl+a"))):
			p.Home()
		case key.Matches(msg, key.NewBinding(key.WithKeys("end", "ctrl+e"))):
			p.End()
		}
	}

	return p, nil
}

// View implements tea.Model
func (p *StandardScrollPager) View() string {
	if len(p.content.Items) == 0 {
		return p.renderEmptyState()
	}

	var result strings.Builder

	// Calculate visible content area
	visibleHeight := p.height
	if p.content.ShowIndicators {
		// Reserve space for scroll indicators
		if p.content.Position.CanScrollUp() {
			visibleHeight--
		}
		if p.content.Position.CanScrollDown(len(p.content.Items)) {
			visibleHeight--
		}
	}

	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Update viewport height if it has changed
	if p.content.Position.ViewportHeight != visibleHeight {
		p.content.SetViewportHeight(visibleHeight)
	}

	// Top scroll indicator
	if p.content.ShowIndicators && p.content.Position.CanScrollUp() {
		indicator := p.renderScrollIndicator("▲ More content above", "Use ↑ or PgUp to scroll")
		result.WriteString(indicator + "\n")
	}

	// Render visible items
	start, end := p.content.Position.GetVisibleRange(len(p.content.Items))
	for i := start; i < end; i++ {
		if i >= len(p.content.Items) {
			break
		}

		item := p.content.Items[i]
		selected := (i == p.content.Position.SelectedIndex)
		rendered := item.Render(p.width, selected, p.content.Theme)
		
		result.WriteString(rendered)
		if i < end-1 {
			result.WriteString("\n")
		}
	}

	// Bottom scroll indicator
	if p.content.ShowIndicators && p.content.Position.CanScrollDown(len(p.content.Items)) {
		result.WriteString("\n")
		indicator := p.renderScrollIndicator("▼ More content below", "Use ↓ or PgDown to scroll")
		result.WriteString(indicator)
	}

	return result.String()
}

// SetSize implements domain.TUIComponent
func (p *StandardScrollPager) SetSize(width, height int) {
	p.width = width
	p.height = height
	
	// Update viewport height based on new dimensions
	visibleHeight := height
	if p.content.ShowIndicators {
		// Account for potential scroll indicators
		visibleHeight -= 2
	}
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	
	p.content.SetViewportHeight(visibleHeight)
}

// SetTheme implements domain.TUIComponent
func (p *StandardScrollPager) SetTheme(theme domain.Theme) {
	p.content.Theme = theme
}

// Focus implements domain.TUIComponent
func (p *StandardScrollPager) Focus() {
	p.focused = true
}

// Blur implements domain.TUIComponent
func (p *StandardScrollPager) Blur() {
	p.focused = false
}

// SetItems implements ScrollableList
func (p *StandardScrollPager) SetItems(items []ScrollableItem) {
	p.content.SetItems(items)
}

// GetItems implements ScrollableList
func (p *StandardScrollPager) GetItems() []ScrollableItem {
	return p.content.Items
}

// AddItem implements ScrollableList
func (p *StandardScrollPager) AddItem(item ScrollableItem) {
	p.content.AddItem(item)
}

// RemoveItem implements ScrollableList
func (p *StandardScrollPager) RemoveItem(index int) {
	p.content.RemoveItem(index)
}

// GetSelected implements ScrollableList
func (p *StandardScrollPager) GetSelected() int {
	return p.content.Position.SelectedIndex
}

// SetSelected implements ScrollableList
func (p *StandardScrollPager) SetSelected(index int) {
	if len(p.content.Items) == 0 {
		p.content.Position.SelectedIndex = 0
		return
	}

	if index < 0 {
		index = 0
	}
	if index >= len(p.content.Items) {
		index = len(p.content.Items) - 1
	}

	p.content.Position.SelectedIndex = index
	p.content.Position.EnsureSelectionVisible(len(p.content.Items))
}

// GetScrollPosition implements ScrollableList
func (p *StandardScrollPager) GetScrollPosition() ScrollPosition {
	return p.content.Position
}

// ScrollToItem implements ScrollableList
func (p *StandardScrollPager) ScrollToItem(index int) {
	p.SetSelected(index)
}

// MoveUp implements ScrollableList
func (p *StandardScrollPager) MoveUp() bool {
	if len(p.content.Items) == 0 {
		return false
	}

	// Find previous selectable item
	for i := p.content.Position.SelectedIndex - 1; i >= 0; i-- {
		if p.content.Items[i].IsSelectable() {
			p.SetSelected(i)
			return true
		}
	}

	return false
}

// MoveDown implements ScrollableList
func (p *StandardScrollPager) MoveDown() bool {
	if len(p.content.Items) == 0 {
		return false
	}

	// Find next selectable item
	for i := p.content.Position.SelectedIndex + 1; i < len(p.content.Items); i++ {
		if p.content.Items[i].IsSelectable() {
			p.SetSelected(i)
			return true
		}
	}

	return false
}

// PageUp implements ScrollableList
func (p *StandardScrollPager) PageUp() bool {
	if len(p.content.Items) == 0 {
		return false
	}

	// Move up by viewport height minus one for context
	pageSize := p.content.Position.ViewportHeight - 1
	if pageSize < 1 {
		pageSize = 1
	}

	targetIndex := p.content.Position.SelectedIndex - pageSize
	if targetIndex < 0 {
		targetIndex = 0
	}

	// Find nearest selectable item at or after target
	for i := targetIndex; i < len(p.content.Items); i++ {
		if p.content.Items[i].IsSelectable() {
			p.SetSelected(i)
			return true
		}
	}

	return false
}

// PageDown implements ScrollableList
func (p *StandardScrollPager) PageDown() bool {
	if len(p.content.Items) == 0 {
		return false
	}

	// Move down by viewport height minus one for context
	pageSize := p.content.Position.ViewportHeight - 1
	if pageSize < 1 {
		pageSize = 1
	}

	targetIndex := p.content.Position.SelectedIndex + pageSize
	if targetIndex >= len(p.content.Items) {
		targetIndex = len(p.content.Items) - 1
	}

	// Find nearest selectable item at or before target
	for i := targetIndex; i >= 0; i-- {
		if p.content.Items[i].IsSelectable() {
			p.SetSelected(i)
			return true
		}
	}

	return false
}

// Home implements ScrollableList
func (p *StandardScrollPager) Home() bool {
	if len(p.content.Items) == 0 {
		return false
	}

	// Find first selectable item
	for i := 0; i < len(p.content.Items); i++ {
		if p.content.Items[i].IsSelectable() {
			p.SetSelected(i)
			return true
		}
	}

	return false
}

// End implements ScrollableList
func (p *StandardScrollPager) End() bool {
	if len(p.content.Items) == 0 {
		return false
	}

	// Find last selectable item
	for i := len(p.content.Items) - 1; i >= 0; i-- {
		if p.content.Items[i].IsSelectable() {
			p.SetSelected(i)
			return true
		}
	}

	return false
}

// GetVisibleRange implements ScrollableList
func (p *StandardScrollPager) GetVisibleRange() (start, end int) {
	return p.content.Position.GetVisibleRange(len(p.content.Items))
}

// IsItemVisible implements ScrollableList
func (p *StandardScrollPager) IsItemVisible(index int) bool {
	return p.content.Position.IsItemVisible(index)
}

// SetShowScrollIndicators controls whether scroll indicators are displayed
func (p *StandardScrollPager) SetShowScrollIndicators(show bool) {
	p.content.ShowIndicators = show
}

// renderScrollIndicator renders a scroll indicator with styling
func (p *StandardScrollPager) renderScrollIndicator(icon, helpText string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Align(lipgloss.Center).
		Width(p.width)

	text := icon
	if helpText != "" {
		text += " - " + helpText
	}

	return style.Render(text)
}

// renderEmptyState renders the empty state when no items are present
func (p *StandardScrollPager) renderEmptyState() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true).
		Align(lipgloss.Center).
		Width(p.width).
		Height(p.height)

	return style.Render("No items to display")
}