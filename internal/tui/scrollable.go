// Package tui contains scrollable view components
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// ScrollableView provides a standardized scrollable content view
// It can work with both string content (legacy mode) and ScrollableItem lists (new mode)
type ScrollableView struct {
	viewport    viewport.Model
	ready       bool
	width       int
	height      int
	theme       domain.Theme
	keyMap      KeyMap
	focused     bool
	headerText  string
	footerText  string
	content     string
	
	// New scrollable list support
	items         []ScrollableItem
	selectedIndex int
	useScrollableList bool
}

// NewScrollableView creates a new scrollable view
func NewScrollableView() *ScrollableView {
	return &ScrollableView{
		keyMap:            DefaultKeyMap(),
		focused:           true,
		ready:             false,
		items:             make([]ScrollableItem, 0),
		selectedIndex:     0,
		useScrollableList: false,
	}
}

// Init implements tea.Model
func (s *ScrollableView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (s *ScrollableView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.handleResize(msg.Width, msg.Height)
	}

	// Handle viewport updates if ready
	if s.ready {
		s.viewport, cmd = s.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return s, tea.Batch(cmds...)
}

// View implements tea.Model
func (s *ScrollableView) View() string {
	if !s.ready {
		s.initializeIfReady()
		if !s.ready {
			return "\n  Initializing..."
		}
	}

	var result strings.Builder

	// Header
	if s.headerText != "" {
		result.WriteString(s.renderHeader())
		result.WriteString("\n")
	}

	// Viewport content
	result.WriteString(s.viewport.View())

	// Footer
	if s.footerText != "" {
		result.WriteString("\n")
		result.WriteString(s.renderFooter())
	}

	return result.String()
}

// SetSize implements domain.TUIComponent
func (s *ScrollableView) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.handleResize(width, height)
}

// SetTheme implements domain.TUIComponent
func (s *ScrollableView) SetTheme(theme domain.Theme) {
	s.theme = theme
}

// Focus implements domain.TUIComponent
func (s *ScrollableView) Focus() {
	s.focused = true
}

// Blur implements domain.TUIComponent
func (s *ScrollableView) Blur() {
	s.focused = false
}

// SetContent sets the main content to be displayed
// This method disables scrollable list mode and uses string content
func (s *ScrollableView) SetContent(content string) {
	s.content = content
	s.useScrollableList = false // Disable scrollable list mode when setting string content
	if s.ready {
		s.viewport.SetContent(content)
	}
}

// SetHeader sets the header text
func (s *ScrollableView) SetHeader(header string) {
	s.headerText = header
	s.handleResize(s.width, s.height) // Recalculate viewport size
}

// SetFooter sets the footer text
func (s *ScrollableView) SetFooter(footer string) {
	s.footerText = footer
	s.handleResize(s.width, s.height) // Recalculate viewport size
}

// GetScrollPercent returns the current scroll percentage
func (s *ScrollableView) GetScrollPercent() float64 {
	if s.ready {
		return s.viewport.ScrollPercent()
	}
	return 0.0
}

// ScrollToTop scrolls to the top
func (s *ScrollableView) ScrollToTop() {
	if s.ready {
		s.viewport.GotoTop()
	}
}

// ScrollToBottom scrolls to the bottom
func (s *ScrollableView) ScrollToBottom() {
	if s.ready {
		s.viewport.GotoBottom()
	}
}

// handleResize handles window resize events
func (s *ScrollableView) handleResize(width, height int) {
	s.width = width
	s.height = height

	headerHeight := 0
	if s.headerText != "" {
		headerHeight = lipgloss.Height(s.renderHeader()) + 1 // +1 for spacing
	}

	footerHeight := 0
	if s.footerText != "" {
		footerHeight = lipgloss.Height(s.renderFooter()) + 1 // +1 for spacing
	}

	verticalMarginHeight := headerHeight + footerHeight

	// Ensure minimum viewport height
	viewportHeight := height - verticalMarginHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	if s.ready {
		s.viewport.Width = width
		s.viewport.Height = viewportHeight
	} else {
		s.initializeIfReady()
	}
}

