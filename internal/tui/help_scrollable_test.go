package tui

import (
	"strings"
	"testing"
)

// mockTheme implements Theme interface for testing
type mockTheme struct{}

func (m mockTheme) GetColor(element string) string {
	return "#ffffff"
}

func (m mockTheme) GetStyle(element string) map[string]interface{} {
	return make(map[string]interface{})
}

func (m mockTheme) SetColor(element, color string) {
	// No-op for testing
}

func TestNewHelpSection(t *testing.T) {
	items := []HelpItem{
		NewHelpItem("↑/↓", "Navigate up/down"),
		NewHelpItem("Enter", "Select item"),
	}
	
	section := NewHelpSection("Test Section", items)
	
	if section.Title != "Test Section" {
		t.Errorf("Expected title 'Test Section', got '%s'", section.Title)
	}
	
	if len(section.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(section.Items))
	}
	
	expectedID := "test_section"
	if section.ID != expectedID {
		t.Errorf("Expected ID '%s', got '%s'", expectedID, section.ID)
	}
}

func TestNewHelpItem(t *testing.T) {
	item := NewHelpItem("Ctrl+C", "Quit application")
	
	if item.Key != "Ctrl+C" {
		t.Errorf("Expected key 'Ctrl+C', got '%s'", item.Key)
	}
	
	if item.Description != "Quit application" {
		t.Errorf("Expected description 'Quit application', got '%s'", item.Description)
	}
}

func TestHelpSection_GetID(t *testing.T) {
	section := NewHelpSection("Navigation & Scrolling", []HelpItem{})
	expectedID := "navigation_&_scrolling"
	
	if section.GetID() != expectedID {
		t.Errorf("Expected ID '%s', got '%s'", expectedID, section.GetID())
	}
}

func TestHelpSection_IsSelectable(t *testing.T) {
	section := NewHelpSection("Test", []HelpItem{})
	
	if !section.IsSelectable() {
		t.Error("Expected help section to be selectable")
	}
}

func TestHelpSection_GetHeight(t *testing.T) {
	tests := []struct {
		name          string
		itemCount     int
		expectedHeight int
	}{
		{
			name:          "empty section",
			itemCount:     0,
			expectedHeight: 2, // title + spacing
		},
		{
			name:          "single item",
			itemCount:     1,
			expectedHeight: 3, // title + 1 item + spacing
		},
		{
			name:          "multiple items",
			itemCount:     5,
			expectedHeight: 7, // title + 5 items + spacing
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]HelpItem, tt.itemCount)
			for i := 0; i < tt.itemCount; i++ {
				items[i] = NewHelpItem("key", "description")
			}
			
			section := NewHelpSection("Test", items)
			height := section.GetHeight()
			
			if height != tt.expectedHeight {
				t.Errorf("Expected height %d, got %d", tt.expectedHeight, height)
			}
		})
	}
}

func TestHelpSection_Render(t *testing.T) {
	items := []HelpItem{
		NewHelpItem("↑/↓", "Navigate up/down"),
		NewHelpItem("Enter", "Select item"),
	}
	section := NewHelpSection("Navigation", items)
	theme := mockTheme{} // Default theme
	
	// Test unselected rendering
	rendered := section.Render(80, false, theme)
	
	if !strings.Contains(rendered, "Navigation") {
		t.Error("Rendered content should contain section title")
	}
	
	if !strings.Contains(rendered, "↑/↓") {
		t.Error("Rendered content should contain first item key")
	}
	
	if !strings.Contains(rendered, "Navigate up/down") {
		t.Error("Rendered content should contain first item description")
	}
	
	if !strings.Contains(rendered, "Enter") {
		t.Error("Rendered content should contain second item key")
	}
	
	if !strings.Contains(rendered, "Select item") {
		t.Error("Rendered content should contain second item description")
	}
	
	// Test selected rendering
	selectedRendered := section.Render(80, true, theme)
	
	if !strings.Contains(selectedRendered, "Navigation") {
		t.Error("Selected rendered content should contain section title")
	}
	
	// Selected rendering should be different from unselected
	if rendered == selectedRendered {
		t.Error("Selected rendering should be different from unselected")
	}
}

func TestHelpSection_RenderEmptySection(t *testing.T) {
	section := NewHelpSection("Empty Section", []HelpItem{})
	theme := mockTheme{}
	
	rendered := section.Render(80, false, theme)
	
	if !strings.Contains(rendered, "Empty Section") {
		t.Error("Rendered content should contain section title even when empty")
	}
	
	// Should only contain title and newlines, no item content
	lines := strings.Split(strings.TrimSpace(rendered), "\n")
	if len(lines) < 1 {
		t.Error("Should have at least title line")
	}
}

