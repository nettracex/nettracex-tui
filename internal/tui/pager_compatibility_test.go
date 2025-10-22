package tui

import (
	"testing"
)

func TestPagerScrollableAdapter(t *testing.T) {
	// Create a regular pager with small height to force scrolling
	pager := NewPager()
	pager.SetSize(80, 3) // Small height to force scrolling with more content
	
	// Wrap it with the adapter
	adapter := NewPagerScrollableAdapter(pager)
	
	// Test that it implements ScrollableList interface
	var _ ScrollableList = adapter
	
	// Test adding more items to force scrolling
	for i := 1; i <= 10; i++ {
		item := NewStringScrollableItem("Line "+string(rune('0'+i)), string(rune('0'+i)))
		adapter.AddItem(item)
	}
	
	// Test getting items
	items := adapter.GetItems()
	if len(items) != 10 {
		t.Errorf("Expected 10 items, got %d", len(items))
	}
	
	// Test initial state
	initialSelected := adapter.GetSelected()
	if initialSelected != 0 {
		t.Errorf("Expected initial selected 0, got %d", initialSelected)
	}
	
	// Test navigation - should be able to scroll down
	moved := adapter.MoveDown()
	if !moved {
		t.Error("Expected MoveDown to return true with scrollable content")
	}
	
	selected := adapter.GetSelected()
	if selected <= initialSelected {
		t.Errorf("Expected selected to increase after MoveDown, got %d", selected)
	}
	
	// Test scroll position
	pos := adapter.GetScrollPosition()
	if pos.SelectedIndex != selected {
		t.Errorf("Expected scroll position selected index %d, got %d", selected, pos.SelectedIndex)
	}
	
	// Test Home navigation
	adapter.Home()
	if adapter.GetSelected() != 0 {
		t.Errorf("Expected Home to go to position 0, got %d", adapter.GetSelected())
	}
}



func TestStringScrollableItem(t *testing.T) {
	item := NewStringScrollableItem("Test content", "test-id")
	
	// Test interface implementation
	var _ ScrollableItem = item
	
	// Test methods
	if item.GetID() != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", item.GetID())
	}
	
	if item.GetHeight() != 1 {
		t.Errorf("Expected height 1, got %d", item.GetHeight())
	}
	
	if item.IsSelectable() {
		t.Error("Expected string items to not be selectable by default")
	}
	
	theme := mockTheme{}
	rendered := item.Render(80, false, theme)
	if rendered != "Test content" {
		t.Errorf("Expected 'Test content', got '%s'", rendered)
	}
}

func TestScrollableViewScrollableListMode(t *testing.T) {
	view := NewScrollableView()
	view.SetSize(80, 10)
	
	// Test that it implements ScrollableList interface
	var _ ScrollableList = view
	
	// Test adding items
	item1 := NewStringScrollableItem("Item 1", "1")
	item2 := NewStringScrollableItem("Item 2", "2")
	
	view.AddItem(item1)
	view.AddItem(item2)
	
	// Verify scrollable list mode is enabled
	if !view.useScrollableList {
		t.Error("Expected scrollable list mode to be enabled")
	}
	
	// Test getting items
	items := view.GetItems()
	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}
	
	// Test selection
	if view.GetSelected() != 0 {
		t.Errorf("Expected initial selection 0, got %d", view.GetSelected())
	}
	
	view.SetSelected(1)
	if view.GetSelected() != 1 {
		t.Errorf("Expected selection 1, got %d", view.GetSelected())
	}
}

func TestScrollableViewLegacyMode(t *testing.T) {
	view := NewScrollableView()
	view.SetSize(80, 10)
	
	// Set string content (legacy mode)
	view.SetContent("Line 1\nLine 2\nLine 3")
	
	// Verify scrollable list mode is disabled
	if view.useScrollableList {
		t.Error("Expected scrollable list mode to be disabled in legacy mode")
	}
	
	// Test that content is set
	if view.content != "Line 1\nLine 2\nLine 3" {
		t.Errorf("Expected content to be set correctly, got '%s'", view.content)
	}
}

func TestScrollableViewModeTransition(t *testing.T) {
	view := NewScrollableView()
	view.SetSize(80, 10)
	
	// Start with string content
	view.SetContent("String content")
	if view.useScrollableList {
		t.Error("Expected scrollable list mode to be disabled")
	}
	
	// Switch to scrollable list mode
	item := NewStringScrollableItem("Item 1", "1")
	view.AddItem(item)
	if !view.useScrollableList {
		t.Error("Expected scrollable list mode to be enabled")
	}
	
	// Switch back to string mode
	view.SetContent("New string content")
	if view.useScrollableList {
		t.Error("Expected scrollable list mode to be disabled after SetContent")
	}
}