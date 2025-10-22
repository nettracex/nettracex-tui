// Package tui contains tests for core scrolling interfaces and types
package tui

import (
	"testing"

	"github.com/nettracex/nettracex-tui/internal/domain"
)



// MockScrollableItem implements ScrollableItem for testing
type MockScrollableItem struct {
	id         string
	content    string
	height     int
	selectable bool
}

func (m MockScrollableItem) Render(width int, selected bool, theme domain.Theme) string {
	if selected {
		return "[SELECTED] " + m.content
	}
	return m.content
}

func (m MockScrollableItem) GetHeight() int {
	return m.height
}

func (m MockScrollableItem) IsSelectable() bool {
	return m.selectable
}

func (m MockScrollableItem) GetID() string {
	return m.id
}

func TestNewScrollPosition(t *testing.T) {
	pos := NewScrollPosition()
	
	if pos.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex to be 0, got %d", pos.SelectedIndex)
	}
	
	if pos.TopVisible != 0 {
		t.Errorf("Expected TopVisible to be 0, got %d", pos.TopVisible)
	}
	
	if pos.ViewportHeight != 1 {
		t.Errorf("Expected ViewportHeight to be 1, got %d", pos.ViewportHeight)
	}
}

func TestScrollPosition_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		pos       ScrollPosition
		itemCount int
		expected  bool
	}{
		{
			name: "valid position with items",
			pos: ScrollPosition{
				SelectedIndex:  1,
				TopVisible:     0,
				ViewportHeight: 3,
			},
			itemCount: 5,
			expected:  true,
		},
		{
			name: "valid position empty list",
			pos: ScrollPosition{
				SelectedIndex:  0,
				TopVisible:     0,
				ViewportHeight: 1,
			},
			itemCount: 0,
			expected:  true,
		},
		{
			name: "invalid selected index too high",
			pos: ScrollPosition{
				SelectedIndex:  5,
				TopVisible:     0,
				ViewportHeight: 3,
			},
			itemCount: 3,
			expected:  false,
		},
		{
			name: "invalid selected index negative",
			pos: ScrollPosition{
				SelectedIndex:  -1,
				TopVisible:     0,
				ViewportHeight: 3,
			},
			itemCount: 5,
			expected:  false,
		},
		{
			name: "invalid viewport height zero",
			pos: ScrollPosition{
				SelectedIndex:  1,
				TopVisible:     0,
				ViewportHeight: 0,
			},
			itemCount: 5,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pos.IsValid(tt.itemCount)
			if result != tt.expected {
				t.Errorf("Expected IsValid to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestScrollPosition_EnsureSelectionVisible(t *testing.T) {
	tests := []struct {
		name           string
		pos            ScrollPosition
		itemCount      int
		expectedTop    int
		expectedSelect int
	}{
		{
			name: "selection above viewport",
			pos: ScrollPosition{
				SelectedIndex:  1,
				TopVisible:     3,
				ViewportHeight: 2,
			},
			itemCount:      10,
			expectedTop:    1,
			expectedSelect: 1,
		},
		{
			name: "selection below viewport",
			pos: ScrollPosition{
				SelectedIndex:  5,
				TopVisible:     0,
				ViewportHeight: 3,
			},
			itemCount:      10,
			expectedTop:    3,
			expectedSelect: 5,
		},
		{
			name: "selection within viewport",
			pos: ScrollPosition{
				SelectedIndex:  2,
				TopVisible:     1,
				ViewportHeight: 3,
			},
			itemCount:      10,
			expectedTop:    1,
			expectedSelect: 2,
		},
		{
			name: "empty list",
			pos: ScrollPosition{
				SelectedIndex:  5,
				TopVisible:     3,
				ViewportHeight: 2,
			},
			itemCount:      0,
			expectedTop:    0,
			expectedSelect: 0,
		},
		{
			name: "selection beyond items",
			pos: ScrollPosition{
				SelectedIndex:  10,
				TopVisible:     0,
				ViewportHeight: 3,
			},
			itemCount:      5,
			expectedTop:    2,
			expectedSelect: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := tt.pos
			pos.EnsureSelectionVisible(tt.itemCount)
			
			if pos.TopVisible != tt.expectedTop {
				t.Errorf("Expected TopVisible to be %d, got %d", tt.expectedTop, pos.TopVisible)
			}
			
			if pos.SelectedIndex != tt.expectedSelect {
				t.Errorf("Expected SelectedIndex to be %d, got %d", tt.expectedSelect, pos.SelectedIndex)
			}
		})
	}
}

func TestScrollPosition_CanScroll(t *testing.T) {
	pos := ScrollPosition{
		SelectedIndex:  2,
		TopVisible:     1,
		ViewportHeight: 3,
	}
	
	// Can scroll up when TopVisible > 0
	if !pos.CanScrollUp() {
		t.Error("Expected CanScrollUp to return true when TopVisible > 0")
	}
	
	// Can scroll down when there are more items below viewport
	if !pos.CanScrollDown(10) {
		t.Error("Expected CanScrollDown to return true when there are items below viewport")
	}
	
	// Cannot scroll up when at top
	pos.TopVisible = 0
	if pos.CanScrollUp() {
		t.Error("Expected CanScrollUp to return false when TopVisible = 0")
	}
	
	// Cannot scroll down when all items visible (TopVisible=1, ViewportHeight=3, so can see items 1,2,3 out of 4 total)
	// This should still be able to scroll down since item 0 is not visible
	// Let's test with a case where all items are truly visible
	pos.TopVisible = 0
	pos.ViewportHeight = 5 // Can see all 4 items (0,1,2,3)
	if pos.CanScrollDown(4) {
		t.Error("Expected CanScrollDown to return false when all items are visible")
	}
}

func TestScrollPosition_GetVisibleRange(t *testing.T) {
	pos := ScrollPosition{
		SelectedIndex:  2,
		TopVisible:     1,
		ViewportHeight: 3,
	}
	
	start, end := pos.GetVisibleRange(10)
	
	if start != 1 {
		t.Errorf("Expected start to be 1, got %d", start)
	}
	
	if end != 4 {
		t.Errorf("Expected end to be 4, got %d", end)
	}
	
	// Test with fewer items than viewport
	start, end = pos.GetVisibleRange(2)
	
	if start != 1 {
		t.Errorf("Expected start to be 1, got %d", start)
	}
	
	if end != 2 {
		t.Errorf("Expected end to be 2, got %d", end)
	}
}

func TestScrollPosition_IsItemVisible(t *testing.T) {
	pos := ScrollPosition{
		SelectedIndex:  2,
		TopVisible:     1,
		ViewportHeight: 3,
	}
	
	// Item within visible range
	if !pos.IsItemVisible(2) {
		t.Error("Expected item 2 to be visible")
	}
	
	// Item before visible range
	if pos.IsItemVisible(0) {
		t.Error("Expected item 0 to not be visible")
	}
	
	// Item after visible range
	if pos.IsItemVisible(5) {
		t.Error("Expected item 5 to not be visible")
	}
}

func TestNewScrollableContent(t *testing.T) {
	content := NewScrollableContent()
	
	if content == nil {
		t.Fatal("Expected NewScrollableContent to return non-nil")
	}
	
	if len(content.Items) != 0 {
		t.Errorf("Expected Items to be empty, got length %d", len(content.Items))
	}
	
	if !content.ShowIndicators {
		t.Error("Expected ShowIndicators to be true by default")
	}
	
	if content.Position.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex to be 0, got %d", content.Position.SelectedIndex)
	}
}

func TestScrollableContent_AddItem(t *testing.T) {
	content := NewScrollableContent()
	item := MockScrollableItem{
		id:         "test1",
		content:    "Test Item 1",
		height:     1,
		selectable: true,
	}
	
	content.AddItem(item)
	
	if len(content.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(content.Items))
	}
	
	if content.Items[0].GetID() != "test1" {
		t.Errorf("Expected item ID to be 'test1', got '%s'", content.Items[0].GetID())
	}
}

