// Package dns provides TUI model for DNS diagnostic tool
package dns

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Model represents the DNS tool TUI model
type Model struct {
	tool           *Tool
	state          ModelState
	input          textinput.Model
	result         domain.DNSResult
	error          error
	width          int
	height         int
	theme          domain.Theme
	loading        bool
	selectedTypes  map[domain.DNSRecordType]bool
	typeSelection  int
	showTypeSelect bool
	resultTab      int
	resultTabs     []ResultTab
	scrollOffset   int
	maxScroll      int
}

// ModelState represents the current state of the model
type ModelState int

const (
	StateInput ModelState = iota
	StateTypeSelection
	StateLoading
	StateResult
	StateError
)

// ResultTab represents a tab in the result view
type ResultTab struct {
	Name    string
	Records []domain.DNSRecord
	Active  bool
}

// NewModel creates a new DNS model
func NewModel(tool *Tool) *Model {
	input := textinput.New()
	input.Placeholder = "Enter domain name (e.g., example.com, google.com)"
	input.Focus()
	input.CharLimit = 253
	input.Width = 50

	// Default to all record types selected
	selectedTypes := map[domain.DNSRecordType]bool{
		domain.DNSRecordTypeA:     true,
		domain.DNSRecordTypeAAAA:  true,
		domain.DNSRecordTypeMX:    true,
		domain.DNSRecordTypeTXT:   true,
		domain.DNSRecordTypeCNAME: true,
		domain.DNSRecordTypeNS:    true,
	}

	return &Model{
		tool:           tool,
		state:          StateInput,
		input:          input,
		loading:        false,
		selectedTypes:  selectedTypes,
		typeSelection:  0,
		showTypeSelect: false,
		resultTab:      0,
		resultTabs:     []ResultTab{},
		scrollOffset:   0,
		maxScroll:      0,
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == StateTypeSelection {
				m.state = StateInput
				m.showTypeSelect = false
				m.input.Focus()
				return m, nil
			} else if m.state != StateInput {
				m.state = StateInput
				m.input.SetValue("")
				m.input.Focus()
				m.error = nil
				m.showTypeSelect = false
				m.resultTabs = []ResultTab{}
				m.resultTab = 0
				m.scrollOffset = 0
				m.maxScroll = 0
				return m, nil
			}
		case "tab":
			if m.state == StateInput {
				m.state = StateTypeSelection
				m.showTypeSelect = true
				m.input.Blur()
				return m, nil
			}
		case "enter":
			if m.state == StateInput && m.input.Value() != "" {
				return m, m.performLookup()
			} else if m.state == StateTypeSelection {
				m.state = StateInput
				m.showTypeSelect = false
				m.input.Focus()
				return m, nil
			}
		case "up":
			if m.state == StateTypeSelection && m.typeSelection > 0 {
				m.typeSelection--
			} else if m.state == StateResult && m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down":
			if m.state == StateTypeSelection && m.typeSelection < 5 {
				m.typeSelection++
			} else if m.state == StateResult && m.scrollOffset < m.maxScroll {
				m.scrollOffset++
			}
		case "left":
			if m.state == StateResult && len(m.resultTabs) > 0 && m.resultTab > 0 {
				m.resultTab--
				m.scrollOffset = 0 // Reset scroll when changing tabs
			}
		case "right":
			if m.state == StateResult && len(m.resultTabs) > 0 && m.resultTab < len(m.resultTabs)-1 {
				m.resultTab++
				m.scrollOffset = 0 // Reset scroll when changing tabs
			}
		case " ":
			if m.state == StateTypeSelection {
				recordType := m.getRecordTypeByIndex(m.typeSelection)
				m.selectedTypes[recordType] = !m.selectedTypes[recordType]
			}
		}

	case lookupStartMsg:
		m.state = StateLoading
		m.loading = true
		return m, nil

	case lookupResultMsg:
		m.state = StateResult
		m.loading = false
		m.result = msg.result
		m.buildResultTabs()
		m.calculateMaxScroll()
		return m, nil

	case lookupErrorMsg:
		m.state = StateError
		m.loading = false
		m.error = msg.error
		return m, nil
	}

	// Update input field
	if m.state == StateInput {
		m.input, cmd = m.input.Update(msg)
	}

	return m, cmd
}

