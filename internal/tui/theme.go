// Package tui contains theme system for the TUI
package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// DefaultTheme implements the domain.Theme interface
type DefaultTheme struct {
	colors map[string]string
	styles map[string]map[string]interface{}
}

// NewDefaultTheme creates a new default theme
func NewDefaultTheme() *DefaultTheme {
	colors := map[string]string{
		"primary":     "62",   // Blue
		"secondary":   "205",  // Pink
		"success":     "46",   // Green
		"warning":     "226",  // Yellow
		"error":       "196",  // Red
		"info":        "39",   // Light Blue
		"background":  "235",  // Dark Gray
		"foreground":  "252",  // Light Gray
		"muted":       "243",  // Medium Gray
		"border":      "240",  // Border Gray
		"highlight":   "230",  // White
	}

	styles := map[string]map[string]interface{}{
		"header": {
			"background": colors["primary"],
			"foreground": colors["highlight"],
			"bold":       true,
			"padding":    "0 1",
		},
		"footer": {
			"background": colors["border"],
			"foreground": colors["foreground"],
			"padding":    "0 1",
		},
		"menu_item": {
			"padding": "0 2",
		},
		"menu_item_selected": {
			"background": colors["primary"],
			"foreground": colors["highlight"],
			"bold":       true,
		},
		"menu_item_disabled": {
			"foreground": colors["border"],
		},
		"form_label": {
			"bold": true,
		},
		"form_input": {
			"border":           "rounded",
			"border_foreground": colors["border"],
			"padding":          "0 1",
		},
		"form_input_focused": {
			"border":           "rounded",
			"border_foreground": colors["primary"],
			"padding":          "0 1",
		},
		"table_header": {
			"background": colors["primary"],
			"foreground": colors["highlight"],
			"bold":       true,
			"padding":    "0 1",
		},
		"table_row": {
			"padding": "0 1",
		},
		"table_row_selected": {
			"background": colors["primary"],
			"foreground": colors["highlight"],
		},
		"progress_bar": {
			"foreground": colors["primary"],
			"background": colors["border"],
		},
		"error": {
			"foreground": colors["error"],
			"italic":     true,
		},
		"success": {
			"foreground": colors["success"],
		},
		"warning": {
			"foreground": colors["warning"],
		},
		"info": {
			"foreground": colors["info"],
		},
		"muted": {
			"foreground": colors["muted"],
			"italic":     true,
		},
	}

	return &DefaultTheme{
		colors: colors,
		styles: styles,
	}
}

// GetColor implements domain.Theme
func (t *DefaultTheme) GetColor(element string) string {
	if color, exists := t.colors[element]; exists {
		return color
	}
	return t.colors["foreground"] // Default color
}

// GetStyle implements domain.Theme
func (t *DefaultTheme) GetStyle(element string) map[string]interface{} {
	if style, exists := t.styles[element]; exists {
		return style
	}
	return make(map[string]interface{}) // Empty style
}

// SetColor implements domain.Theme
func (t *DefaultTheme) SetColor(element, color string) {
	t.colors[element] = color
}

// GetLipglossStyle returns a lipgloss.Style for the given element
func (t *DefaultTheme) GetLipglossStyle(element string) lipgloss.Style {
	style := lipgloss.NewStyle()
	styleMap := t.GetStyle(element)

	// Apply style properties
	if bg, ok := styleMap["background"].(string); ok {
		style = style.Background(lipgloss.Color(bg))
	}
	if fg, ok := styleMap["foreground"].(string); ok {
		style = style.Foreground(lipgloss.Color(fg))
	}
	if bold, ok := styleMap["bold"].(bool); ok && bold {
		style = style.Bold(true)
	}
	if italic, ok := styleMap["italic"].(bool); ok && italic {
		style = style.Italic(true)
	}
	if _, ok := styleMap["padding"].(string); ok {
		// Parse padding string (e.g., "0 1" -> top/bottom=0, left/right=1)
		style = style.Padding(0, 1) // Simplified for now
	}
	if border, ok := styleMap["border"].(string); ok {
		switch border {
		case "rounded":
			style = style.Border(lipgloss.RoundedBorder())
		case "normal":
			style = style.Border(lipgloss.NormalBorder())
		case "thick":
			style = style.Border(lipgloss.ThickBorder())
		}
	}
	if borderFg, ok := styleMap["border_foreground"].(string); ok {
		style = style.BorderForeground(lipgloss.Color(borderFg))
	}

	return style
}

// DarkTheme creates a dark theme variant
type DarkTheme struct {
	*DefaultTheme
}

