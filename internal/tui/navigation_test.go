// Package tui contains tests for navigation components
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewNavigationModel(t *testing.T) {
	model := NewNavigationModel()

	assert.NotNil(t, model)
	assert.NotNil(t, model.scrollPager)
	items := model.scrollPager.GetItems()
	assert.True(t, len(items) > 0)
	assert.Equal(t, 0, model.scrollPager.GetSelected())
	assert.True(t, model.focused)
	assert.Empty(t, model.breadcrumbs)

	// Check that default items are present
	expectedItems := []string{"whois", "ping", "traceroute", "dns", "ssl", "settings"}
	assert.Equal(t, len(expectedItems), len(items))
	
	for i, expectedID := range expectedItems {
		navItem, ok := items[i].(NavigationItem)
		assert.True(t, ok)
		assert.Equal(t, expectedID, navItem.ID)
		assert.True(t, navItem.Enabled)
		assert.NotEmpty(t, navItem.Title)
		assert.NotEmpty(t, navItem.Description)
	}
}

func TestNavigationModel_Init(t *testing.T) {
	model := NewNavigationModel()
	cmd := model.Init()

	assert.Nil(t, cmd)
}

func TestNavigationModel_Update_UpDown(t *testing.T) {
	model := NewNavigationModel()
	model.focused = true

	// Test down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd := model.Update(msg)
	navModel := updatedModel.(*NavigationModel)
	
	assert.Equal(t, 1, navModel.scrollPager.GetSelected())
	assert.Nil(t, cmd)

	// Test up key
	msg = tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd = navModel.Update(msg)
	navModel = updatedModel.(*NavigationModel)
	
	assert.Equal(t, 0, navModel.scrollPager.GetSelected())
	assert.Nil(t, cmd)
}

func TestNavigationModel_Update_UpDown_Wraparound(t *testing.T) {
	model := NewNavigationModel()
	model.focused = true
	items := model.scrollPager.GetItems()

	// Test up key at beginning (StandardScrollPager doesn't wrap, so it should stay at 0)
	msg := tea.KeyMsg{Type: tea.KeyUp}
	updatedModel, cmd := model.Update(msg)
	navModel := updatedModel.(*NavigationModel)
	
	assert.Equal(t, 0, navModel.scrollPager.GetSelected())
	assert.Nil(t, cmd)

	// Test down key at end (StandardScrollPager doesn't wrap, so it should stay at last)
	navModel.scrollPager.SetSelected(len(items) - 1)
	msg = tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, cmd = navModel.Update(msg)
	navModel = updatedModel.(*NavigationModel)
	
	assert.Equal(t, len(items)-1, navModel.scrollPager.GetSelected())
	assert.Nil(t, cmd)
}

func TestNavigationModel_Update_Enter(t *testing.T) {
	model := NewNavigationModel()
	model.focused = true
	model.scrollPager.SetSelected(0)

	// Test enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	navModel := updatedModel.(*NavigationModel)
	
	assert.Equal(t, 0, navModel.scrollPager.GetSelected()) // Selection shouldn't change
	assert.NotNil(t, cmd)

	// Execute the command to get the message
	if cmd != nil {
		result := cmd()
		navMsg, ok := result.(NavigationMsg)
		assert.True(t, ok)
		assert.Equal(t, NavigationActionSelect, navMsg.Action)
		
		item, ok := navMsg.Data.(NavigationItem)
		assert.True(t, ok)
		items := model.scrollPager.GetItems()
		firstItem, _ := items[0].(NavigationItem)
		assert.Equal(t, firstItem.ID, item.ID)
	}
}

func TestNavigationModel_Update_Enter_DisabledItem(t *testing.T) {
	model := NewNavigationModel()
	model.focused = true
	model.scrollPager.SetSelected(0)
	
	// Disable first item
	model.DisableItem("whois")

	// Test enter key on disabled item
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := model.Update(msg)
	navModel := updatedModel.(*NavigationModel)
	
	assert.Equal(t, 0, navModel.scrollPager.GetSelected())
	assert.Nil(t, cmd) // Should not generate command for disabled item
}