// View renders the model
func (m *Model) View() string {
	var content strings.Builder

	// Header
	content.WriteString(m.renderHeader())
	content.WriteString("\n\n")

	switch m.state {
	case StateInput:
		content.WriteString(m.renderInput())
		if m.showTypeSelect {
			content.WriteString("\n\n")
			content.WriteString(m.renderTypeSelection())
		}
	case StateTypeSelection:
		content.WriteString(m.renderInput())
		content.WriteString("\n\n")
		content.WriteString(m.renderTypeSelection())
	case StateLoading:
		content.WriteString(m.renderLoading())
	case StateResult:
		content.WriteString(m.renderResult())
	case StateError:
		content.WriteString(m.renderError())
	}

	// Footer
	content.WriteString("\n\n")
	content.WriteString(m.renderFooter())

	return content.String()
}

// SetSize sets the model dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.Width = width - 4
}

// SetTheme sets the model theme
func (m *Model) SetTheme(theme domain.Theme) {
	m.theme = theme
}

// Focus focuses the model
func (m *Model) Focus() {
	if m.state == StateInput {
		m.input.Focus()
	}
}

// Blur blurs the model
func (m *Model) Blur() {
	m.input.Blur()
}

// renderHeader renders the tool header
func (m *Model) renderHeader() string {
	title := "DNS Lookup Tool"
	description := "Query DNS records for domains with support for multiple record types"
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	return titleStyle.Render(title) + "\n" + descStyle.Render(description)
}

// renderInput renders the input form
func (m *Model) renderInput() string {
	var content strings.Builder
	
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))
	
	content.WriteString(labelStyle.Render("Domain:"))
	content.WriteString("\n")
	content.WriteString(m.input.View())
	content.WriteString("\n\n")
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	
	content.WriteString(helpStyle.Render("Enter a domain name (e.g., example.com, google.com)"))
	
	return content.String()
}

// renderTypeSelection renders the record type selection interface
func (m *Model) renderTypeSelection() string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))
	
	content.WriteString(titleStyle.Render("Record Types:"))
	content.WriteString("\n")
	
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}
	
	for i, recordType := range recordTypes {
		var line strings.Builder
		
		// Selection indicator
		if i == m.typeSelection {
			line.WriteString("▶ ")
		} else {
			line.WriteString("  ")
		}
		
		// Checkbox
		if m.selectedTypes[recordType] {
			line.WriteString("☑ ")
		} else {
			line.WriteString("☐ ")
		}
		
		// Record type name and description
		line.WriteString(GetRecordTypeString(recordType))
		line.WriteString(" - ")
		line.WriteString(m.getRecordTypeDescription(recordType))
		
		// Style based on selection
		if i == m.typeSelection {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Bold(true)
			content.WriteString(selectedStyle.Render(line.String()))
		} else {
			content.WriteString(line.String())
		}
		content.WriteString("\n")
	}
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Use ↑/↓ to navigate, space to toggle, enter to confirm"))
	
	return content.String()
}

// renderLoading renders the loading state
func (m *Model) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	
	selectedCount := 0
	for _, selected := range m.selectedTypes {
		if selected {
			selectedCount++
		}
	}
	
	return loadingStyle.Render(fmt.Sprintf("🔍 Performing DNS lookups for '%s' (%d record types)...", m.input.Value(), selectedCount))
}

// renderResult renders the DNS result with tabbed interface
func (m *Model) renderResult() string {
	if m.result.Query == "" {
		return "No result available"
	}
	
	var content strings.Builder
	
	// Query info section
	content.WriteString(m.renderSection("Query Information", [][]string{
		{"Domain", m.result.Query},
		{"Server", m.result.Server},
		{"Response Time", m.result.ResponseTime.String()},
		{"Total Records", fmt.Sprintf("%d", len(m.result.Records))},
	}))
	
	// Render tabs if we have multiple record types
	if len(m.resultTabs) > 1 {
		content.WriteString("\n\n")
		content.WriteString(m.renderTabs())
		content.WriteString("\n")
		content.WriteString(m.renderActiveTabContent())
	} else if len(m.resultTabs) == 1 {
		// Single tab, render directly
		content.WriteString("\n\n")
		content.WriteString(m.renderTabContent(m.resultTabs[0]))
	} else {
		// No records, show message
		content.WriteString("\n\n")
		noRecordsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
		content.WriteString(noRecordsStyle.Render("No DNS records found"))
	}
	
	// Add authority and additional sections if present
	if len(m.result.Authority) > 0 {
		content.WriteString("\n\n")
		content.WriteString(m.renderAuthoritySection())
	}
	
	if len(m.result.Additional) > 0 {
		content.WriteString("\n\n")
		content.WriteString(m.renderAdditionalSection())
	}
	
	return content.String()
}

