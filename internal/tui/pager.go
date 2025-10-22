// Package tui contains pager component for scrollable content
//
// MIGRATION GUIDE:
// 
// The Pager component has been enhanced to work with the new standardized scrolling system.
// Existing code using Pager will continue to work unchanged, but new code should consider
// using the StandardScrollPager or ScrollableView components for better consistency.
//
// Migration options:
//
// 1. Keep using Pager (no changes needed):
//    pager := NewPager()
//    pager.SetContent("your content")
//
// 2. Use PagerScrollableAdapter for ScrollableList compatibility:
//    pager := NewPager()
//    adapter := NewPagerScrollableAdapter(pager)
//    adapter.SetItems([]ScrollableItem{...})
//
// 3. Migrate to StandardScrollPager (recommended for new code):
//    scrollPager := NewStandardScrollPager()
//    scrollPager.SetItems([]ScrollableItem{...})
//
// 4. Use ScrollableView for viewport-based scrolling:
//    view := NewScrollableView()
//    view.SetItems([]ScrollableItem{...}) // New scrollable list mode
//    // OR
//    view.SetContent("string content")     // Legacy string mode
//
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Pager provides scrollable content functionality
type Pager struct {
	width       int
	height      int
	content     []string
	scrollY     int
	maxScroll   int
	theme       domain.Theme
	keyMap      KeyMap
	focused     bool
	showScrollIndicators bool
}

// NewPager creates a new pager
func NewPager() *Pager {
	return &Pager{
		keyMap:      DefaultKeyMap(),
		focused:     true,
		showScrollIndicators: true,
	}
}

// SetContent sets the content to be displayed in the pager
func (p *Pager) SetContent(content string) {
	p.content = strings.Split(content, "\n")
	p.updateMaxScroll()
	p.scrollY = 0 // Reset scroll position when content changes
}

// SetContentLines sets the content as a slice of lines
func (p *Pager) SetContentLines(lines []string) {
	p.content = lines
	p.updateMaxScroll()
	p.scrollY = 0
}

// Update handles pager input
func (p *Pager) Update(msg tea.Msg) (*Pager, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused {
			return p, nil
		}

		switch {
		case key.Matches(msg, p.keyMap.Up):
			p.scrollUp()
		case key.Matches(msg, p.keyMap.Down):
			p.scrollDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			p.pageUp()
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdown"))):
			p.pageDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("home"))):
			p.scrollToTop()
		case key.Matches(msg, key.NewBinding(key.WithKeys("end"))):
			p.scrollToBottom()
		}
	}

	return p, nil
}

// View renders the pager content
func (p *Pager) View() string {
	if len(p.content) == 0 {
		return "No content to display"
	}

	// Calculate visible content area
	visibleHeight := p.height
	if p.showScrollIndicators {
		visibleHeight -= 2 // Reserve space for scroll indicators
	}
	if visibleHeight <= 0 {
		visibleHeight = 1
	}

	startLine := p.scrollY
	endLine := startLine + visibleHeight
	if endLine > len(p.content) {
		endLine = len(p.content)
	}

	var visibleContent []string
	if startLine < len(p.content) {
		visibleContent = p.content[startLine:endLine]
	}

	var result strings.Builder

	// Top scroll indicator
	if p.showScrollIndicators && p.scrollY > 0 {
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Align(lipgloss.Center).
			Width(p.width).
			Render("▲ More content above - Use ↑ or PgUp to scroll")
		result.WriteString(scrollIndicator + "\n")
	}

	// Content
	result.WriteString(strings.Join(visibleContent, "\n"))

	// Bottom scroll indicator
	if p.showScrollIndicators && endLine < len(p.content) {
		result.WriteString("\n")
		scrollIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Align(lipgloss.Center).
			Width(p.width).
			Render("▼ More content below - Use ↓ or PgDown to scroll")
		result.WriteString(scrollIndicator)
	}

	return result.String()
}

// SetSize sets the pager dimensions
func (p *Pager) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.updateMaxScroll()
}

// SetTheme sets the pager theme
func (p *Pager) SetTheme(theme domain.Theme) {
	p.theme = theme
}

// Focus focuses the pager
func (p *Pager) Focus() {
	p.focused = true
}

// Blur blurs the pager
func (p *Pager) Blur() {
	p.focused = false
}

