// Package whois provides TUI model for WHOIS diagnostic tool
package whois

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Model represents the WHOIS tool TUI model
type Model struct {
	tool        *Tool
	state       ModelState
	input       textinput.Model
	result      domain.WHOISResult
	error       error
	width       int
	height      int
	theme       domain.Theme
	loading     bool
}

// ModelState represents the current state of the model
type ModelState int

const (
	StateInput ModelState = iota
	StateLoading
	StateResult
	StateError
)

// NewModel creates a new WHOIS model
func NewModel(tool *Tool) *Model {
	input := textinput.New()
	input.Placeholder = "Enter domain name or IP address (e.g., example.com, 8.8.8.8)"
	input.Focus()
	input.CharLimit = 253
	input.Width = 50

	return &Model{
		tool:    tool,
		state:   StateInput,
		input:   input,
		loading: false,
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
			if m.state != StateInput {
				m.state = StateInput
				m.input.SetValue("")
				m.input.Focus()
				m.error = nil
				return m, nil
			}
		case "enter":
			if m.state == StateInput && m.input.Value() != "" {
				return m, m.performLookup()
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
	title := "WHOIS Lookup Tool"
	description := "Query domain registration and IP address information"
	
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
	
	content.WriteString(labelStyle.Render("Query:"))
	content.WriteString("\n")
	content.WriteString(m.input.View())
	content.WriteString("\n\n")
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	
	content.WriteString(helpStyle.Render("Enter a domain name (e.g., example.com) or IP address (e.g., 8.8.8.8)"))
	
	return content.String()
}

// renderLoading renders the loading state
func (m *Model) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	
	return loadingStyle.Render(fmt.Sprintf("🔍 Looking up WHOIS information for '%s'...", m.input.Value()))
}

// renderResult renders the WHOIS result
func (m *Model) renderResult() string {
	if m.result.Domain == "" {
		return "No result available"
	}
	
	var content strings.Builder
	
	// Domain info section
	content.WriteString(m.renderSection("Domain Information", [][]string{
		{"Domain", m.result.Domain},
		{"Registrar", m.result.Registrar},
	}))
	
	// Dates section
	dateInfo := [][]string{}
	if !m.result.Created.IsZero() {
		dateInfo = append(dateInfo, []string{"Created", m.result.Created.Format("2006-01-02 15:04:05")})
	}
	if !m.result.Updated.IsZero() {
		dateInfo = append(dateInfo, []string{"Updated", m.result.Updated.Format("2006-01-02 15:04:05")})
	}
	if !m.result.Expires.IsZero() {
		dateInfo = append(dateInfo, []string{"Expires", m.result.Expires.Format("2006-01-02 15:04:05")})
	}
	
	if len(dateInfo) > 0 {
		content.WriteString("\n")
		content.WriteString(m.renderSection("Important Dates", dateInfo))
	}
	
	// Name servers section
	if len(m.result.NameServers) > 0 {
		content.WriteString("\n")
		nsInfo := [][]string{}
		for i, ns := range m.result.NameServers {
			nsInfo = append(nsInfo, []string{fmt.Sprintf("NS %d", i+1), ns})
		}
		content.WriteString(m.renderSection("Name Servers", nsInfo))
	}
	
	// Status section
	if len(m.result.Status) > 0 {
		content.WriteString("\n")
		statusInfo := [][]string{}
		for i, status := range m.result.Status {
			statusInfo = append(statusInfo, []string{fmt.Sprintf("Status %d", i+1), status})
		}
		content.WriteString(m.renderSection("Domain Status", statusInfo))
	}
	
	// Contacts section
	if len(m.result.Contacts) > 0 {
		content.WriteString("\n")
		content.WriteString(m.renderContactsSection())
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

// renderContactsSection renders the contacts section
func (m *Model) renderContactsSection() string {
	var content strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	content.WriteString(titleStyle.Render("Contacts"))
	content.WriteString("\n")
	
	for contactType, contact := range m.result.Contacts {
		if contact.Name != "" || contact.Email != "" || contact.Organization != "" {
			contactData := [][]string{}
			
			if contact.Name != "" {
				contactData = append(contactData, []string{"Name", contact.Name})
			}
			if contact.Organization != "" {
				contactData = append(contactData, []string{"Organization", contact.Organization})
			}
			if contact.Email != "" {
				contactData = append(contactData, []string{"Email", contact.Email})
			}
			if contact.Phone != "" {
				contactData = append(contactData, []string{"Phone", contact.Phone})
			}
			
			if len(contactData) > 0 {
				content.WriteString(m.renderSection(strings.Title(contactType), contactData))
				content.WriteString("\n")
			}
		}
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
		help = []string{"enter: lookup", "q: quit"}
	case StateResult, StateError:
		help = []string{"esc: new lookup", "q: quit"}
	case StateLoading:
		help = []string{"q: quit"}
	}
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	return helpStyle.Render(strings.Join(help, " • "))
}

// performLookup performs the WHOIS lookup
func (m *Model) performLookup() tea.Cmd {
	query := strings.TrimSpace(m.input.Value())
	
	return tea.Batch(
		func() tea.Msg { return lookupStartMsg{} },
		func() tea.Msg {
			// Create parameters
			params := domain.NewWHOISParameters(query)
			
			// Execute lookup
			result, err := m.tool.Execute(context.Background(), params)
			if err != nil {
				return lookupErrorMsg{error: err}
			}
			
			// Extract WHOIS result
			whoisResult, ok := result.Data().(domain.WHOISResult)
			if !ok {
				return lookupErrorMsg{error: fmt.Errorf("invalid result type")}
			}
			
			return lookupResultMsg{result: whoisResult}
		},
	)
}

// Messages for async operations
type lookupStartMsg struct{}

type lookupResultMsg struct {
	result domain.WHOISResult
}

type lookupErrorMsg struct {
	error error
}