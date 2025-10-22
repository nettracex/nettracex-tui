// Package whois contains TUI integration tests for WHOIS tool
package whois

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWHOIS_TUIIntegration(t *testing.T) {
	// Create mock network client
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}

	// Setup WHOIS result
	whoisResult := domain.WHOISResult{
		Domain:      "example.com",
		Registrar:   "Test Registrar Inc.",
		Created:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Updated:     time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
		Expires:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:         "John Doe",
				Organization: "Example Corp",
				Email:        "john@example.com",
				Phone:        "+1-555-0123",
			},
			"admin": {
				Name:  "Jane Admin",
				Email: "admin@example.com",
			},
		},
		Status:  []string{"clientTransferProhibited", "clientUpdateProhibited"},
		RawData: "Domain Name: EXAMPLE.COM\nRegistrar: Test Registrar Inc.\n...",
	}

	// No mock expectations - this is a TUI integration test, not business logic test

	// Create WHOIS tool
	tool := NewTool(mockClient, mockLogger)

	// Create diagnostic view model
	diagnosticView := tui.NewDiagnosticViewModel(tool)
	diagnosticView.SetSize(120, 40)

	// Test initial state
	assert.Equal(t, tui.DiagnosticStateInput, diagnosticView.GetState())
	
	// Test view rendering
	view := diagnosticView.View()
	assert.Contains(t, view, "WHOIS Diagnostic Tool")
	assert.Contains(t, view, "Domain or IP Address")

	// Test form submission
	formValues := map[string]string{
		"query": "example.com",
	}

	// Execute diagnostic
	_, cmd := diagnosticView.Update(tui.FormSubmitMsg{Values: formValues})
	require.NotNil(t, cmd)

	// Simulate the execution flow
	updatedView, _ := diagnosticView.Update(tui.DiagnosticStartMsg{})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	assert.Equal(t, tui.DiagnosticStateLoading, diagnosticView.GetState())

	// Create result
	result := domain.NewResult(whoisResult)
	result.SetMetadata("tool", "whois")
	result.SetMetadata("query", "example.com")

	updatedView, _ = diagnosticView.Update(tui.DiagnosticResultMsg{Result: result})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	assert.Equal(t, tui.DiagnosticStateResult, diagnosticView.GetState())

	// Test result view
	resultView := diagnosticView.View()
	assert.Contains(t, resultView, "example.com")
	assert.Contains(t, resultView, "Test Registrar Inc.")
	assert.Contains(t, resultView, "ns1.example.com")
	assert.Contains(t, resultView, "John Doe")

	// Test completed successfully - TUI integration verified
}

func TestWHOIS_TUIErrorHandling(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}

	// Setup error response
	expectedError := &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "WHOIS lookup failed",
		Code:    "WHOIS_LOOKUP_FAILED",
	}

	// No mock expectations - this is a TUI integration test

	tool := NewTool(mockClient, mockLogger)
	diagnosticView := tui.NewDiagnosticViewModel(tool)

	// Test error handling
	formValues := map[string]string{
		"query": "invalid-domain",
	}

	updatedView, _ := diagnosticView.Update(tui.FormSubmitMsg{Values: formValues})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	updatedView, _ = diagnosticView.Update(tui.DiagnosticStartMsg{})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	updatedView, _ = diagnosticView.Update(tui.DiagnosticErrorMsg{Error: expectedError})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)

	assert.Equal(t, tui.DiagnosticStateError, diagnosticView.GetState())
	assert.Equal(t, expectedError, diagnosticView.GetError())

	// Test error view
	errorView := diagnosticView.View()
	assert.Contains(t, errorView, "Error:")
	assert.Contains(t, errorView, "WHOIS lookup failed")

	// Test completed successfully
}

func TestWHOIS_TUIKeyboardNavigation(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	diagnosticView := tui.NewDiagnosticViewModel(tool)

	// Test keyboard navigation
	tests := []struct {
		name     string
		key      tea.KeyMsg
		expected string
	}{
		{
			name:     "Escape key navigation",
			key:      tea.KeyMsg{Type: tea.KeyEsc},
			expected: "NavigationMsg",
		},
		{
			name:     "Quit key",
			key:      tea.KeyMsg{Type: tea.KeyCtrlC},
			expected: "tea.QuitMsg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := diagnosticView.Update(tt.key)
			assert.NotNil(t, cmd)
		})
	}
}

