// Package tui contains tests for help model functionality
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewHelpModel(t *testing.T) {
	model := NewHelpModel()
	
	if model == nil {
		t.Fatal("NewHelpModel() returned nil")
	}
	
	if model.scrollPager == nil {
		t.Fatal("HelpModel should have a scroll pager")
	}
	
	if !model.focused {
		t.Error("HelpModel should be focused by default")
	}
}

func TestHelpModel_Init(t *testing.T) {
	model := NewHelpModel()
	cmd := model.Init()
	
	if cmd != nil {
		t.Error("HelpModel.Init() should return nil")
	}
}

func TestHelpModel_SetSize(t *testing.T) {
	model := NewHelpModel()
	
	// Set size before ready
	model.SetSize(80, 24)
	if model.width != 80 || model.height != 24 {
		t.Error("SetSize should update width and height")
	}
	
	// Initialize the model
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	
	// Set size after ready
	model.SetSize(100, 30)
	if model.width != 100 || model.height != 30 {
		t.Error("SetSize should update width and height after initialization")
	}
}

func TestHelpModel_SetTheme(t *testing.T) {
	model := NewHelpModel()
	theme := NewDefaultTheme()
	
	model.SetTheme(theme)
	if model.theme != theme {
		t.Error("SetTheme should update the theme")
	}
}

func TestHelpModel_Focus_Blur(t *testing.T) {
	model := NewHelpModel()
	
	// Test focus
	model.Focus()
	if !model.focused {
		t.Error("Focus() should set focused to true")
	}
	
	// Test blur
	model.Blur()
	if model.focused {
		t.Error("Blur() should set focused to false")
	}
}

func TestHelpModel_Update_WindowSize(t *testing.T) {
	model := NewHelpModel()
	
	// Send window size message
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, cmd := model.Update(msg)
	
	if updatedModel == nil {
		t.Fatal("Update should return a model")
	}
	
	if cmd != nil {
		t.Error("Update with WindowSizeMsg should not return a command")
	}
	
	helpModel := updatedModel.(*HelpModel)
	if !helpModel.ready {
		t.Error("HelpModel should be ready after WindowSizeMsg")
	}
	
	// Check that help content was initialized
	items := helpModel.scrollPager.GetItems()
	if len(items) == 0 {
		t.Error("Help content should be initialized after WindowSizeMsg")
	}
}

func TestHelpModel_Update_BackKey(t *testing.T) {
	model := NewHelpModel()
	model.Focus()
	
	// Send back key message
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := model.Update(msg)
	
	if cmd == nil {
		t.Error("Update with back key should return a navigation command")
	}
	
	// Execute the command to get the message
	result := cmd()
	if navMsg, ok := result.(NavigationMsg); !ok || navMsg.Action != NavigationActionBack {
		t.Error("Back key should send NavigationActionBack message")
	}
}

func TestHelpModel_Update_NotFocused(t *testing.T) {
	model := NewHelpModel()
	model.Blur()
	
	// Send key message when not focused
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := model.Update(msg)
	
	if cmd != nil {
		t.Error("Update should not process keys when not focused")
	}
}

func TestHelpModel_View_NotReady(t *testing.T) {
	model := NewHelpModel()
	
	view := model.View()
	if view != "\n  Initializing help..." {
		t.Error("View should show initialization message when not ready")
	}
}

func TestHelpModel_View_Ready(t *testing.T) {
	model := NewHelpModel()
	model.SetSize(80, 24)
	
	// Initialize the model
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	
	view := model.View()
	if view == "" {
		t.Error("View should return content when ready")
	}
	
	// Should contain header
	if !contains(view, "NetTraceX Help") {
		t.Error("View should contain header")
	}
	
	// Should contain footer with scroll info
	if !contains(view, "Press Esc or ? to close") {
		t.Error("View should contain footer")
	}
}

func TestHelpModel_InitializeHelpContent(t *testing.T) {
	model := NewHelpModel()
	model.initializeHelpContent()
	
	items := model.scrollPager.GetItems()
	if len(items) == 0 {
		t.Error("initializeHelpContent should create help sections")
	}
	
	// Check that we have the expected sections
	expectedSections := []string{
		"navigation_&_scrolling",
		"diagnostic_tools", 
		"form_controls",
		"result_views",
		"tips_&_examples",
		"troubleshooting",
	}
	
	if len(items) != len(expectedSections) {
		t.Errorf("Expected %d help sections, got %d", len(expectedSections), len(items))
	}
	
	// Verify each section implements ScrollableItem
	for i, item := range items {
		if !item.IsSelectable() {
			t.Errorf("Help section %d should be selectable", i)
		}
		
		if item.GetHeight() <= 0 {
			t.Errorf("Help section %d should have positive height", i)
		}
		
		if item.GetID() == "" {
			t.Errorf("Help section %d should have an ID", i)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}