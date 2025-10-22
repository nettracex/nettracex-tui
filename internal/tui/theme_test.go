// Package tui contains tests for theme system
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

func TestNewDefaultTheme(t *testing.T) {
	theme := NewDefaultTheme()

	assert.NotNil(t, theme)
	assert.NotEmpty(t, theme.colors)
	assert.NotEmpty(t, theme.styles)

	// Test that essential colors are defined
	essentialColors := []string{
		"primary", "secondary", "success", "warning", "error",
		"info", "background", "foreground", "muted", "border", "highlight",
	}

	for _, color := range essentialColors {
		assert.NotEmpty(t, theme.GetColor(color), "Color %s should be defined", color)
	}
}

func TestDefaultTheme_GetColor(t *testing.T) {
	theme := NewDefaultTheme()

	// Test existing color
	primary := theme.GetColor("primary")
	assert.Equal(t, "62", primary)

	// Test non-existing color (should return foreground as default)
	nonExistent := theme.GetColor("nonexistent")
	assert.Equal(t, theme.GetColor("foreground"), nonExistent)
}

func TestDefaultTheme_SetColor(t *testing.T) {
	theme := NewDefaultTheme()

	// Set a new color
	theme.SetColor("custom", "123")
	assert.Equal(t, "123", theme.GetColor("custom"))

	// Override existing color
	theme.SetColor("primary", "456")
	assert.Equal(t, "456", theme.GetColor("primary"))
}

func TestDefaultTheme_GetStyle(t *testing.T) {
	theme := NewDefaultTheme()

	// Test existing style
	headerStyle := theme.GetStyle("header")
	assert.NotEmpty(t, headerStyle)
	assert.Equal(t, "62", headerStyle["background"])
	assert.Equal(t, "230", headerStyle["foreground"])
	assert.Equal(t, true, headerStyle["bold"])

	// Test non-existing style
	nonExistentStyle := theme.GetStyle("nonexistent")
	assert.Empty(t, nonExistentStyle)
}

func TestDefaultTheme_GetLipglossStyle(t *testing.T) {
	theme := NewDefaultTheme()

	// Test header style
	headerStyle := theme.GetLipglossStyle("header")
	assert.NotNil(t, headerStyle)

	// Test that the style has the expected properties
	// Note: We can't directly test lipgloss.Style properties,
	// but we can test that it doesn't panic and returns a valid style
	rendered := headerStyle.Render("Test")
	assert.NotEmpty(t, rendered)

	// Test non-existent style
	emptyStyle := theme.GetLipglossStyle("nonexistent")
	assert.NotNil(t, emptyStyle)
}

func TestNewDarkTheme(t *testing.T) {
	theme := NewDarkTheme()

	assert.NotNil(t, theme)
	assert.Equal(t, "0", theme.GetColor("background"))   // Black
	assert.Equal(t, "15", theme.GetColor("foreground"))  // White
	assert.Equal(t, "8", theme.GetColor("muted"))        // Dark Gray
	assert.Equal(t, "8", theme.GetColor("border"))       // Dark Gray
}

func TestNewLightTheme(t *testing.T) {
	theme := NewLightTheme()

	assert.NotNil(t, theme)
	assert.Equal(t, "15", theme.GetColor("background"))  // White
	assert.Equal(t, "0", theme.GetColor("foreground"))   // Black
	assert.Equal(t, "8", theme.GetColor("muted"))        // Gray
	assert.Equal(t, "7", theme.GetColor("border"))       // Light Gray
	assert.Equal(t, "4", theme.GetColor("primary"))      // Blue
}

func TestNewThemeManager(t *testing.T) {
	manager := NewThemeManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.themes)
	assert.Equal(t, "default", manager.currentName)
	assert.NotNil(t, manager.current)

	// Test that default themes are registered
	availableThemes := manager.GetAvailableThemes()
	expectedThemes := []string{"default", "dark", "light"}
	
	for _, expected := range expectedThemes {
		assert.Contains(t, availableThemes, expected)
	}
}

func TestThemeManager_SetTheme(t *testing.T) {
	manager := NewThemeManager()

	// Test setting existing theme
	success := manager.SetTheme("dark")
	assert.True(t, success)
	assert.Equal(t, "dark", manager.GetCurrentThemeName())

	// Test setting non-existent theme
	success = manager.SetTheme("nonexistent")
	assert.False(t, success)
	assert.Equal(t, "dark", manager.GetCurrentThemeName()) // Should remain unchanged
}

