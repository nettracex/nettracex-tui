// Package tui contains help section adapters for scrollable help content
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// HelpSection represents a section of help content that implements ScrollableItem
type HelpSection struct {
	// Title is the section title (e.g., "Navigation & Scrolling")
	Title string
	
	// Items contains the help entries within this section
	Items []HelpItem
	
	// ID is a unique identifier for this section
	ID string
}

// HelpItem represents an individual help entry within a section
type HelpItem struct {
	// Key is the keyboard shortcut or command (e.g., "↑/↓ or j/k")
	Key string
	
	// Description explains what the key/command does
	Description string
}

// NewHelpSection creates a new help section with the given title and items
func NewHelpSection(title string, items []HelpItem) *HelpSection {
	return &HelpSection{
		Title: title,
		Items: items,
		ID:    strings.ToLower(strings.ReplaceAll(title, " ", "_")),
	}
}

// NewHelpItem creates a new help item with key and description
func NewHelpItem(key, description string) HelpItem {
	return HelpItem{
		Key:         key,
		Description: description,
	}
}

// Render implements ScrollableItem interface
func (hs *HelpSection) Render(width int, selected bool, theme domain.Theme) string {
	var content strings.Builder
	
	// Section title styling
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	if selected {
		// Highlight selected section with background
		titleStyle = titleStyle.
			Background(lipgloss.Color("237")).
			Padding(0, 1)
	}
	
	content.WriteString(titleStyle.Render(hs.Title))
	content.WriteString("\n")
	
	// Render help items
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Width(20).
		Align(lipgloss.Left)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	if selected {
		// Slightly different styling for selected section items
		valueStyle = valueStyle.Foreground(lipgloss.Color("255"))
	}
	
	for _, item := range hs.Items {
		key := keyStyle.Render(item.Key)
		value := valueStyle.Render(item.Description)
		content.WriteString("  " + key + " " + value + "\n")
	}
	
	return content.String()
}

// GetHeight implements ScrollableItem interface
func (hs *HelpSection) GetHeight() int {
	// Title line + items + spacing
	return 1 + len(hs.Items) + 1 // +1 for title, +1 for spacing after section
}

// IsSelectable implements ScrollableItem interface
func (hs *HelpSection) IsSelectable() bool {
	return true // Help sections can be selected for navigation
}

// GetID implements ScrollableItem interface
func (hs *HelpSection) GetID() string {
	return hs.ID
}

// CreateHelpSections creates all help sections for the application
// This function converts the existing help content structure to scrollable sections
func CreateHelpSections() []ScrollableItem {
	sections := make([]ScrollableItem, 0)
	
	// Navigation & Scrolling section
	navigationItems := []HelpItem{
		NewHelpItem("↑/↓ or j/k", "Navigate up/down in menus or scroll content"),
		NewHelpItem("←/→ or h/l", "Navigate left/right (when applicable)"),
		NewHelpItem("PgUp/PgDown", "Scroll page up/down in help and results"),
		NewHelpItem("Home/End", "Jump to top/bottom of scrollable content"),
		NewHelpItem("Enter", "Select menu item or execute action"),
		NewHelpItem("Tab", "Move to next input field in forms"),
		NewHelpItem("Esc", "Go back to previous screen"),
		NewHelpItem("q or Ctrl+C", "Quit application"),
		NewHelpItem("?", "Show/hide this help screen"),
	}
	sections = append(sections, NewHelpSection("Navigation & Scrolling", navigationItems))
	
	// Diagnostic Tools section
	diagnosticItems := []HelpItem{
		NewHelpItem("🔍 WHOIS Lookup", "Query domain registration and IP information"),
		NewHelpItem("📡 Ping Test", "Test connectivity and measure latency"),
		NewHelpItem("🗺️ Traceroute", "Trace network path to destination"),
		NewHelpItem("🌐 DNS Lookup", "Query DNS records (A, AAAA, MX, TXT, etc.)"),
		NewHelpItem("🔒 SSL Certificate", "Check SSL certificate validity and details"),
		NewHelpItem("⚙️ Settings", "Configure application preferences"),
	}
	sections = append(sections, NewHelpSection("Diagnostic Tools", diagnosticItems))
	
	// Form Controls section
	formItems := []HelpItem{
		NewHelpItem("Tab/Shift+Tab", "Navigate between form fields"),
		NewHelpItem("Enter", "Submit form or execute diagnostic"),
		NewHelpItem("Esc", "Cancel and return to main menu"),
		NewHelpItem("Type normally", "Enter text in input fields"),
	}
	sections = append(sections, NewHelpSection("Form Controls", formItems))
	
	// Result Views section
	resultItems := []HelpItem{
		NewHelpItem("f", "Switch to formatted view"),
		NewHelpItem("t", "Switch to table view"),
		NewHelpItem("r", "Switch to raw data view"),
		NewHelpItem("Tab", "Cycle through view modes"),
		NewHelpItem("↑/↓", "Navigate/scroll in table and result views"),
		NewHelpItem("PgUp/PgDown", "Page up/down in long results"),
		NewHelpItem("Esc", "Return to tool input"),
	}
	sections = append(sections, NewHelpSection("Result Views", resultItems))
	
	// Tips & Examples section
	tipsItems := []HelpItem{
		NewHelpItem("Domain examples", "google.com, github.io, example.dev, lavan.dev"),
		NewHelpItem("IP examples", "8.8.8.8, 1.1.1.1, 192.168.1.1"),
		NewHelpItem("Ping counts", "Use 1-100 for ping count (default: 4)"),
		NewHelpItem("DNS records", "A, AAAA, MX, TXT, CNAME, NS supported"),
		NewHelpItem("SSL ports", "443 (HTTPS), 993 (IMAPS), 995 (POP3S)"),
		NewHelpItem("WHOIS queries", "Works with domains and IP addresses"),
		NewHelpItem("Traceroute", "Shows network path with hop details"),
	}
	sections = append(sections, NewHelpSection("Tips & Examples", tipsItems))
	
	// Troubleshooting section
	troubleshootingItems := []HelpItem{
		NewHelpItem("No results", "Check network connection and query format"),
		NewHelpItem("Timeout errors", "Try again or check if host is reachable"),
		NewHelpItem("WHOIS no data", "Some domains may have privacy protection"),
		NewHelpItem("DNS failures", "Verify domain exists and DNS servers work"),
		NewHelpItem("SSL errors", "Check if port supports SSL/TLS"),
		NewHelpItem("Long results", "Use ↑/↓ or PgUp/PgDown to scroll"),
	}
	sections = append(sections, NewHelpSection("Troubleshooting", troubleshootingItems))
	
	return sections
}

// GetHelpSectionByID finds a help section by its ID
func GetHelpSectionByID(sections []ScrollableItem, id string) *HelpSection {
	for _, section := range sections {
		if helpSection, ok := section.(*HelpSection); ok && helpSection.GetID() == id {
			return helpSection
		}
	}
	return nil
}

// CalculateTotalHelpHeight calculates the total height needed for all help sections
func CalculateTotalHelpHeight(sections []ScrollableItem) int {
	totalHeight := 0
	for _, section := range sections {
		totalHeight += section.GetHeight()
	}
	return totalHeight
}