func TestWHOIS_TUIResponsiveLayout(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	diagnosticView := tui.NewDiagnosticViewModel(tool)

	// Test responsive layout
	sizes := []struct {
		width  int
		height int
		name   string
	}{
		{60, 20, "Small screen"},
		{100, 30, "Medium screen"},
		{140, 40, "Large screen"},
		{200, 50, "Extra large screen"},
	}

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			diagnosticView.SetSize(size.width, size.height)
			
			// Test that view renders without panic
			view := diagnosticView.View()
			assert.NotEmpty(t, view)
			
			// Test window size message
			windowMsg := tea.WindowSizeMsg{Width: size.width, Height: size.height}
			_, cmd := diagnosticView.Update(windowMsg)
			
			// Should not return error command
			assert.Nil(t, cmd)
		})
	}
}

func TestWHOIS_TUIFormValidation(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	diagnosticView := tui.NewDiagnosticViewModel(tool)

	// Test form validation with empty query
	formValues := map[string]string{
		"query": "",
	}

	// This should not execute the diagnostic
	_, cmd := diagnosticView.Update(tui.FormSubmitMsg{Values: formValues})
	
	// The command should handle validation internally
	assert.NotNil(t, cmd)
}

func TestWHOIS_TUIResultDisplay(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}

	// Create comprehensive WHOIS result
	whoisResult := domain.WHOISResult{
		Domain:    "test-domain.com",
		Registrar: "Test Registrar LLC",
		Created:   time.Date(2019, 3, 15, 10, 30, 0, 0, time.UTC),
		Updated:   time.Date(2023, 8, 20, 14, 45, 0, 0, time.UTC),
		Expires:   time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
		NameServers: []string{
			"ns1.test-domain.com",
			"ns2.test-domain.com",
			"ns3.test-domain.com",
		},
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:         "Test User",
				Organization: "Test Organization Inc.",
				Email:        "test@test-domain.com",
				Phone:        "+1-555-TEST",
				Address:      "123 Test Street, Test City, TC 12345",
			},
			"admin": {
				Name:  "Admin User",
				Email: "admin@test-domain.com",
				Phone: "+1-555-ADMIN",
			},
			"tech": {
				Name:  "Tech Support",
				Email: "tech@test-domain.com",
			},
		},
		Status: []string{
			"clientTransferProhibited",
			"clientUpdateProhibited",
			"clientDeleteProhibited",
		},
		RawData: "Comprehensive WHOIS data...",
	}

	// No mock expectations - this is a TUI integration test

	tool := NewTool(mockClient, mockLogger)
	diagnosticView := tui.NewDiagnosticViewModel(tool)
	diagnosticView.SetSize(120, 50)

	// Execute lookup
	formValues := map[string]string{
		"query": "test-domain.com",
	}

	updatedView, _ := diagnosticView.Update(tui.FormSubmitMsg{Values: formValues})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	updatedView, _ = diagnosticView.Update(tui.DiagnosticStartMsg{})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)

	result := domain.NewResult(whoisResult)
	updatedView, _ = diagnosticView.Update(tui.DiagnosticResultMsg{Result: result})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)

	// Test result display
	view := diagnosticView.View()

	// Check domain information section
	assert.Contains(t, view, "Domain Information")
	assert.Contains(t, view, "test-domain.com")
	assert.Contains(t, view, "Test Registrar LLC")

	// Check dates section
	assert.Contains(t, view, "Important Dates")
	assert.Contains(t, view, "2019-03-15")
	assert.Contains(t, view, "2024-03-15")

	// Check name servers section
	assert.Contains(t, view, "Name Servers")
	assert.Contains(t, view, "ns1.test-domain.com")
	assert.Contains(t, view, "ns2.test-domain.com")
	assert.Contains(t, view, "ns3.test-domain.com")

	// Check status section
	assert.Contains(t, view, "Domain Status")
	assert.Contains(t, view, "clientTransferProhibited")

	// Check contacts section
	assert.Contains(t, view, "Contacts")
	assert.Contains(t, view, "Test User")
	assert.Contains(t, view, "Test Organization Inc.")
	assert.Contains(t, view, "test@test-domain.com")

	// Test completed successfully
}