// NewDarkTheme creates a new dark theme
func NewDarkTheme() *DarkTheme {
	base := NewDefaultTheme()
	
	// Override colors for dark theme
	base.colors["background"] = "0"    // Black
	base.colors["foreground"] = "15"   // White
	base.colors["muted"] = "8"         // Dark Gray
	base.colors["border"] = "8"        // Dark Gray
	
	return &DarkTheme{DefaultTheme: base}
}

// LightTheme creates a light theme variant
type LightTheme struct {
	*DefaultTheme
}

// NewLightTheme creates a new light theme
func NewLightTheme() *LightTheme {
	base := NewDefaultTheme()
	
	// Override colors for light theme
	base.colors["background"] = "15"   // White
	base.colors["foreground"] = "0"    // Black
	base.colors["muted"] = "8"         // Gray
	base.colors["border"] = "7"        // Light Gray
	base.colors["primary"] = "4"       // Blue
	
	return &LightTheme{DefaultTheme: base}
}

// ThemeManager manages theme switching and application
type ThemeManager struct {
	themes      map[string]domain.Theme
	currentName string
	current     domain.Theme
}

// NewThemeManager creates a new theme manager
func NewThemeManager() *ThemeManager {
	themes := map[string]domain.Theme{
		"default": NewDefaultTheme(),
		"dark":    NewDarkTheme(),
		"light":   NewLightTheme(),
	}

	return &ThemeManager{
		themes:      themes,
		currentName: "default",
		current:     themes["default"],
	}
}

// GetTheme returns the current theme
func (tm *ThemeManager) GetTheme() domain.Theme {
	return tm.current
}

// SetTheme sets the current theme by name
func (tm *ThemeManager) SetTheme(name string) bool {
	if theme, exists := tm.themes[name]; exists {
		tm.currentName = name
		tm.current = theme
		return true
	}
	return false
}

// GetCurrentThemeName returns the name of the current theme
func (tm *ThemeManager) GetCurrentThemeName() string {
	return tm.currentName
}

// GetAvailableThemes returns a list of available theme names
func (tm *ThemeManager) GetAvailableThemes() []string {
	var names []string
	for name := range tm.themes {
		names = append(names, name)
	}
	return names
}

// RegisterTheme registers a new theme
func (tm *ThemeManager) RegisterTheme(name string, theme domain.Theme) {
	tm.themes[name] = theme
}

// ApplyThemeToComponent applies the current theme to a TUI component
func (tm *ThemeManager) ApplyThemeToComponent(component domain.TUIComponent) {
	component.SetTheme(tm.current)
}

// ResponsiveLayout handles responsive layout calculations
type ResponsiveLayout struct {
	width  int
	height int
}

// NewResponsiveLayout creates a new responsive layout manager
func NewResponsiveLayout() *ResponsiveLayout {
	return &ResponsiveLayout{}
}

// SetSize updates the layout dimensions
func (rl *ResponsiveLayout) SetSize(width, height int) {
	rl.width = width
	rl.height = height
}

// GetContentArea returns the available content area dimensions
func (rl *ResponsiveLayout) GetContentArea(headerHeight, footerHeight int) (int, int) {
	contentWidth := rl.width
	contentHeight := rl.height - headerHeight - footerHeight
	
	if contentHeight < 1 {
		contentHeight = 1
	}
	if contentWidth < 1 {
		contentWidth = 1
	}
	
	return contentWidth, contentHeight
}

// IsSmallScreen returns true if the screen is considered small
func (rl *ResponsiveLayout) IsSmallScreen() bool {
	return rl.width < 80 || rl.height < 24
}

// IsMediumScreen returns true if the screen is considered medium
func (rl *ResponsiveLayout) IsMediumScreen() bool {
	return rl.width >= 80 && rl.width < 120 && rl.height >= 24
}

// IsLargeScreen returns true if the screen is considered large
func (rl *ResponsiveLayout) IsLargeScreen() bool {
	return rl.width >= 120 && rl.height >= 30
}

// GetColumnCount returns the recommended number of columns for the current screen size
func (rl *ResponsiveLayout) GetColumnCount() int {
	if rl.IsSmallScreen() {
		return 1
	} else if rl.IsMediumScreen() {
		return 2
	}
	return 3
}

// GetMaxTableWidth returns the maximum recommended table width
func (rl *ResponsiveLayout) GetMaxTableWidth() int {
	return rl.width - 4 // Account for padding and borders
}

// GetFormWidth returns the recommended form width
func (rl *ResponsiveLayout) GetFormWidth() int {
	maxWidth := 60
	if rl.width < maxWidth {
		return rl.width - 4
	}
	return maxWidth
}