func TestScrollableContent_RemoveItem(t *testing.T) {
	content := NewScrollableContent()
	
	// Add some items
	for i := 0; i < 3; i++ {
		item := MockScrollableItem{
			id:         string(rune('a' + i)),
			content:    "Test Item",
			height:     1,
			selectable: true,
		}
		content.AddItem(item)
	}
	
	// Set selection to last item
	content.Position.SelectedIndex = 2
	
	// Remove middle item
	content.RemoveItem(1)
	
	if len(content.Items) != 2 {
		t.Errorf("Expected 2 items after removal, got %d", len(content.Items))
	}
	
	// Selection should be adjusted
	if content.Position.SelectedIndex != 1 {
		t.Errorf("Expected SelectedIndex to be adjusted to 1, got %d", content.Position.SelectedIndex)
	}
	
	// Remove invalid index should not crash
	content.RemoveItem(10)
	if len(content.Items) != 2 {
		t.Errorf("Expected items count to remain 2 after invalid removal, got %d", len(content.Items))
	}
}

func TestScrollableContent_SetItems(t *testing.T) {
	content := NewScrollableContent()
	
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
		MockScrollableItem{id: "3", content: "Item 3", height: 1, selectable: true},
	}
	
	content.SetItems(items)
	
	if len(content.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(content.Items))
	}
	
	if content.Position.SelectedIndex != 0 {
		t.Errorf("Expected SelectedIndex to be reset to 0, got %d", content.Position.SelectedIndex)
	}
	
	if content.Position.TopVisible != 0 {
		t.Errorf("Expected TopVisible to be reset to 0, got %d", content.Position.TopVisible)
	}
}