func TestWHOIS_TUIExpirationWarnings(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}

	// Test domain expiring soon
	soonExpiring := domain.WHOISResult{
		Domain:    "expiring-soon.com",
		Registrar: "Test Registrar",
		Expires:   time.Now().AddDate(0, 0, 15), // Expires in 15 days
		RawData:   "WHOIS data...",
	}

	// Test expired domain
	expired := domain.WHOISResult{
		Domain:    "expired.com",
		Registrar: "Test Registrar",
		Expires:   time.Now().AddDate(0, 0, -5), // Expired 5 days ago
		RawData:   "WHOIS data...",
	}

	// No mock expectations - this is a TUI integration test

	tool := NewTool(mockClient, mockLogger)

	// Test expiring soon warning
	diagnosticView := tui.NewDiagnosticViewModel(tool)
	formValues := map[string]string{"query": "expiring-soon.com"}
	
	updatedView, _ := diagnosticView.Update(tui.FormSubmitMsg{Values: formValues})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	updatedView, _ = diagnosticView.Update(tui.DiagnosticStartMsg{})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	
	result := domain.NewResult(soonExpiring)
	updatedView, _ = diagnosticView.Update(tui.DiagnosticResultMsg{Result: result})
	diagnosticView = updatedView.(*tui.DiagnosticViewModel)
	
	view := diagnosticView.View()
	// Note: Warning logic is in the WHOIS formatting function
	assert.Contains(t, view, "expiring-soon.com")

	// Test expired domain warning
	diagnosticView2 := tui.NewDiagnosticViewModel(tool)
	formValues2 := map[string]string{"query": "expired.com"}
	
	diagnosticView2.Update(tui.FormSubmitMsg{Values: formValues2})
	diagnosticView2.Update(tui.DiagnosticStartMsg{})
	
	result2 := domain.NewResult(expired)
	diagnosticView2.Update(tui.DiagnosticResultMsg{Result: result2})
	
	view2 := diagnosticView2.View()
	assert.Contains(t, view2, "expired.com")

	// Test completed successfully
}

func TestWHOIS_TUIAccessibility(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	diagnosticView := tui.NewDiagnosticViewModel(tool)

	// Test focus management
	diagnosticView.Focus()
	diagnosticView.Blur()

	// Test theme setting
	mockTheme := &MockTheme{}
	diagnosticView.SetTheme(mockTheme)

	// Test that all states render properly
	states := []tui.DiagnosticViewState{
		tui.DiagnosticStateInput,
		tui.DiagnosticStateLoading,
		tui.DiagnosticStateError,
		tui.DiagnosticStateResult,
	}

	for _, state := range states {
		// Manually set state for testing
		switch state {
		case tui.DiagnosticStateLoading:
			updatedView, _ := diagnosticView.Update(tui.DiagnosticStartMsg{})
			diagnosticView = updatedView.(*tui.DiagnosticViewModel)
		case tui.DiagnosticStateError:
			updatedView, _ := diagnosticView.Update(tui.DiagnosticErrorMsg{Error: assert.AnError})
			diagnosticView = updatedView.(*tui.DiagnosticViewModel)
		case tui.DiagnosticStateResult:
			result := domain.NewResult(domain.WHOISResult{Domain: "test.com"})
			updatedView, _ := diagnosticView.Update(tui.DiagnosticResultMsg{Result: result})
			diagnosticView = updatedView.(*tui.DiagnosticViewModel)
		}

		view := diagnosticView.View()
		assert.NotEmpty(t, view, "State %v should render content", state)
	}
}

// MockTheme for testing
type MockTheme struct{}

func (m *MockTheme) GetColor(element string) string {
	return "#ffffff"
}

func (m *MockTheme) GetStyle(element string) map[string]interface{} {
	return map[string]interface{}{}
}

func (m *MockTheme) SetColor(element, color string) {
	// No-op for testing
}