// SetShowScrollIndicators controls whether scroll indicators are shown
func (p *Pager) SetShowScrollIndicators(show bool) {
	p.showScrollIndicators = show
	p.updateMaxScroll()
}

// CanScrollUp returns true if content can be scrolled up
func (p *Pager) CanScrollUp() bool {
	return p.scrollY > 0
}

// CanScrollDown returns true if content can be scrolled down
func (p *Pager) CanScrollDown() bool {
	return p.scrollY < p.maxScroll
}

// GetScrollPosition returns current scroll position and max scroll
func (p *Pager) GetScrollPosition() (int, int) {
	return p.scrollY, p.maxScroll
}

// scrollUp scrolls content up by one line
func (p *Pager) scrollUp() {
	if p.scrollY > 0 {
		p.scrollY--
	}
}

// scrollDown scrolls content down by one line
func (p *Pager) scrollDown() {
	if p.scrollY < p.maxScroll {
		p.scrollY++
	}
}

// pageUp scrolls up by one page
func (p *Pager) pageUp() {
	pageSize := p.height
	if p.showScrollIndicators {
		pageSize -= 2
	}
	if pageSize <= 0 {
		pageSize = 1
	}
	
	p.scrollY -= pageSize
	if p.scrollY < 0 {
		p.scrollY = 0
	}
}

// pageDown scrolls down by one page
func (p *Pager) pageDown() {
	pageSize := p.height
	if p.showScrollIndicators {
		pageSize -= 2
	}
	if pageSize <= 0 {
		pageSize = 1
	}
	
	p.scrollY += pageSize
	if p.scrollY > p.maxScroll {
		p.scrollY = p.maxScroll
	}
}

// scrollToTop scrolls to the top of content
func (p *Pager) scrollToTop() {
	p.scrollY = 0
}

// scrollToBottom scrolls to the bottom of content
func (p *Pager) scrollToBottom() {
	p.scrollY = p.maxScroll
}

// updateMaxScroll calculates the maximum scroll position
func (p *Pager) updateMaxScroll() {
	visibleHeight := p.height
	if p.showScrollIndicators {
		visibleHeight -= 2
	}
	if visibleHeight <= 0 {
		visibleHeight = 1
	}
	
	p.maxScroll = len(p.content) - visibleHeight
	if p.maxScroll < 0 {
		p.maxScroll = 0
	}
}

// PagerScrollableAdapter provides backward compatibility by adapting the existing Pager
// to work with the new ScrollableList interface. This allows existing code using Pager
// to continue working while gradually migrating to the new scrolling system.
type PagerScrollableAdapter struct {
	*Pager
	items []ScrollableItem
}

// Init implements the TUIComponent interface (required by ScrollableList)
func (p *PagerScrollableAdapter) Init() tea.Cmd {
	return nil
}

// Update implements the TUIComponent interface (required by ScrollableList)
func (p *PagerScrollableAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedPager, cmd := p.Pager.Update(msg)
	p.Pager = updatedPager
	return p, cmd
}

// NewPagerScrollableAdapter creates a new adapter that wraps an existing Pager
func NewPagerScrollableAdapter(pager *Pager) *PagerScrollableAdapter {
	return &PagerScrollableAdapter{
		Pager: pager,
		items: make([]ScrollableItem, 0),
	}
}

// StringScrollableItem is a simple implementation of ScrollableItem for string content
type StringScrollableItem struct {
	content string
	id      string
}

// NewStringScrollableItem creates a new string-based scrollable item
func NewStringScrollableItem(content, id string) *StringScrollableItem {
	return &StringScrollableItem{
		content: content,
		id:      id,
	}
}

// Render implements ScrollableItem interface
func (s *StringScrollableItem) Render(width int, selected bool, theme domain.Theme) string {
	return s.content
}

// GetHeight implements ScrollableItem interface
func (s *StringScrollableItem) GetHeight() int {
	return 1
}

// IsSelectable implements ScrollableItem interface
func (s *StringScrollableItem) IsSelectable() bool {
	return false // String items are not selectable by default
}

// GetID implements ScrollableItem interface
func (s *StringScrollableItem) GetID() string {
	return s.id
}