func TestScrollableContent_GetSelectedItem(t *testing.T) {
	content := NewScrollableContent()
	
	// No items - should return nil
	if content.GetSelectedItem() != nil {
		t.Error("Expected GetSelectedItem to return nil when no items")
	}
	
	// Add items
	items := []ScrollableItem{
		MockScrollableItem{id: "1", content: "Item 1", height: 1, selectable: true},
		MockScrollableItem{id: "2", content: "Item 2", height: 1, selectable: true},
	}
	content.SetItems(items)
	
	// Should return first item by default
	selected := content.GetSelectedItem()
	if selected == nil {
		t.Fatal("Expected GetSelectedItem to return non-nil")
	}
	
	if selected.GetID() != "1" {
		t.Errorf("Expected selected item ID to be '1', got '%s'", selected.GetID())
	}
	
	// Change selection
	content.Position.SelectedIndex = 1
	selected = content.GetSelectedItem()
	if selected.GetID() != "2" {
		t.Errorf("Expected selected item ID to be '2', got '%s'", selected.GetID())
	}
}

func TestScrollableContent_SetViewportHeight(t *testing.T) {
	content := NewScrollableContent()
	
	// Test setting valid height
	content.SetViewportHeight(5)
	if content.Position.ViewportHeight != 5 {
		t.Errorf("Expected ViewportHeight to be 5, got %d", content.Position.ViewportHeight)
	}
	
	// Test setting invalid height (should be clamped to 1)
	content.SetViewportHeight(0)
	if content.Position.ViewportHeight != 1 {
		t.Errorf("Expected ViewportHeight to be clamped to 1, got %d", content.Position.ViewportHeight)
	}
	
	content.SetViewportHeight(-5)
	if content.Position.ViewportHeight != 1 {
		t.Errorf("Expected ViewportHeight to be clamped to 1, got %d", content.Position.ViewportHeight)
	}
}