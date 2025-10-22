// Package tui contains tests for StandardScrollPager
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewStandardScrollPager(t *testing.T) {
	pager := NewStandardScrollPager()
	
	if pager == nil {
		t.Fatal("Expected NewStandardScrollPager to return non-nil")
	}
	
	if pager.content == nil {
		t.Error("Expected content to be initialized")
	}
	
	if !pager.focused {
		t.Error("Expected pager to be focused by default")
	}
	
	if len(pager.content.Items) != 0 {
		t.Errorf("Expected no items initially, got %d", len(pager.content.Items))
	}
}

func TestStandardScrollPager_SetItems(t *testing.T) {
	pager := NewStandardScrollPager()
	
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
		MockScrollableItem{id: "3", content: "Item 3", height: 1, selectable: true},
	}
	
	pager.SetItems(items)
	
	if len(pager.GetItems()) != 3 {
		t.Errorf("Expected 3 items, got %d", len(pager.GetItems()))
	}
	
	if pager.GetSelected() != 0 {
		t.Errorf("Expected selected index to be 0, got %d", pager.GetSelected())
	}
}

func TestStandardScrollPager_AddRemoveItem(t *testing.T) {
	pager := NewStandardScrollPager()
	
	item1 := MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true}
	item2 := MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true}
	
	// Add items
	pager.AddItem(item1)
	pager.AddItem(item2)
	
	if len(pager.GetItems()) != 2 {
		t.Errorf("Expected 2 items after adding, got %d", len(pager.GetItems()))
	}
	
	// Remove item
	pager.RemoveItem(0)
	
	if len(pager.GetItems()) != 1 {
		t.Errorf("Expected 1 item after removing, got %d", len(pager.GetItems()))
	}
	
	if pager.GetItems()[0].GetID() != "2" {
		t.Errorf("Expected remaining item to have ID '2', got '%s'", pager.GetItems()[0].GetID())
	}
}

func TestStandardScrollPager_Selection(t *testing.T) {
	pager := NewStandardScrollPager()
	
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
		MockScrollableItem{id: "3", content: "Item 3", height: 1, selectable: true},
	}
	
	pager.SetItems(items)
	
	// Test SetSelected
	pager.SetSelected(1)
	if pager.GetSelected() != 1 {
		t.Errorf("Expected selected index to be 1, got %d", pager.GetSelected())
	}
	
	// Test ScrollToItem
	pager.ScrollToItem(2)
	if pager.GetSelected() != 2 {
		t.Errorf("Expected selected index to be 2 after ScrollToItem, got %d", pager.GetSelected())
	}
	
	// Test bounds checking
	pager.SetSelected(10)
	if pager.GetSelected() != 2 {
		t.Errorf("Expected selected index to be clamped to 2, got %d", pager.GetSelected())
	}
	
	pager.SetSelected(-1)
	if pager.GetSelected() != 0 {
		t.Errorf("Expected selected index to be clamped to 0, got %d", pager.GetSelected())
	}
}

