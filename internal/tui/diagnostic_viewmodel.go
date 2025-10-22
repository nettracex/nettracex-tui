// Package tui contains diagnostic view models for TUI integration
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// ViewState represents the current state of a view
type ViewState int

const (
	ViewStateInput ViewState = iota
	ViewStateLoading
	ViewStateResult
	ViewStateError
)

// DiagnosticViewState represents the current state of the diagnostic view
type DiagnosticViewState int

const (
	DiagnosticStateInput DiagnosticViewState = iota
	DiagnosticStateLoading
	DiagnosticStateResult
	DiagnosticStateError
)

// DiagnosticViewModel wraps diagnostic tools for TUI integration
type DiagnosticViewModel struct {
	tool        domain.DiagnosticTool
	inputForm   *FormModel
	resultView  *ResultViewModel
	state       DiagnosticViewState
	width       int
	height      int
	theme       domain.Theme
	keyMap      KeyMap
	error       error
	loading     bool
	result      domain.Result
}

// NewDiagnosticViewModel creates a new diagnostic view model
func NewDiagnosticViewModel(tool domain.DiagnosticTool) *DiagnosticViewModel {
	// Create input form based on tool type
	form := NewFormModel(fmt.Sprintf("%s - %s", tool.Name(), tool.Description()))
	
	// Add fields based on tool type
	switch tool.Name() {
	case "whois":
		form.AddField("query", "Domain or IP Address", true)
		form.SetFieldValue("query", "")
	case "ping":
		form.AddField("host", "Host", true)
		form.AddField("count", "Count", false)
		form.SetFieldValue("count", "4")
	case "dns":
		form.AddField("domain", "Domain", true)
		form.AddField("record_type", "Record Type (A, AAAA, MX, TXT, CNAME, NS, or ALL for all types)", false)
		form.SetFieldValue("record_type", "A")
	case "ssl":
		form.AddField("host", "Host", true)
		form.AddField("port", "Port", false)
		form.SetFieldValue("port", "443")
	case "traceroute":
		form.AddField("host", "Host", true)
		form.AddField("max_hops", "Max Hops", false)
		form.SetFieldValue("max_hops", "30")
	}

	return &DiagnosticViewModel{
		tool:       tool,
		inputForm:  form,
		resultView: NewResultViewModel(),
		state:      DiagnosticStateInput,
		keyMap:     DefaultKeyMap(),
	}
}

// Init implements tea.Model
func (m *DiagnosticViewModel) Init() tea.Cmd {
	return m.inputForm.Init()
}

// Update implements tea.Model
func (m *DiagnosticViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Back):
			if m.state != DiagnosticStateInput {
				m.state = DiagnosticStateInput
				m.inputForm.Focus()
				m.error = nil
				m.result = nil
				return m, nil
			}
			// Send navigation back message
			return m, func() tea.Msg {
				return NavigationMsg{
					Action: NavigationActionBack,
				}
			}
		}

	case FormSubmitMsg:
		// Handle form submission
		return m, m.executeDiagnostic(msg.Values)

	case DiagnosticStartMsg:
		m.state = DiagnosticStateLoading
		m.loading = true
		return m, nil

	case DiagnosticResultMsg:
		m.state = DiagnosticStateResult
		m.loading = false
		m.result = msg.Result
		m.resultView.SetResult(msg.Result)
		return m, nil

	case DiagnosticErrorMsg:
		m.state = DiagnosticStateError
		m.loading = false
		m.error = msg.Error
		return m, nil
	}

	// Update the appropriate sub-model based on state
	switch m.state {
	case DiagnosticStateInput:
		updatedForm, formCmd := m.inputForm.Update(msg)
		m.inputForm = updatedForm.(*FormModel)
		cmds = append(cmds, formCmd)

	case DiagnosticStateResult:
		if m.resultView != nil {
			updatedResult, resultCmd := m.resultView.Update(msg)
			m.resultView = updatedResult.(*ResultViewModel)
			cmds = append(cmds, resultCmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *DiagnosticViewModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(m.renderHeader())
	content.WriteString("\n\n")

	// Main content based on state
	switch m.state {
	case DiagnosticStateInput:
		content.WriteString(m.inputForm.View())
	case DiagnosticStateLoading:
		content.WriteString(m.renderLoading())
	case DiagnosticStateResult:
		if m.resultView != nil {
			content.WriteString(m.resultView.View())
		}
	case DiagnosticStateError:
		content.WriteString(m.renderError())
	}

	// Footer
	content.WriteString("\n\n")
	content.WriteString(m.renderFooter())

	return content.String()
}

// renderHeader renders the diagnostic tool header
func (m *DiagnosticViewModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	title := strings.ToUpper(m.tool.Name()) + " Diagnostic Tool"
	description := m.tool.Description()

	return titleStyle.Render(title) + "\n" + descStyle.Render(description)
}

// renderLoading renders the loading state
func (m *DiagnosticViewModel) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	// For simplicity, just show a static loading message
	return loadingStyle.Render("🔍 Executing " + m.tool.Name() + " diagnostic...")
}

// renderError renders the error state
func (m *DiagnosticViewModel) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	retryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	var content strings.Builder
	content.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error: %s", m.error.Error())))
	content.WriteString("\n\n")
	content.WriteString(retryStyle.Render("Press ESC to try again or Q to quit"))

	return content.String()
}