func TestNavigationModel_Update_Back(t *testing.T) {
	model := NewNavigationModel()
	model.focused = true
	originalSelected := model.scrollPager.GetSelected()

	// Test escape key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := model.Update(msg)
	navModel := updatedModel.(*NavigationModel)
	
	assert.Equal(t, originalSelected, navModel.scrollPager.GetSelected()) // Selection shouldn't change
	assert.NotNil(t, cmd)

	// Execute the command to get the message
	if cmd != nil {
		result := cmd()
		navMsg, ok := result.(NavigationMsg)
		assert.True(t, ok)
		assert.Equal(t, NavigationActionBack, navMsg.Action)
		assert.Nil(t, navMsg.Data)
	}
}

func TestNavigationModel_Update_NotFocused(t *testing.T) {
	model := NewNavigationModel()
	model.focused = false
	originalSelected := model.scrollPager.GetSelected()

	// Test that keys don't work when not focused
	testKeys := []tea.KeyMsg{
		{Type: tea.KeyUp},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyEsc},
	}

	for _, key := range testKeys {
		updatedModel, cmd := model.Update(key)
		navModel := updatedModel.(*NavigationModel)
		
		assert.Equal(t, originalSelected, navModel.scrollPager.GetSelected())
		assert.Nil(t, cmd)
	}
}

func TestNavigationModel_View(t *testing.T) {
	model := NewNavigationModel()
	model.SetSize(80, 24)

	view := model.View()
	
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Network Diagnostic Tools")
	assert.Contains(t, view, "WHOIS Lookup")
	assert.Contains(t, view, "Ping Test")
}

func TestNavigationModel_View_EmptyItems(t *testing.T) {
	model := NewNavigationModel()
	model.scrollPager.SetItems([]ScrollableItem{}) // Clear items

	view := model.View()
	
	assert.Contains(t, view, "Network Diagnostic Tools") // Title should still be there
	assert.Contains(t, view, "No items to display") // Empty state from StandardScrollPager
}

func TestNavigationModel_SetSize(t *testing.T) {
	model := NewNavigationModel()
	model.SetSize(100, 50)

	// The size should be passed to the scroll pager (minus title space)
	// We can't directly test the internal state, but we can verify it doesn't panic
	assert.NotNil(t, model.scrollPager)
}

func TestNavigationModel_SetTheme(t *testing.T) {
	model := NewNavigationModel()
	theme := &MockTheme{}
	
	model.SetTheme(theme)
	
	assert.Equal(t, theme, model.theme)
}

func TestNavigationModel_Focus_Blur(t *testing.T) {
	model := NewNavigationModel()
	
	model.Blur()
	assert.False(t, model.focused)
	
	model.Focus()
	assert.True(t, model.focused)
}

func TestNavigationModel_GetSelected(t *testing.T) {
	model := NewNavigationModel()
	model.scrollPager.SetSelected(1)

	selected := model.GetSelected()
	assert.NotNil(t, selected)
	items := model.scrollPager.GetItems()
	expectedItem, _ := items[1].(NavigationItem)
	assert.Equal(t, expectedItem.ID, selected.ID)
}

func TestNavigationModel_GetSelected_InvalidIndex(t *testing.T) {
	model := NewNavigationModel()
	
	// StandardScrollPager handles invalid indices internally, so we test edge cases
	items := model.scrollPager.GetItems()
	
	// Test with valid index at boundary
	model.scrollPager.SetSelected(len(items) - 1)
	selected := model.GetSelected()
	assert.NotNil(t, selected)
	
	// Test with index beyond bounds (StandardScrollPager should clamp it)
	model.scrollPager.SetSelected(len(items) + 10)
	selected = model.GetSelected()
	assert.NotNil(t, selected) // Should still return the last valid item
}