func TestCreateHelpSections(t *testing.T) {
	sections := CreateHelpSections()
	
	if len(sections) == 0 {
		t.Error("Should create at least one help section")
	}
	
	// Verify all sections implement ScrollableItem
	for i, section := range sections {
		if section == nil {
			t.Errorf("Section %d should not be nil", i)
			continue
		}
		
		// Test ScrollableItem interface methods
		if section.GetID() == "" {
			t.Errorf("Section %d should have non-empty ID", i)
		}
		
		if section.GetHeight() <= 0 {
			t.Errorf("Section %d should have positive height", i)
		}
		
		if !section.IsSelectable() {
			t.Errorf("Section %d should be selectable", i)
		}
		
		// Test rendering doesn't panic
		rendered := section.Render(80, false, mockTheme{})
		if rendered == "" {
			t.Errorf("Section %d should render non-empty content", i)
		}
	}
	
	// Verify expected sections exist
	expectedSections := []string{
		"navigation_&_scrolling",
		"diagnostic_tools", 
		"form_controls",
		"result_views",
		"tips_&_examples",
		"troubleshooting",
	}
	
	for _, expectedID := range expectedSections {
		found := false
		for _, section := range sections {
			if section.GetID() == expectedID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected section with ID '%s' not found", expectedID)
		}
	}
}

func TestGetHelpSectionByID(t *testing.T) {
	sections := CreateHelpSections()
	
	// Test finding existing section
	section := GetHelpSectionByID(sections, "navigation_&_scrolling")
	if section == nil {
		t.Error("Should find navigation section")
	} else if section.Title != "Navigation & Scrolling" {
		t.Errorf("Expected title 'Navigation & Scrolling', got '%s'", section.Title)
	}
	
	// Test finding non-existent section
	notFound := GetHelpSectionByID(sections, "non_existent")
	if notFound != nil {
		t.Error("Should return nil for non-existent section")
	}
	
	// Test with empty sections
	emptyNotFound := GetHelpSectionByID([]ScrollableItem{}, "any_id")
	if emptyNotFound != nil {
		t.Error("Should return nil when searching empty sections")
	}
}

func TestCalculateTotalHelpHeight(t *testing.T) {
	sections := CreateHelpSections()
	
	totalHeight := CalculateTotalHelpHeight(sections)
	
	if totalHeight <= 0 {
		t.Error("Total height should be positive")
	}
	
	// Verify it matches sum of individual heights
	expectedTotal := 0
	for _, section := range sections {
		expectedTotal += section.GetHeight()
	}
	
	if totalHeight != expectedTotal {
		t.Errorf("Expected total height %d, got %d", expectedTotal, totalHeight)
	}
	
	// Test with empty sections
	emptyTotal := CalculateTotalHelpHeight([]ScrollableItem{})
	if emptyTotal != 0 {
		t.Errorf("Expected 0 height for empty sections, got %d", emptyTotal)
	}
}

func TestHelpSection_ScrollableItemInterface(t *testing.T) {
	// Verify HelpSection implements ScrollableItem interface
	var _ ScrollableItem = &HelpSection{}
	
	section := NewHelpSection("Test", []HelpItem{
		NewHelpItem("key", "description"),
	})
	
	// Test all interface methods
	id := section.GetID()
	if id == "" {
		t.Error("GetID should return non-empty string")
	}
	
	height := section.GetHeight()
	if height <= 0 {
		t.Error("GetHeight should return positive value")
	}
	
	selectable := section.IsSelectable()
	if !selectable {
		t.Error("IsSelectable should return true for help sections")
	}
	
	rendered := section.Render(80, false, mockTheme{})
	if rendered == "" {
		t.Error("Render should return non-empty string")
	}
}

func TestHelpSection_RenderWithDifferentWidths(t *testing.T) {
	items := []HelpItem{
		NewHelpItem("Very long key name", "Very long description that might wrap"),
	}
	section := NewHelpSection("Test", items)
	theme := mockTheme{}
	
	// Test with different widths
	widths := []int{20, 40, 80, 120}
	
	for _, width := range widths {
		rendered := section.Render(width, false, theme)
		if rendered == "" {
			t.Errorf("Should render content for width %d", width)
		}
		
		// Content should contain the key and description regardless of width
		if !strings.Contains(rendered, "Very long key name") {
			t.Errorf("Should contain key for width %d", width)
		}
		
		if !strings.Contains(rendered, "Very long description") {
			t.Errorf("Should contain description for width %d", width)
		}
	}
}

func TestHelpSection_IDGeneration(t *testing.T) {
	tests := []struct {
		title      string
		expectedID string
	}{
		{
			title:      "Simple Title",
			expectedID: "simple_title",
		},
		{
			title:      "Title With Spaces",
			expectedID: "title_with_spaces",
		},
		{
			title:      "Title & With Special Characters!",
			expectedID: "title_&_with_special_characters!",
		},
		{
			title:      "UPPERCASE TITLE",
			expectedID: "uppercase_title",
		},
		{
			title:      "Mixed Case Title",
			expectedID: "mixed_case_title",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			section := NewHelpSection(tt.title, []HelpItem{})
			if section.GetID() != tt.expectedID {
				t.Errorf("Expected ID '%s', got '%s'", tt.expectedID, section.GetID())
			}
		})
	}
}