func TestStandardScrollPager_Navigation(t *testing.T) {
	pager := NewStandardScrollPager()
	
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
		MockScrollableItem{id: "3", content: "Item 3", height: 1, selectable: true},
		MockScrollableItem{id: "4", content: "Item 4", height: 1, selectable: false}, // Not selectable
		MockScrollableItem{id: "5", content: "Item 5", height: 1, selectable: true},
	}
	
	pager.SetItems(items)
	
	// Test MoveDown
	if !pager.MoveDown() {
		t.Error("Expected MoveDown to return true")
	}
	if pager.GetSelected() != 1 {
		t.Errorf("Expected selected index to be 1 after MoveDown, got %d", pager.GetSelected())
	}
	
	// Test MoveUp
	if !pager.MoveUp() {
		t.Error("Expected MoveUp to return true")
	}
	if pager.GetSelected() != 0 {
		t.Errorf("Expected selected index to be 0 after MoveUp, got %d", pager.GetSelected())
	}
	
	// Test Home
	pager.SetSelected(2)
	if !pager.Home() {
		t.Error("Expected Home to return true")
	}
	if pager.GetSelected() != 0 {
		t.Errorf("Expected selected index to be 0 after Home, got %d", pager.GetSelected())
	}
	
	// Test End
	if !pager.End() {
		t.Error("Expected End to return true")
	}
	if pager.GetSelected() != 4 {
		t.Errorf("Expected selected index to be 4 after End (skipping non-selectable), got %d", pager.GetSelected())
	}
	
	// Test skipping non-selectable items
	pager.SetSelected(2)
	if !pager.MoveDown() {
		t.Error("Expected MoveDown to return true when skipping non-selectable")
	}
	if pager.GetSelected() != 4 {
		t.Errorf("Expected selected index to be 4 after MoveDown (skipping non-selectable), got %d", pager.GetSelected())
	}
}

func TestStandardScrollPager_PageNavigation(t *testing.T) {
	pager := NewStandardScrollPager()
	pager.SetSize(80, 5) // Small viewport
	
	// Create many items
	items := make([]ScrollableItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = MockScrollableItem{
			id:         string(rune('a' + i)),
			content:    "Item " + string(rune('A' + i)),
			height:     1,
			selectable: true,
		}
	}
	
	pager.SetItems(items)
	
	// Test PageDown
	if !pager.PageDown() {
		t.Error("Expected PageDown to return true")
	}
	
	// Should move by viewport height - 1 for context
	// With height 5, accounting for indicators, viewport is about 3, so page size is 2
	expectedMinIndex := 2 // Should move at least 2 positions
	if pager.GetSelected() < expectedMinIndex {
		t.Errorf("Expected selected index to be at least %d after PageDown, got %d", expectedMinIndex, pager.GetSelected())
	}
	
	// Test PageUp
	currentIndex := pager.GetSelected()
	if !pager.PageUp() {
		t.Error("Expected PageUp to return true")
	}
	
	if pager.GetSelected() >= currentIndex {
		t.Errorf("Expected selected index to decrease after PageUp, was %d, now %d", currentIndex, pager.GetSelected())
	}
}

func TestStandardScrollPager_EmptyList(t *testing.T) {
	pager := NewStandardScrollPager()
	
	// Test navigation with empty list
	if pager.MoveUp() {
		t.Error("Expected MoveUp to return false with empty list")
	}
	
	if pager.MoveDown() {
		t.Error("Expected MoveDown to return false with empty list")
	}
	
	if pager.PageUp() {
		t.Error("Expected PageUp to return false with empty list")
	}
	
	if pager.PageDown() {
		t.Error("Expected PageDown to return false with empty list")
	}
	
	if pager.Home() {
		t.Error("Expected Home to return false with empty list")
	}
	
	if pager.End() {
		t.Error("Expected End to return false with empty list")
	}
}

func TestStandardScrollPager_VisibleRange(t *testing.T) {
	pager := NewStandardScrollPager()
	pager.SetSize(80, 5)
	
	items := make([]ScrollableItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = MockScrollableItem{
			id:         string(rune('a' + i)),
			content:    "Item " + string(rune('A' + i)),
			height:     1,
			selectable: true,
		}
	}
	
	pager.SetItems(items)
	
	start, end := pager.GetVisibleRange()
	
	if start != 0 {
		t.Errorf("Expected visible range start to be 0, got %d", start)
	}
	
	// Should be limited by viewport height
	if end <= start {
		t.Errorf("Expected visible range end (%d) to be greater than start (%d)", end, start)
	}
	
	// Test IsItemVisible
	if !pager.IsItemVisible(0) {
		t.Error("Expected first item to be visible")
	}
	
	if !pager.IsItemVisible(1) {
		t.Error("Expected second item to be visible")
	}
	
	// Last items should not be visible in small viewport
	if pager.IsItemVisible(9) {
		t.Error("Expected last item to not be visible in small viewport")
	}
}

