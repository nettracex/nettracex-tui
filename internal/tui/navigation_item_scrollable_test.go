// Package tui contains tests for NavigationItem ScrollableItem implementation
package tui

import (
	"testing"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
)

// TestTheme implements domain.Theme for testing NavigationItem
type TestTheme struct {
	colors map[string]string
	styles map[string]map[string]interface{}
}

func NewTestTheme() *TestTheme {
	return &TestTheme{
		colors: map[string]string{
			"primary":    "62",
			"highlight":  "230",
			"foreground": "252",
			"muted":      "243",
		},
		styles: make(map[string]map[string]interface{}),
	}
}

func (t *TestTheme) GetColor(element string) string {
	if color, exists := t.colors[element]; exists {
		return color
	}
	return "252" // Default foreground
}

func (t *TestTheme) GetStyle(element string) map[string]interface{} {
	if style, exists := t.styles[element]; exists {
		return style
	}
	return make(map[string]interface{})
}

func (t *TestTheme) SetColor(element, color string) {
	t.colors[element] = color
}

func TestNavigationItem_Render(t *testing.T) {
	tests := []struct {
		name     string
		item     NavigationItem
		width    int
		selected bool
		theme    domain.Theme
		wantContains []string
	}{
		{
			name: "basic item with icon and description",
			item: NavigationItem{
				ID:          "test",
				Title:       "Test Item",
				Description: "Test description",
				Icon:        "🔍",
				Enabled:     true,
			},
			width:    80,
			selected: false,
			theme:    NewTestTheme(),
			wantContains: []string{
				"🔍 Test Item",
				"Test description",
			},
		},
		{
			name: "selected item styling",
			item: NavigationItem{
				ID:          "selected",
				Title:       "Selected Item",
				Description: "Selected description",
				Icon:        "📡",
				Enabled:     true,
			},
			width:    80,
			selected: true,
			theme:    NewTestTheme(),
			wantContains: []string{
				"📡 Selected Item",
				"Selected description",
			},
		},
		{
			name: "disabled item",
			item: NavigationItem{
				ID:          "disabled",
				Title:       "Disabled Item",
				Description: "Disabled description",
				Icon:        "🔒",
				Enabled:     false,
			},
			width:    80,
			selected: false,
			theme:    NewTestTheme(),
			wantContains: []string{
				"🔒 Disabled Item (disabled)",
				"Disabled description",
			},
		},
		{
			name: "item without icon uses default bullet",
			item: NavigationItem{
				ID:          "no-icon",
				Title:       "No Icon Item",
				Description: "No icon description",
				Icon:        "",
				Enabled:     true,
			},
			width:    80,
			selected: false,
			theme:    NewTestTheme(),
			wantContains: []string{
				"• No Icon Item",
				"No icon description",
			},
		},
		{
			name: "item without description",
			item: NavigationItem{
				ID:          "no-desc",
				Title:       "No Description",
				Description: "",
				Icon:        "⚙️",
				Enabled:     true,
			},
			width:    80,
			selected: false,
			theme:    NewTestTheme(),
			wantContains: []string{
				"⚙️ No Description",
			},
		},
		{
			name: "item with nil theme uses fallback styling",
			item: NavigationItem{
				ID:          "nil-theme",
				Title:       "Nil Theme Item",
				Description: "Nil theme description",
				Icon:        "🌐",
				Enabled:     true,
			},
			width:    80,
			selected: false,
			theme:    nil,
			wantContains: []string{
				"🌐 Nil Theme Item",
				"Nil theme description",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.Render(tt.width, tt.selected, tt.theme)
			
			// Check that all expected content is present
			for _, expected := range tt.wantContains {
				assert.Contains(t, result, expected, "Rendered output should contain expected text")
			}
			
			// Verify the result is not empty
			assert.NotEmpty(t, result, "Rendered output should not be empty")
		})
	}
}

