// Package ssl provides SSL certificate diagnostic TUI components
package ssl

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/tui"
)

// Model represents the SSL certificate check TUI model
type Model struct {
	tool         *Tool
	state        tui.ViewState
	hostInput    textinput.Model
	portInput    textinput.Model
	focusedInput int
	result       *domain.SSLResult
	error        error
	width        int
	height       int
	theme        domain.Theme
}

// NewModel creates a new SSL model
func NewModel(tool *Tool) *Model {
	hostInput := textinput.New()
	hostInput.Placeholder = "Enter hostname (e.g., google.com)"
	hostInput.Focus()
	hostInput.CharLimit = 253
	hostInput.Width = 50

	portInput := textinput.New()
	portInput.Placeholder = "443"
	portInput.CharLimit = 5
	portInput.Width = 10

	return &Model{
		tool:         tool,
		state:        tui.ViewStateInput,
		hostInput:    hostInput,
		portInput:    portInput,
		focusedInput: 0,
		theme:        tui.NewDefaultTheme(),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state == tui.ViewStateResult || m.state == tui.ViewStateError {
				m.state = tui.ViewStateInput
				m.result = nil
				m.error = nil
				return m, nil
			}
		case "enter":
			if m.state == tui.ViewStateInput {
				return m, m.executeSSLCheck()
			}
		case "tab", "shift+tab":
			if m.state == tui.ViewStateInput {
				if msg.String() == "tab" {
					m.focusedInput = (m.focusedInput + 1) % 2
				} else {
					m.focusedInput = (m.focusedInput - 1 + 2) % 2
				}
				m.updateInputFocus()
			}
		}

	case tui.SSLCheckCompleteMsg:
		m.state = tui.ViewStateResult
		m.result = &msg.Result
		return m, nil

	case tui.SSLCheckErrorMsg:
		m.state = tui.ViewStateError
		m.error = msg.Error
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.hostInput.Width = min(50, m.width-10)
		return m, nil
	}

	// Update inputs
	if m.state == tui.ViewStateInput {
		var cmd tea.Cmd
		if m.focusedInput == 0 {
			m.hostInput, cmd = m.hostInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			m.portInput, cmd = m.portInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m *Model) View() string {
	switch m.state {
	case tui.ViewStateInput:
		return m.renderInputView()
	case tui.ViewStateLoading:
		return m.renderLoadingView()
	case tui.ViewStateResult:
		return m.renderResultView()
	case tui.ViewStateError:
		return m.renderErrorView()
	default:
		return "Unknown state"
	}
}

// SetSize sets the model size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.hostInput.Width = min(50, width-10)
}

// SetTheme sets the model theme
func (m *Model) SetTheme(theme domain.Theme) {
	m.theme = theme
}

// Focus focuses the model
func (m *Model) Focus() {
	if m.state == tui.ViewStateInput {
		m.updateInputFocus()
	}
}

// Blur blurs the model
func (m *Model) Blur() {
	m.hostInput.Blur()
	m.portInput.Blur()
}

// updateInputFocus updates the focus state of inputs
func (m *Model) updateInputFocus() {
	if m.focusedInput == 0 {
		m.hostInput.Focus()
		m.portInput.Blur()
	} else {
		m.hostInput.Blur()
		m.portInput.Focus()
	}
}

// executeSSLCheck executes the SSL certificate check
func (m *Model) executeSSLCheck() tea.Cmd {
	host := strings.TrimSpace(m.hostInput.Value())
	portStr := strings.TrimSpace(m.portInput.Value())
	
	if host == "" {
		return func() tea.Msg {
			return tui.SSLCheckErrorMsg{Error: fmt.Errorf("host is required")}
		}
	}
	
	// Default port if not specified
	if portStr == "" {
		portStr = "443"
	}
	
	m.state = tui.ViewStateLoading
	
	return func() tea.Msg {
		// Create parameters
		params := domain.NewParameters()
		params.Set("host", host)
		params.Set("port", portStr)
		
		// Execute SSL check
		result, err := m.tool.Execute(context.Background(), params)
		if err != nil {
			return tui.SSLCheckErrorMsg{Error: err}
		}
		
		sslResult, ok := result.Data().(domain.SSLResult)
		if !ok {
			return tui.SSLCheckErrorMsg{Error: fmt.Errorf("invalid result type")}
		}
		
		return tui.SSLCheckCompleteMsg{Result: sslResult}
	}
}