func TestStandardScrollPager_Update(t *testing.T) {
	pager := NewStandardScrollPager()
	
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
		MockScrollableItem{id: "3", content: "Item 3", height: 1, selectable: true},
	}
	
	pager.SetItems(items)
	
	// Test key handling when focused
	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, cmd := pager.Update(msg)
	
	if model != pager {
		t.Error("Expected Update to return the same model")
	}
	
	if cmd != nil {
		t.Error("Expected Update to return nil command for navigation")
	}
	
	if pager.GetSelected() != 1 {
		t.Errorf("Expected selection to move to 1 after down key, got %d", pager.GetSelected())
	}
	
	// Test key handling when not focused
	pager.Blur()
	originalSelection := pager.GetSelected()
	
	msg = tea.KeyMsg{Type: tea.KeyDown}
	pager.Update(msg)
	
	if pager.GetSelected() != originalSelection {
		t.Error("Expected selection to not change when not focused")
	}
}

func TestStandardScrollPager_View(t *testing.T) {
	pager := NewStandardScrollPager()
	pager.SetSize(80, 10)
	
	// Test empty view
	view := pager.View()
	if !strings.Contains(view, "No items to display") {
		t.Error("Expected empty state message in view")
	}
	
	// Add items
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
	}
	
	pager.SetItems(items)
	
	view = pager.View()
	
	// Should contain item content
	if !strings.Contains(view, "Item 1") {
		t.Error("Expected view to contain 'Item 1'")
	}
	
	if !strings.Contains(view, "Item 2") {
		t.Error("Expected view to contain 'Item 2'")
	}
	
	// First item should be selected by default
	if !strings.Contains(view, "[SELECTED] Item 1") {
		t.Error("Expected first item to be marked as selected")
	}
}

func TestStandardScrollPager_TUIComponent(t *testing.T) {
	pager := NewStandardScrollPager()
	
	// Test SetSize
	pager.SetSize(100, 20)
	if pager.width != 100 || pager.height != 20 {
		t.Errorf("Expected size to be 100x20, got %dx%d", pager.width, pager.height)
	}
	
	// Test SetTheme
	theme := NewDefaultTheme()
	pager.SetTheme(theme)
	if pager.content.Theme != theme {
		t.Error("Expected theme to be set on content")
	}
	
	// Test Focus/Blur
	pager.Blur()
	if pager.focused {
		t.Error("Expected pager to be unfocused after Blur")
	}
	
	pager.Focus()
	if !pager.focused {
		t.Error("Expected pager to be focused after Focus")
	}
}

func TestStandardScrollPager_ScrollIndicators(t *testing.T) {
	pager := NewStandardScrollPager()
	pager.SetSize(80, 3) // Very small viewport
	
	// Create many items
	items := make([]ScrollableItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = MockScrollableItem{
			id:         string(rune('a' + i)),
			content:    "Item " + string(rune('A' + i)),
			height:     1,
			selectable: true,
		}
	}
	
	pager.SetItems(items)
	
	// At top - should show down indicator
	view := pager.View()
	if !strings.Contains(view, "▼ More content below") {
		t.Error("Expected down scroll indicator at top")
	}
	
	// Move to middle
	pager.SetSelected(5)
	view = pager.View()
	
	// Should show both indicators
	if !strings.Contains(view, "▲ More content above") {
		t.Error("Expected up scroll indicator in middle")
	}
	
	if !strings.Contains(view, "▼ More content below") {
		t.Error("Expected down scroll indicator in middle")
	}
	
	// Test disabling indicators
	pager.SetShowScrollIndicators(false)
	view = pager.View()
	
	if strings.Contains(view, "▲ More content above") || strings.Contains(view, "▼ More content below") {
		t.Error("Expected no scroll indicators when disabled")
	}
}