// renderFooter renders the footer with help text
func (m *DiagnosticViewModel) renderFooter() string {
	var help []string

	switch m.state {
	case DiagnosticStateInput:
		help = []string{"tab: next field", "enter: execute", "esc: back", "q: quit"}
	case DiagnosticStateResult, DiagnosticStateError:
		help = []string{"esc: new query", "q: quit"}
	case DiagnosticStateLoading:
		help = []string{"q: quit"}
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	return helpStyle.Render(strings.Join(help, " • "))
}

// executeDiagnostic executes the diagnostic tool with the provided parameters
func (m *DiagnosticViewModel) executeDiagnostic(values map[string]string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return DiagnosticStartMsg{} },
		func() tea.Msg {
			// Create parameters based on tool type
			var params domain.Parameters
			var err error

			switch m.tool.Name() {
			case "whois":
				query := values["query"]
				params = domain.NewWHOISParameters(query)
			case "ping":
				host := values["host"]
				// For now, use default ping options
				options := domain.PingOptions{
					Count:      4,
					PacketSize: 64,
					TTL:        64,
				}
				params = domain.NewPingParameters(host, options)
			case "dns":
				domainName := values["domain"]
				recordTypeStr := values["record_type"]
				
				// Parse the record type string
				var recordType domain.DNSRecordType
				if recordTypeStr != "" {
					switch strings.ToUpper(strings.TrimSpace(recordTypeStr)) {
					case "A":
						recordType = domain.DNSRecordTypeA
					case "AAAA":
						recordType = domain.DNSRecordTypeAAAA
					case "MX":
						recordType = domain.DNSRecordTypeMX
					case "TXT":
						recordType = domain.DNSRecordTypeTXT
					case "CNAME":
						recordType = domain.DNSRecordTypeCNAME
					case "NS":
						recordType = domain.DNSRecordTypeNS
					case "SOA":
						recordType = domain.DNSRecordTypeSOA
					case "PTR":
						recordType = domain.DNSRecordTypePTR
					default:
						recordType = domain.DNSRecordTypeA // Default fallback
					}
				} else {
					recordType = domain.DNSRecordTypeA // Default to A record
				}
				
				params = domain.NewDNSParameters(domainName, recordType)
				
				// If user wants all record types (empty or "ALL"), set multiple types
				if recordTypeStr == "" || strings.ToUpper(strings.TrimSpace(recordTypeStr)) == "ALL" {
					allTypes := []domain.DNSRecordType{
						domain.DNSRecordTypeA,
						domain.DNSRecordTypeAAAA,
						domain.DNSRecordTypeMX,
						domain.DNSRecordTypeTXT,
						domain.DNSRecordTypeCNAME,
						domain.DNSRecordTypeNS,
					}
					params.Set("record_types", allTypes)
				}
			case "ssl":
				host := values["host"]
				port := 443 // Default HTTPS port
				params = domain.NewSSLParameters(host, port)
			case "traceroute":
				host := values["host"]
				options := domain.TraceOptions{
					MaxHops:    30,
					Timeout:    5 * time.Second, // Add missing timeout
					PacketSize: 64,
					Queries:    3,
					IPv6:       false,
				}
				params = domain.NewTracerouteParameters(host, options)
			default:
				return DiagnosticErrorMsg{Error: fmt.Errorf("unsupported tool: %s", m.tool.Name())}
			}

			// Execute the diagnostic
			result, err := m.tool.Execute(context.Background(), params)
			if err != nil {
				return DiagnosticErrorMsg{Error: err}
			}

			return DiagnosticResultMsg{Result: result}
		},
	)
}

// SetSize implements domain.TUIComponent
func (m *DiagnosticViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	if m.inputForm != nil {
		m.inputForm.SetSize(width, height)
	}

	if m.resultView != nil {
		m.resultView.SetSize(width, height)
	}
}

// SetTheme implements domain.TUIComponent
func (m *DiagnosticViewModel) SetTheme(theme domain.Theme) {
	m.theme = theme

	if m.inputForm != nil {
		m.inputForm.SetTheme(theme)
	}

	if m.resultView != nil {
		m.resultView.SetTheme(theme)
	}
}

// Focus implements domain.TUIComponent
func (m *DiagnosticViewModel) Focus() {
	if m.state == DiagnosticStateInput && m.inputForm != nil {
		m.inputForm.Focus()
	}
}

// Blur implements domain.TUIComponent
func (m *DiagnosticViewModel) Blur() {
	if m.inputForm != nil {
		m.inputForm.Blur()
	}
}

// GetTool returns the underlying diagnostic tool
func (m *DiagnosticViewModel) GetTool() domain.DiagnosticTool {
	return m.tool
}

// GetState returns the current view state
func (m *DiagnosticViewModel) GetState() DiagnosticViewState {
	return m.state
}

// IsLoading returns whether the view is in loading state
func (m *DiagnosticViewModel) IsLoading() bool {
	return m.loading
}

// GetResult returns the current result
func (m *DiagnosticViewModel) GetResult() domain.Result {
	return m.result
}

// GetError returns the current error
func (m *DiagnosticViewModel) GetError() error {
	return m.error
}

// Messages for diagnostic operations
type DiagnosticStartMsg struct{}

type DiagnosticResultMsg struct {
	Result domain.Result
}

type DiagnosticErrorMsg struct {
	Error error
}

// SSL-specific messages
type SSLCheckCompleteMsg struct {
	Result domain.SSLResult
}

type SSLCheckErrorMsg struct {
	Error error
}