func TestNavigationItem_GetHeight(t *testing.T) {
	tests := []struct {
		name         string
		item         NavigationItem
		expectedHeight int
	}{
		{
			name: "item with description",
			item: NavigationItem{
				ID:          "with-desc",
				Title:       "Item with Description",
				Description: "This item has a description",
				Icon:        "🔍",
				Enabled:     true,
			},
			expectedHeight: 3, // title line + description line + spacing line
		},
		{
			name: "item without description",
			item: NavigationItem{
				ID:          "no-desc",
				Title:       "Item without Description",
				Description: "",
				Icon:        "📡",
				Enabled:     true,
			},
			expectedHeight: 2, // title line + spacing line
		},
		{
			name: "disabled item with description",
			item: NavigationItem{
				ID:          "disabled-desc",
				Title:       "Disabled Item",
				Description: "This disabled item has a description",
				Icon:        "🔒",
				Enabled:     false,
			},
			expectedHeight: 3, // title line + description line + spacing line
		},
		{
			name: "item with empty description",
			item: NavigationItem{
				ID:          "empty-desc",
				Title:       "Item with Empty Description",
				Description: "",
				Icon:        "⚙️",
				Enabled:     true,
			},
			expectedHeight: 2, // title line + spacing line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			height := tt.item.GetHeight()
			assert.Equal(t, tt.expectedHeight, height, "Height should match expected value")
		})
	}
}

func TestNavigationItem_IsSelectable(t *testing.T) {
	tests := []struct {
		name     string
		item     NavigationItem
		expected bool
	}{
		{
			name: "enabled item is selectable",
			item: NavigationItem{
				ID:      "enabled",
				Title:   "Enabled Item",
				Enabled: true,
			},
			expected: true,
		},
		{
			name: "disabled item is not selectable",
			item: NavigationItem{
				ID:      "disabled",
				Title:   "Disabled Item",
				Enabled: false,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selectable := tt.item.IsSelectable()
			assert.Equal(t, tt.expected, selectable, "Selectability should match enabled state")
		})
	}
}

func TestNavigationItem_GetID(t *testing.T) {
	tests := []struct {
		name     string
		item     NavigationItem
		expected string
	}{
		{
			name: "returns correct ID",
			item: NavigationItem{
				ID:    "test-id",
				Title: "Test Item",
			},
			expected: "test-id",
		},
		{
			name: "returns empty ID",
			item: NavigationItem{
				ID:    "",
				Title: "Item with Empty ID",
			},
			expected: "",
		},
		{
			name: "returns complex ID",
			item: NavigationItem{
				ID:    "complex-id-with-dashes_and_underscores.123",
				Title: "Complex ID Item",
			},
			expected: "complex-id-with-dashes_and_underscores.123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.item.GetID()
			assert.Equal(t, tt.expected, id, "ID should match expected value")
		})
	}
}

func TestNavigationItem_ScrollableItemInterface(t *testing.T) {
	// Test that NavigationItem implements ScrollableItem interface
	var item ScrollableItem = NavigationItem{
		ID:          "interface-test",
		Title:       "Interface Test Item",
		Description: "Testing interface implementation",
		Icon:        "🧪",
		Enabled:     true,
	}

	// Test all interface methods
	t.Run("interface methods work", func(t *testing.T) {
		theme := NewTestTheme()
		
		// Test Render method
		rendered := item.Render(80, false, theme)
		assert.NotEmpty(t, rendered, "Render should return non-empty string")
		assert.Contains(t, rendered, "Interface Test Item", "Render should contain item title")
		
		// Test GetHeight method
		height := item.GetHeight()
		assert.Greater(t, height, 0, "GetHeight should return positive value")
		
		// Test IsSelectable method
		selectable := item.IsSelectable()
		assert.True(t, selectable, "Enabled item should be selectable")
		
		// Test GetID method
		id := item.GetID()
		assert.Equal(t, "interface-test", id, "GetID should return correct ID")
	})
}