// renderSection renders a section with key-value pairs
func (m *Model) renderSection(title string, data [][]string) string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Width(15).
		Align(lipgloss.Right)
	
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")
	
	for _, row := range data {
		if len(row) >= 2 && row[1] != "" {
			content.WriteString(keyStyle.Render(row[0]+":"))
			content.WriteString(" ")
			content.WriteString(valueStyle.Render(row[1]))
			content.WriteString("\n")
		}
	}
	
	return content.String()
}



// renderAuthoritySection renders the authority records section
func (m *Model) renderAuthoritySection() string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	content.WriteString(titleStyle.Render("Authority Records"))
	content.WriteString("\n")
	
	recordStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	for _, record := range m.result.Authority {
		content.WriteString(recordStyle.Render(fmt.Sprintf("  %s %d %s", 
			record.Name, record.TTL, record.Value)))
		content.WriteString("\n")
	}
	
	return content.String()
}

// renderAdditionalSection renders the additional records section
func (m *Model) renderAdditionalSection() string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	content.WriteString(titleStyle.Render("Additional Records"))
	content.WriteString("\n")
	
	recordStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	
	for _, record := range m.result.Additional {
		content.WriteString(recordStyle.Render(fmt.Sprintf("  %s %d %s", 
			record.Name, record.TTL, record.Value)))
		content.WriteString("\n")
	}
	
	return content.String()
}

// renderError renders the error state
func (m *Model) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	return errorStyle.Render(fmt.Sprintf("❌ Error: %s", m.error.Error()))
}

// renderFooter renders the footer with help text
func (m *Model) renderFooter() string {
	var help []string
	
	switch m.state {
	case StateInput:
		help = []string{"enter: lookup", "tab: select record types", "q: quit"}
	case StateTypeSelection:
		help = []string{"↑/↓: navigate", "space: toggle", "enter: confirm", "esc: back"}
	case StateResult:
		if len(m.resultTabs) > 1 {
			help = []string{"←/→: switch tabs", "↑/↓: scroll", "esc: new lookup", "q: quit"}
		} else {
			help = []string{"↑/↓: scroll", "esc: new lookup", "q: quit"}
		}
	case StateError:
		help = []string{"esc: new lookup", "q: quit"}
	case StateLoading:
		help = []string{"q: quit"}
	}
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	return helpStyle.Render(strings.Join(help, " • "))
}

// performLookup performs the DNS lookup
func (m *Model) performLookup() tea.Cmd {
	domainName := strings.TrimSpace(m.input.Value())
	
	// Get selected record types
	var selectedTypes []domain.DNSRecordType
	for recordType, selected := range m.selectedTypes {
		if selected {
			selectedTypes = append(selectedTypes, recordType)
		}
	}
	
	return tea.Batch(
		func() tea.Msg { return lookupStartMsg{} },
		func() tea.Msg {
			// Create parameters
			params := domain.NewDNSParameters(domainName, domain.DNSRecordTypeA) // Default type, will be overridden
			params.Set("record_types", selectedTypes)
			
			// Execute lookup
			result, err := m.tool.Execute(context.Background(), params)
			if err != nil {
				return lookupErrorMsg{error: err}
			}
			
			// Extract DNS result
			dnsResult, ok := result.Data().(domain.DNSResult)
			if !ok {
				return lookupErrorMsg{error: fmt.Errorf("invalid result type")}
			}
			
			return lookupResultMsg{result: dnsResult}
		},
	)
}

// getRecordTypeByIndex returns the record type at the given index
func (m *Model) getRecordTypeByIndex(index int) domain.DNSRecordType {
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}
	
	if index >= 0 && index < len(recordTypes) {
		return recordTypes[index]
	}
	return domain.DNSRecordTypeA
}

// getRecordTypeDescription returns a description for a DNS record type
func (m *Model) getRecordTypeDescription(recordType domain.DNSRecordType) string {
	switch recordType {
	case domain.DNSRecordTypeA:
		return "IPv4 address records"
	case domain.DNSRecordTypeAAAA:
		return "IPv6 address records"
	case domain.DNSRecordTypeMX:
		return "Mail exchange records"
	case domain.DNSRecordTypeTXT:
		return "Text records"
	case domain.DNSRecordTypeCNAME:
		return "Canonical name records"
	case domain.DNSRecordTypeNS:
		return "Name server records"
	default:
		return "Unknown record type"
	}
}