// renderInputView renders the input form
func (m *Model) renderInputView() string {
	var b strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.theme.GetColor("primary"))).
		MarginBottom(1)
	
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.theme.GetColor("text")))
	
	b.WriteString(titleStyle.Render("SSL Certificate Check"))
	b.WriteString("\n\n")
	
	// Host input
	b.WriteString(labelStyle.Render("Host:"))
	b.WriteString("\n")
	b.WriteString(m.hostInput.View())
	b.WriteString("\n\n")
	
	// Port input
	b.WriteString(labelStyle.Render("Port:"))
	b.WriteString("\n")
	b.WriteString(m.portInput.View())
	b.WriteString("\n\n")
	
	// Instructions
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.GetColor("muted"))).
		Italic(true)
	
	b.WriteString(helpStyle.Render("Tab: Switch fields • Enter: Check certificate • Esc: Back • Ctrl+C: Quit"))
	
	return b.String()
}

// renderLoadingView renders the loading state
func (m *Model) renderLoadingView() string {
	loadingStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.theme.GetColor("primary")))
	
	return loadingStyle.Render("Checking SSL certificate...")
}

// renderResultView renders the SSL check results
func (m *Model) renderResultView() string {
	if m.result == nil {
		return "No results available"
	}
	
	var b strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.theme.GetColor("primary"))).
		MarginBottom(1)
	
	b.WriteString(titleStyle.Render(fmt.Sprintf("SSL Certificate: %s:%d", m.result.Host, m.result.Port)))
	b.WriteString("\n\n")
	
	// Certificate status
	statusStyle := lipgloss.NewStyle().Bold(true)
	if m.result.Valid {
		statusStyle = statusStyle.Foreground(lipgloss.Color(m.theme.GetColor("success")))
		b.WriteString(statusStyle.Render("✅ Certificate Valid"))
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color(m.theme.GetColor("error")))
		b.WriteString(statusStyle.Render("❌ Certificate Invalid"))
	}
	b.WriteString("\n\n")
	
	// Certificate details
	if m.result.Certificate != nil {
		cert := m.result.Certificate
		
		detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetColor("text")))
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.GetColor("accent")))
		
		b.WriteString(labelStyle.Render("Subject: "))
		b.WriteString(detailStyle.Render(m.result.Subject))
		b.WriteString("\n")
		
		b.WriteString(labelStyle.Render("Issuer: "))
		b.WriteString(detailStyle.Render(m.result.Issuer))
		b.WriteString("\n")
		
		b.WriteString(labelStyle.Render("Valid From: "))
		b.WriteString(detailStyle.Render(cert.NotBefore.Format("2006-01-02 15:04:05 UTC")))
		b.WriteString("\n")
		
		b.WriteString(labelStyle.Render("Valid Until: "))
		b.WriteString(detailStyle.Render(cert.NotAfter.Format("2006-01-02 15:04:05 UTC")))
		b.WriteString("\n")
		
		// Days until expiry
		daysUntilExpiry := int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24)
		expiryStyle := detailStyle
		if daysUntilExpiry <= 30 && daysUntilExpiry > 0 {
			expiryStyle = expiryStyle.Foreground(lipgloss.Color(m.theme.GetColor("warning")))
		} else if daysUntilExpiry <= 0 {
			expiryStyle = expiryStyle.Foreground(lipgloss.Color(m.theme.GetColor("error")))
		}
		
		b.WriteString(labelStyle.Render("Days Until Expiry: "))
		if daysUntilExpiry > 0 {
			b.WriteString(expiryStyle.Render(fmt.Sprintf("%d", daysUntilExpiry)))
		} else {
			b.WriteString(expiryStyle.Render("EXPIRED"))
		}
		b.WriteString("\n")
		
		b.WriteString(labelStyle.Render("Signature Algorithm: "))
		b.WriteString(detailStyle.Render(cert.SignatureAlgorithm.String()))
		b.WriteString("\n")
		
		// Key size for RSA
		if cert.PublicKeyAlgorithm.String() == "RSA" {
			if rsaKey, ok := cert.PublicKey.(interface{ Size() int }); ok {
				keySize := rsaKey.Size() * 8
				b.WriteString(labelStyle.Render("Key Size: "))
				keyStyle := detailStyle
				if keySize < 2048 {
					keyStyle = keyStyle.Foreground(lipgloss.Color(m.theme.GetColor("warning")))
				}
				b.WriteString(keyStyle.Render(fmt.Sprintf("%d bits", keySize)))
				b.WriteString("\n")
			}
		}
		
		// Subject Alternative Names
		if len(m.result.SANs) > 0 {
			b.WriteString("\n")
			b.WriteString(labelStyle.Render("Subject Alternative Names:"))
			b.WriteString("\n")
			for _, san := range m.result.SANs {
				b.WriteString(detailStyle.Render(fmt.Sprintf("  • %s", san)))
				b.WriteString("\n")
			}
		}
		
		// Certificate chain
		if len(m.result.Chain) > 1 {
			b.WriteString("\n")
			b.WriteString(labelStyle.Render(fmt.Sprintf("Certificate Chain (%d certificates):", len(m.result.Chain))))
			b.WriteString("\n")
			for i, chainCert := range m.result.Chain {
				if i == 0 {
					b.WriteString(detailStyle.Render(fmt.Sprintf("  1. %s (End Entity)", chainCert.Subject.CommonName)))
				} else {
					b.WriteString(detailStyle.Render(fmt.Sprintf("  %d. %s", i+1, chainCert.Subject.CommonName)))
				}
				b.WriteString("\n")
			}
		}
	}
	
	// Security issues
	if len(m.result.Errors) > 0 {
		b.WriteString("\n")
		errorStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(m.theme.GetColor("error")))
		
		b.WriteString(errorStyle.Render("Security Issues:"))
		b.WriteString("\n")
		
		issueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetColor("error")))
		for _, err := range m.result.Errors {
			b.WriteString(issueStyle.Render(fmt.Sprintf("  ⚠️  %s", err)))
			b.WriteString("\n")
		}
	}
	
	// Security recommendations
	recommendations := GetSecurityRecommendations(*m.result)
	if len(recommendations) > 0 {
		b.WriteString("\n")
		recStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(m.theme.GetColor("accent")))
		
		b.WriteString(recStyle.Render("Recommendations:"))
		b.WriteString("\n")
		
		for _, rec := range recommendations {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetColor("text"))).Render(fmt.Sprintf("  • %s", rec)))
			b.WriteString("\n")
		}
	}
	
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.GetColor("muted"))).
		Italic(true)
	
	b.WriteString(helpStyle.Render("Esc: Back • Ctrl+C: Quit"))
	
	return b.String()
}

// renderErrorView renders the error state
func (m *Model) renderErrorView() string {
	var b strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.theme.GetColor("error"))).
		MarginBottom(1)
	
	b.WriteString(titleStyle.Render("SSL Check Error"))
	b.WriteString("\n\n")
	
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.GetColor("error")))
	b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.error)))
	b.WriteString("\n\n")
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.GetColor("muted"))).
		Italic(true)
	
	b.WriteString(helpStyle.Render("Esc: Back • Ctrl+C: Quit"))
	
	return b.String()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}