func TestThemeManager_GetTheme(t *testing.T) {
	manager := NewThemeManager()

	theme := manager.GetTheme()
	assert.NotNil(t, theme)
	assert.Equal(t, manager.current, theme)
}

func TestThemeManager_RegisterTheme(t *testing.T) {
	manager := NewThemeManager()
	customTheme := NewDefaultTheme()

	// Register custom theme
	manager.RegisterTheme("custom", customTheme)

	// Test that it's available
	availableThemes := manager.GetAvailableThemes()
	assert.Contains(t, availableThemes, "custom")

	// Test that we can set it
	success := manager.SetTheme("custom")
	assert.True(t, success)
	assert.Equal(t, "custom", manager.GetCurrentThemeName())
}

func TestThemeManager_ApplyThemeToComponent(t *testing.T) {
	manager := NewThemeManager()
	
	// Create a mock component
	component := &MockTUIComponent{}
	component.On("SetTheme", manager.current).Return()

	// Apply theme
	manager.ApplyThemeToComponent(component)

	// Verify the theme was applied
	component.AssertExpectations(t)
}

// MockTUIComponent for testing
type MockTUIComponent struct {
	MockTheme
}

func (m *MockTUIComponent) Init() tea.Cmd {
	return nil
}

func (m *MockTUIComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *MockTUIComponent) View() string {
	return "mock view"
}

func (m *MockTUIComponent) SetSize(width, height int) {
	m.Called(width, height)
}

func (m *MockTUIComponent) SetTheme(theme domain.Theme) {
	m.Called(theme)
}

func (m *MockTUIComponent) Focus() {
	m.Called()
}

func (m *MockTUIComponent) Blur() {
	m.Called()
}

func TestNewResponsiveLayout(t *testing.T) {
	layout := NewResponsiveLayout()

	assert.NotNil(t, layout)
	assert.Equal(t, 0, layout.width)
	assert.Equal(t, 0, layout.height)
}

func TestResponsiveLayout_SetSize(t *testing.T) {
	layout := NewResponsiveLayout()

	layout.SetSize(100, 50)
	assert.Equal(t, 100, layout.width)
	assert.Equal(t, 50, layout.height)
}

func TestResponsiveLayout_GetContentArea(t *testing.T) {
	layout := NewResponsiveLayout()
	layout.SetSize(100, 50)

	width, height := layout.GetContentArea(3, 2)
	assert.Equal(t, 100, width)
	assert.Equal(t, 45, height) // 50 - 3 - 2

	// Test minimum height
	width, height = layout.GetContentArea(30, 30)
	assert.Equal(t, 100, width)
	assert.Equal(t, 1, height) // Should be at least 1
}

func TestResponsiveLayout_ScreenSizeDetection(t *testing.T) {
	layout := NewResponsiveLayout()

	// Test small screen
	layout.SetSize(60, 20)
	assert.True(t, layout.IsSmallScreen())
	assert.False(t, layout.IsMediumScreen())
	assert.False(t, layout.IsLargeScreen())

	// Test medium screen
	layout.SetSize(100, 25)
	assert.False(t, layout.IsSmallScreen())
	assert.True(t, layout.IsMediumScreen())
	assert.False(t, layout.IsLargeScreen())

	// Test large screen
	layout.SetSize(130, 35)
	assert.False(t, layout.IsSmallScreen())
	assert.False(t, layout.IsMediumScreen())
	assert.True(t, layout.IsLargeScreen())
}

func TestResponsiveLayout_GetColumnCount(t *testing.T) {
	layout := NewResponsiveLayout()

	// Small screen
	layout.SetSize(60, 20)
	assert.Equal(t, 1, layout.GetColumnCount())

	// Medium screen
	layout.SetSize(100, 25)
	assert.Equal(t, 2, layout.GetColumnCount())

	// Large screen
	layout.SetSize(130, 35)
	assert.Equal(t, 3, layout.GetColumnCount())
}

func TestResponsiveLayout_GetMaxTableWidth(t *testing.T) {
	layout := NewResponsiveLayout()
	layout.SetSize(100, 50)

	maxWidth := layout.GetMaxTableWidth()
	assert.Equal(t, 96, maxWidth) // 100 - 4
}

func TestResponsiveLayout_GetFormWidth(t *testing.T) {
	layout := NewResponsiveLayout()

	// Test with wide screen
	layout.SetSize(100, 50)
	formWidth := layout.GetFormWidth()
	assert.Equal(t, 60, formWidth) // Max form width

	// Test with narrow screen
	layout.SetSize(50, 30)
	formWidth = layout.GetFormWidth()
	assert.Equal(t, 46, formWidth) // 50 - 4
}