func TestNavigationModel_SetSelected(t *testing.T) {
	model := NewNavigationModel()
	
	model.SetSelected(2)
	assert.Equal(t, 2, model.scrollPager.GetSelected())
	
	// StandardScrollPager handles invalid indices by clamping them
	model.SetSelected(-1)
	assert.Equal(t, 0, model.scrollPager.GetSelected()) // Should clamp to 0
	
	items := model.scrollPager.GetItems()
	model.SetSelected(len(items) + 10)
	assert.Equal(t, len(items)-1, model.scrollPager.GetSelected()) // Should clamp to last valid index
}

func TestNavigationModel_Breadcrumbs(t *testing.T) {
	model := NewNavigationModel()
	
	// Test adding breadcrumbs
	model.AddBreadcrumb("Home")
	model.AddBreadcrumb("Tools")
	model.AddBreadcrumb("WHOIS")
	
	breadcrumbs := model.GetBreadcrumbs()
	expected := []string{"Home", "Tools", "WHOIS"}
	assert.Equal(t, expected, breadcrumbs)
	
	// Test popping breadcrumbs
	popped := model.PopBreadcrumb()
	assert.Equal(t, "WHOIS", popped)
	
	breadcrumbs = model.GetBreadcrumbs()
	expected = []string{"Home", "Tools"}
	assert.Equal(t, expected, breadcrumbs)
	
	// Test popping from empty
	model.breadcrumbs = []string{}
	popped = model.PopBreadcrumb()
	assert.Equal(t, "", popped)
}

func TestNavigationModel_EnableDisableItem(t *testing.T) {
	model := NewNavigationModel()
	
	// Test disabling item
	model.DisableItem("whois")
	items := model.scrollPager.GetItems()
	for _, item := range items {
		if navItem, ok := item.(NavigationItem); ok && navItem.ID == "whois" {
			assert.False(t, navItem.Enabled)
		}
	}
	
	// Test enabling item
	model.EnableItem("whois")
	items = model.scrollPager.GetItems()
	for _, item := range items {
		if navItem, ok := item.(NavigationItem); ok && navItem.ID == "whois" {
			assert.True(t, navItem.Enabled)
		}
	}
	
	// Test with non-existent item
	model.DisableItem("nonexistent")
	// Should not panic
}

func TestNavigationModel_AddRemoveItem(t *testing.T) {
	model := NewNavigationModel()
	originalItems := model.scrollPager.GetItems()
	originalCount := len(originalItems)
	
	// Test adding item
	newItem := NavigationItem{
		ID:          "test",
		Title:       "Test Tool",
		Description: "Test description",
		Enabled:     true,
	}
	model.AddItem(newItem)
	
	items := model.scrollPager.GetItems()
	assert.Equal(t, originalCount+1, len(items))
	lastItem, ok := items[len(items)-1].(NavigationItem)
	assert.True(t, ok)
	assert.Equal(t, "test", lastItem.ID)
	
	// Test removing item
	model.RemoveItem("test")
	items = model.scrollPager.GetItems()
	assert.Equal(t, originalCount, len(items))
	
	// Verify item is actually removed
	for _, item := range items {
		if navItem, ok := item.(NavigationItem); ok {
			assert.NotEqual(t, "test", navItem.ID)
		}
	}
	
	// Test removing non-existent item
	model.RemoveItem("nonexistent")
	items = model.scrollPager.GetItems()
	assert.Equal(t, originalCount, len(items))
}

func TestNavigationModel_RemoveItem_AdjustSelection(t *testing.T) {
	model := NewNavigationModel()
	items := model.scrollPager.GetItems()
	model.scrollPager.SetSelected(len(items) - 1) // Select last item
	lastItem, _ := items[len(items)-1].(NavigationItem)
	lastItemID := lastItem.ID
	
	// Remove the last item
	model.RemoveItem(lastItemID)
	
	// Selection should be adjusted by StandardScrollPager
	newItems := model.scrollPager.GetItems()
	assert.Equal(t, len(newItems)-1, model.scrollPager.GetSelected())
	
	// Test removing all items
	for len(model.scrollPager.GetItems()) > 0 {
		items := model.scrollPager.GetItems()
		firstItem, _ := items[0].(NavigationItem)
		model.RemoveItem(firstItem.ID)
	}
	
	assert.Equal(t, 0, model.scrollPager.GetSelected())
}