// SetItems implements ScrollableList interface
func (p *PagerScrollableAdapter) SetItems(items []ScrollableItem) {
	p.items = items
	// Convert items to string content for the underlying pager
	var lines []string
	for _, item := range items {
		lines = append(lines, item.Render(p.width, false, p.theme))
	}
	p.Pager.SetContentLines(lines)
}

// GetItems implements ScrollableList interface
func (p *PagerScrollableAdapter) GetItems() []ScrollableItem {
	return p.items
}

// AddItem implements ScrollableList interface
func (p *PagerScrollableAdapter) AddItem(item ScrollableItem) {
	p.items = append(p.items, item)
	// Update the underlying pager content
	var lines []string
	for _, item := range p.items {
		lines = append(lines, item.Render(p.width, false, p.theme))
	}
	p.Pager.SetContentLines(lines)
}

// RemoveItem implements ScrollableList interface
func (p *PagerScrollableAdapter) RemoveItem(index int) {
	if index < 0 || index >= len(p.items) {
		return
	}
	p.items = append(p.items[:index], p.items[index+1:]...)
	// Update the underlying pager content
	var lines []string
	for _, item := range p.items {
		lines = append(lines, item.Render(p.width, false, p.theme))
	}
	p.Pager.SetContentLines(lines)
}

// GetSelected implements ScrollableList interface
func (p *PagerScrollableAdapter) GetSelected() int {
	// Since the original Pager doesn't have selection, return the top visible line
	return p.scrollY
}

// SetSelected implements ScrollableList interface
func (p *PagerScrollableAdapter) SetSelected(index int) {
	// Scroll to make the selected line visible
	if index >= 0 && index < len(p.content) {
		p.scrollY = index
		if p.scrollY > p.maxScroll {
			p.scrollY = p.maxScroll
		}
	}
}

// GetScrollPosition implements ScrollableList interface
func (p *PagerScrollableAdapter) GetScrollPosition() ScrollPosition {
	visibleHeight := p.height
	if p.showScrollIndicators {
		visibleHeight -= 2
	}
	if visibleHeight <= 0 {
		visibleHeight = 1
	}
	
	return ScrollPosition{
		SelectedIndex:  p.scrollY,
		TopVisible:     p.scrollY,
		ViewportHeight: visibleHeight,
	}
}

// ScrollToItem implements ScrollableList interface
func (p *PagerScrollableAdapter) ScrollToItem(index int) {
	p.SetSelected(index)
}

// MoveUp implements ScrollableList interface
func (p *PagerScrollableAdapter) MoveUp() bool {
	oldScrollY := p.scrollY
	p.scrollUp()
	return p.scrollY != oldScrollY
}

// MoveDown implements ScrollableList interface
func (p *PagerScrollableAdapter) MoveDown() bool {
	oldScrollY := p.scrollY
	p.scrollDown()
	return p.scrollY != oldScrollY
}

// PageUp implements ScrollableList interface
func (p *PagerScrollableAdapter) PageUp() bool {
	oldScrollY := p.scrollY
	p.pageUp()
	return p.scrollY != oldScrollY
}

// PageDown implements ScrollableList interface
func (p *PagerScrollableAdapter) PageDown() bool {
	oldScrollY := p.scrollY
	p.pageDown()
	return p.scrollY != oldScrollY
}

// Home implements ScrollableList interface
func (p *PagerScrollableAdapter) Home() bool {
	if p.scrollY > 0 {
		p.scrollToTop()
		return true
	}
	return false
}

// End implements ScrollableList interface
func (p *PagerScrollableAdapter) End() bool {
	if p.scrollY < p.maxScroll {
		p.scrollToBottom()
		return true
	}
	return false
}

// GetVisibleRange implements ScrollableList interface
func (p *PagerScrollableAdapter) GetVisibleRange() (start, end int) {
	visibleHeight := p.height
	if p.showScrollIndicators {
		visibleHeight -= 2
	}
	if visibleHeight <= 0 {
		visibleHeight = 1
	}
	
	start = p.scrollY
	end = p.scrollY + visibleHeight
	if end > len(p.content) {
		end = len(p.content)
	}
	return start, end
}

// IsItemVisible implements ScrollableList interface
func (p *PagerScrollableAdapter) IsItemVisible(index int) bool {
	start, end := p.GetVisibleRange()
	return index >= start && index < end
}