// initializeIfReady initializes the viewport if dimensions are available
func (s *ScrollableView) initializeIfReady() {
	if s.width > 0 && s.height > 0 {
		headerHeight := 0
		if s.headerText != "" {
			headerHeight = 2 // Approximate header height
		}

		footerHeight := 0
		if s.footerText != "" {
			footerHeight = 2 // Approximate footer height
		}

		verticalMarginHeight := headerHeight + footerHeight

		// Ensure minimum viewport height
		viewportHeight := s.height - verticalMarginHeight
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		s.viewport = viewport.New(s.width, viewportHeight)
		s.viewport.YPosition = headerHeight
		
		// Set content based on mode
		if s.useScrollableList {
			s.updateContentFromItems()
		} else if s.content != "" {
			s.viewport.SetContent(s.content)
		}
		s.ready = true
	}
}

// renderHeader renders the header
func (s *ScrollableView) renderHeader() string {
	if s.headerText == "" {
		return ""
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Align(lipgloss.Center).
		Width(s.width)

	return style.Render(s.headerText)
}

// renderFooter renders the footer with scroll information
func (s *ScrollableView) renderFooter() string {
	if s.footerText == "" {
		return ""
	}

	scrollInfo := ""
	if s.ready {
		scrollInfo = fmt.Sprintf("%.0f%% • ", s.viewport.ScrollPercent()*100)
	}

	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render(scrollInfo + s.footerText)

	line := strings.Repeat("─", max(0, s.width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

// ScrollableList interface implementation for ScrollableView

// SetItems implements ScrollableList interface
func (s *ScrollableView) SetItems(items []ScrollableItem) {
	s.items = items
	s.selectedIndex = 0
	s.useScrollableList = true
	s.updateContentFromItems()
}

// GetItems implements ScrollableList interface
func (s *ScrollableView) GetItems() []ScrollableItem {
	return s.items
}

// AddItem implements ScrollableList interface
func (s *ScrollableView) AddItem(item ScrollableItem) {
	s.items = append(s.items, item)
	s.useScrollableList = true
	s.updateContentFromItems()
}

// RemoveItem implements ScrollableList interface
func (s *ScrollableView) RemoveItem(index int) {
	if index < 0 || index >= len(s.items) {
		return
	}
	s.items = append(s.items[:index], s.items[index+1:]...)
	
	// Adjust selection if necessary
	if s.selectedIndex >= len(s.items) && len(s.items) > 0 {
		s.selectedIndex = len(s.items) - 1
	} else if len(s.items) == 0 {
		s.selectedIndex = 0
	}
	
	s.updateContentFromItems()
}

// GetSelected implements ScrollableList interface
func (s *ScrollableView) GetSelected() int {
	return s.selectedIndex
}

// SetSelected implements ScrollableList interface
func (s *ScrollableView) SetSelected(index int) {
	if len(s.items) == 0 {
		s.selectedIndex = 0
		return
	}
	
	if index < 0 {
		index = 0
	}
	if index >= len(s.items) {
		index = len(s.items) - 1
	}
	
	s.selectedIndex = index
	s.updateContentFromItems()
}

// GetScrollPosition implements ScrollableList interface
func (s *ScrollableView) GetScrollPosition() ScrollPosition {
	if !s.ready {
		return ScrollPosition{
			SelectedIndex:  s.selectedIndex,
			TopVisible:     0,
			ViewportHeight: 1,
		}
	}
	
	// Calculate viewport height
	viewportHeight := s.viewport.Height
	if viewportHeight <= 0 {
		viewportHeight = 1
	}
	
	// Calculate top visible based on viewport scroll
	topVisible := int(s.viewport.YOffset)
	
	return ScrollPosition{
		SelectedIndex:  s.selectedIndex,
		TopVisible:     topVisible,
		ViewportHeight: viewportHeight,
	}
}

// ScrollToItem implements ScrollableList interface
func (s *ScrollableView) ScrollToItem(index int) {
	s.SetSelected(index)
	if s.ready && s.useScrollableList {
		// Calculate the line number for this item
		lineNum := 0
		for i := 0; i < index && i < len(s.items); i++ {
			lineNum += s.items[i].GetHeight()
		}
		s.viewport.SetYOffset(lineNum)
	}
}

// MoveUp implements ScrollableList interface
func (s *ScrollableView) MoveUp() bool {
	if s.selectedIndex > 0 {
		s.selectedIndex--
		s.updateContentFromItems()
		s.ScrollToItem(s.selectedIndex)
		return true
	}
	return false
}

// MoveDown implements ScrollableList interface
func (s *ScrollableView) MoveDown() bool {
	if s.selectedIndex < len(s.items)-1 {
		s.selectedIndex++
		s.updateContentFromItems()
		s.ScrollToItem(s.selectedIndex)
		return true
	}
	return false
}

// PageUp implements ScrollableList interface
func (s *ScrollableView) PageUp() bool {
	if s.ready {
		oldOffset := s.viewport.YOffset
		s.viewport.ViewUp()
		return s.viewport.YOffset != oldOffset
	}
	return false
}

// PageDown implements ScrollableList interface
func (s *ScrollableView) PageDown() bool {
	if s.ready {
		oldOffset := s.viewport.YOffset
		s.viewport.ViewDown()
		return s.viewport.YOffset != oldOffset
	}
	return false
}

// Home implements ScrollableList interface
func (s *ScrollableView) Home() bool {
	if s.ready {
		oldOffset := s.viewport.YOffset
		s.viewport.GotoTop()
		if s.useScrollableList {
			s.selectedIndex = 0
			s.updateContentFromItems()
		}
		return s.viewport.YOffset != oldOffset
	}
	return false
}

// End implements ScrollableList interface
func (s *ScrollableView) End() bool {
	if s.ready {
		oldOffset := s.viewport.YOffset
		s.viewport.GotoBottom()
		if s.useScrollableList && len(s.items) > 0 {
			s.selectedIndex = len(s.items) - 1
			s.updateContentFromItems()
		}
		return s.viewport.YOffset != oldOffset
	}
	return false
}

// GetVisibleRange implements ScrollableList interface
func (s *ScrollableView) GetVisibleRange() (start, end int) {
	if !s.ready || !s.useScrollableList {
		return 0, len(s.items)
	}
	
	// Calculate which items are visible based on viewport
	topLine := int(s.viewport.YOffset)
	bottomLine := topLine + s.viewport.Height
	
	start = 0
	end = len(s.items)
	currentLine := 0
	
	// Find first visible item
	for i, item := range s.items {
		itemHeight := item.GetHeight()
		if currentLine+itemHeight > topLine {
			start = i
			break
		}
		currentLine += itemHeight
	}
	
	// Find last visible item
	currentLine = 0
	for i, item := range s.items {
		itemHeight := item.GetHeight()
		if currentLine >= bottomLine {
			end = i
			break
		}
		currentLine += itemHeight
	}
	
	return start, end
}

// IsItemVisible implements ScrollableList interface
func (s *ScrollableView) IsItemVisible(index int) bool {
	start, end := s.GetVisibleRange()
	return index >= start && index < end
}

// updateContentFromItems updates the viewport content from the items list
func (s *ScrollableView) updateContentFromItems() {
	if !s.useScrollableList {
		return
	}
	
	var contentBuilder strings.Builder
	for i, item := range s.items {
		selected := (i == s.selectedIndex)
		rendered := item.Render(s.width, selected, s.theme)
		contentBuilder.WriteString(rendered)
		if i < len(s.items)-1 {
			contentBuilder.WriteString("\n")
		}
	}
	
	s.content = contentBuilder.String()
	if s.ready {
		s.viewport.SetContent(s.content)
	}
}

// EnableScrollableList enables the new scrollable list mode
func (s *ScrollableView) EnableScrollableList() {
	s.useScrollableList = true
}

// DisableScrollableList disables the new scrollable list mode and reverts to string content
func (s *ScrollableView) DisableScrollableList() {
	s.useScrollableList = false
}