// buildResultTabs builds tabs from DNS result records
func (m *Model) buildResultTabs() {
	m.resultTabs = []ResultTab{}
	
	// Group records by type
	recordsByType := make(map[domain.DNSRecordType][]domain.DNSRecord)
	for _, record := range m.result.Records {
		recordsByType[record.Type] = append(recordsByType[record.Type], record)
	}
	
	// Create tabs for each record type that has records
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}
	
	for _, recordType := range recordTypes {
		if records, exists := recordsByType[recordType]; exists && len(records) > 0 {
			tab := ResultTab{
				Name:    GetRecordTypeString(recordType),
				Records: records,
				Active:  len(m.resultTabs) == 0, // First tab is active
			}
			m.resultTabs = append(m.resultTabs, tab)
		}
	}
	
	// Reset tab selection
	m.resultTab = 0
}

// calculateMaxScroll calculates the maximum scroll offset
func (m *Model) calculateMaxScroll() {
	if len(m.resultTabs) == 0 {
		m.maxScroll = 0
		return
	}
	
	// Calculate content height for current tab
	activeTab := m.resultTabs[m.resultTab]
	contentLines := len(activeTab.Records) + 5 // Add some padding for headers
	
	// Available height for content (subtract header, tabs, footer)
	availableHeight := m.height - 10 // Conservative estimate
	if availableHeight < 5 {
		availableHeight = 5
	}
	
	m.maxScroll = contentLines - availableHeight
	if m.maxScroll < 0 {
		m.maxScroll = 0
	}
}

// renderTabs renders the tab navigation
func (m *Model) renderTabs() string {
	if len(m.resultTabs) <= 1 {
		return ""
	}
	
	var tabs []string
	
	for i, tab := range m.resultTabs {
		tabStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Border(lipgloss.RoundedBorder(), true, true, false, true)
		
		if i == m.resultTab {
			// Active tab
			tabStyle = tabStyle.
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("62")).
				Bold(true)
		} else {
			// Inactive tab
			tabStyle = tabStyle.
				Foreground(lipgloss.Color("243")).
				Background(lipgloss.Color("236"))
		}
		
		tabText := fmt.Sprintf("%s (%d)", tab.Name, len(tab.Records))
		tabs = append(tabs, tabStyle.Render(tabText))
	}
	
	return lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
}

// renderActiveTabContent renders the content of the active tab
func (m *Model) renderActiveTabContent() string {
	if len(m.resultTabs) == 0 || m.resultTab >= len(m.resultTabs) {
		return "No tab content available"
	}
	
	return m.renderTabContent(m.resultTabs[m.resultTab])
}

// renderTabContent renders the content of a specific tab
func (m *Model) renderTabContent(tab ResultTab) string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	content.WriteString(titleStyle.Render(fmt.Sprintf("%s Records (%d)", tab.Name, len(tab.Records))))
	content.WriteString("\n")
	
	if len(tab.Records) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
		content.WriteString(emptyStyle.Render("No records found"))
		return content.String()
	}
	
	// Apply scrolling offset
	startIdx := m.scrollOffset
	availableLines := 10 // Conservative estimate for available lines
	if m.height > 20 {
		availableLines = m.height - 15
	}
	endIdx := startIdx + availableLines
	if endIdx > len(tab.Records) {
		endIdx = len(tab.Records)
	}
	if startIdx >= len(tab.Records) {
		startIdx = len(tab.Records) - 1
	}
	if startIdx < 0 {
		startIdx = 0
	}
	
	recordStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 2)
	
	for i := startIdx; i < endIdx; i++ {
		record := tab.Records[i]
		var recordLine string
		
		if record.Priority > 0 {
			recordLine = fmt.Sprintf("%-30s %6d  %-50s (Priority: %d)", 
				record.Name, record.TTL, record.Value, record.Priority)
		} else {
			recordLine = fmt.Sprintf("%-30s %6d  %s", 
				record.Name, record.TTL, record.Value)
		}
		
		content.WriteString(recordStyle.Render(recordLine))
		content.WriteString("\n")
	}
	
	// Show scroll indicator if needed
	scrollStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)
	
	scrollInfo := fmt.Sprintf("Showing %d-%d of %d records", 
		startIdx+1, endIdx, len(tab.Records))
	if m.maxScroll > 0 && (m.scrollOffset > 0 || endIdx < len(tab.Records)) {
		scrollInfo += " (↑/↓ to scroll)"
	}
	
	content.WriteString("\n")
	content.WriteString(scrollStyle.Render(scrollInfo))
	
	return content.String()
}

// Messages for async operations
type lookupStartMsg struct{}

type lookupResultMsg struct {
	result domain.DNSResult
}

type lookupErrorMsg struct {
	error error
}