func TestNavigationItem_RenderStyling(t *testing.T) {
	theme := NewTestTheme()
	item := NavigationItem{
		ID:          "styling-test",
		Title:       "Styling Test",
		Description: "Testing styling behavior",
		Icon:        "🎨",
		Enabled:     true,
	}

	t.Run("selected vs unselected styling", func(t *testing.T) {
		unselected := item.Render(80, false, theme)
		selected := item.Render(80, true, theme)
		
		// Both should contain the same content
		assert.Contains(t, unselected, "Styling Test", "Unselected should contain title")
		assert.Contains(t, selected, "Styling Test", "Selected should contain title")
		
		// Both should render successfully (we can't easily test ANSI codes in unit tests)
		assert.NotEmpty(t, unselected, "Unselected should render successfully")
		assert.NotEmpty(t, selected, "Selected should render successfully")
		
		// The styling logic is tested by the fact that both render without errors
		// and contain the expected content. The actual ANSI styling differences
		// are handled by lipgloss and are difficult to test in unit tests.
	})

	t.Run("enabled vs disabled styling", func(t *testing.T) {
		enabledItem := item
		enabledItem.Enabled = true
		
		disabledItem := item
		disabledItem.Enabled = false
		
		enabledRender := enabledItem.Render(80, false, theme)
		disabledRender := disabledItem.Render(80, false, theme)
		
		// Disabled should have "(disabled)" suffix
		assert.Contains(t, disabledRender, "(disabled)", "Disabled item should show disabled indicator")
		assert.NotContains(t, enabledRender, "(disabled)", "Enabled item should not show disabled indicator")
	})

	t.Run("theme vs no theme styling", func(t *testing.T) {
		withTheme := item.Render(80, false, theme)
		withoutTheme := item.Render(80, false, nil)
		
		// Both should contain the same content
		assert.Contains(t, withTheme, "Styling Test", "With theme should contain title")
		assert.Contains(t, withoutTheme, "Styling Test", "Without theme should contain title")
		
		// Content should be the same, but styling may differ
		// We can't easily test ANSI codes, but we can ensure both render successfully
		assert.NotEmpty(t, withTheme, "With theme should render successfully")
		assert.NotEmpty(t, withoutTheme, "Without theme should render successfully")
	})
}

func TestNavigationItem_EdgeCases(t *testing.T) {
	t.Run("very narrow width", func(t *testing.T) {
		item := NavigationItem{
			ID:          "narrow",
			Title:       "Very Long Title That Exceeds Width",
			Description: "Very long description that definitely exceeds the narrow width",
			Icon:        "📏",
			Enabled:     true,
		}
		
		// Test with very narrow width
		result := item.Render(10, false, NewTestTheme())
		assert.NotEmpty(t, result, "Should render even with narrow width")
	})

	t.Run("zero width", func(t *testing.T) {
		item := NavigationItem{
			ID:      "zero-width",
			Title:   "Zero Width Test",
			Icon:    "0️⃣",
			Enabled: true,
		}
		
		// Test with zero width (edge case)
		result := item.Render(0, false, NewTestTheme())
		assert.NotEmpty(t, result, "Should render even with zero width")
	})

	t.Run("unicode characters in content", func(t *testing.T) {
		item := NavigationItem{
			ID:          "unicode",
			Title:       "Unicode Test 测试 🌍",
			Description: "Description with unicode: 描述 🚀 ñáéíóú",
			Icon:        "🦄",
			Enabled:     true,
		}
		
		result := item.Render(80, false, NewTestTheme())
		assert.Contains(t, result, "Unicode Test 测试 🌍", "Should handle unicode in title")
		assert.Contains(t, result, "描述 🚀 ñáéíóú", "Should handle unicode in description")
	})
}

func TestNavigationItem_ConsistencyWithExistingItems(t *testing.T) {
	// Test that our ScrollableItem implementation produces consistent results
	// with the existing navigation items used in NavigationModel
	
	existingItems := []NavigationItem{
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
			ID:          "settings",
			Title:       "Settings",
			Description: "Configure application preferences",
			Icon:        "⚙️",
			Enabled:     true,
		},
	}

	theme := NewTestTheme()
	
	for _, item := range existingItems {
		t.Run("item_"+item.ID, func(t *testing.T) {
			// Test that all existing items implement the interface correctly
			var scrollableItem ScrollableItem = item
			
			// Test basic functionality
			rendered := scrollableItem.Render(80, false, theme)
			assert.NotEmpty(t, rendered, "Item should render successfully")
			assert.Contains(t, rendered, item.Title, "Rendered output should contain title")
			
			if item.Description != "" {
				assert.Contains(t, rendered, item.Description, "Rendered output should contain description")
			}
			
			// Test height calculation
			height := scrollableItem.GetHeight()
			expectedHeight := 2 // title + spacing
			if item.Description != "" {
				expectedHeight = 3 // title + description + spacing
			}
			assert.Equal(t, expectedHeight, height, "Height should be calculated correctly")
			
			// Test selectability
			assert.Equal(t, item.Enabled, scrollableItem.IsSelectable(), "Selectability should match enabled state")
			
			// Test ID
			assert.Equal(t, item.ID, scrollableItem.GetID(), "ID should match")
		})
	}
}