// Package whois provides tests for WHOIS TUI model
package whois

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewModel(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	model := NewModel(tool)

	assert.NotNil(t, model)
	assert.Equal(t, tool, model.tool)
	assert.Equal(t, StateInput, model.state)
	assert.False(t, model.loading)
	assert.NotNil(t, model.input)
	assert.True(t, model.input.Focused())
}

func TestModel_Init(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestModel_SetSize(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetSize(100, 50)

	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)
	assert.Equal(t, 96, model.input.Width) // width - 4
}

func TestModel_Focus_Blur(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test Focus
	model.state = StateInput
	model.input.Blur() // Start unfocused
	model.Focus()
	assert.True(t, model.input.Focused())

	// Test Blur
	model.Blur()
	assert.False(t, model.input.Focused())

	// Test Focus when not in input state
	model.state = StateResult
	model.Focus() // Should not focus input when not in input state
}

func TestModel_Update_KeyMessages(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	tests := []struct {
		name          string
		key           string
		initialState  ModelState
		inputValue    string
		expectedState ModelState
		expectQuit    bool
	}{
		{
			name:          "quit with ctrl+c",
			key:           "ctrl+c",
			initialState:  StateInput,
			expectedState: StateInput,
			expectQuit:    true,
		},
		{
			name:          "quit with q",
			key:           "q",
			initialState:  StateInput,
			expectedState: StateInput,
			expectQuit:    true,
		},
		{
			name:          "escape from result to input",
			key:           "esc",
			initialState:  StateResult,
			expectedState: StateInput,
			expectQuit:    false,
		},
		{
			name:          "escape from error to input",
			key:           "esc",
			initialState:  StateError,
			expectedState: StateInput,
			expectQuit:    false,
		},
		{
			name:          "escape in input state does nothing",
			key:           "esc",
			initialState:  StateInput,
			expectedState: StateInput,
			expectQuit:    false,
		},
		{
			name:          "enter with empty input",
			key:           "enter",
			initialState:  StateInput,
			inputValue:    "",
			expectedState: StateInput,
			expectQuit:    false,
		},
		{
			name:          "enter with valid input",
			key:           "enter",
			initialState:  StateInput,
			inputValue:    "example.com",
			expectedState: StateInput, // State change happens in async message
			expectQuit:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset model state
			model.state = tt.initialState
			model.input.SetValue(tt.inputValue)

			// Create key message
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else if tt.key == "enter" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			}

			// Update model
			updatedModel, cmd := model.Update(keyMsg)
			model = updatedModel.(*Model)

			// Check state
			assert.Equal(t, tt.expectedState, model.state)

			// Check quit command
			if tt.expectQuit {
				assert.NotNil(t, cmd)
			}
		})
	}
}

func TestModel_Update_LookupMessages(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test lookup start message
	model.state = StateInput
	updatedModel, cmd := model.Update(lookupStartMsg{})
	model = updatedModel.(*Model)

	assert.Equal(t, StateLoading, model.state)
	assert.True(t, model.loading)
	assert.Nil(t, cmd)

	// Test lookup result message
	whoisResult := domain.WHOISResult{
		Domain:    "example.com",
		Registrar: "Test Registrar",
		RawData:   "Test data",
	}
	
	updatedModel, cmd = model.Update(lookupResultMsg{result: whoisResult})
	model = updatedModel.(*Model)

	assert.Equal(t, StateResult, model.state)
	assert.False(t, model.loading)
	assert.Equal(t, whoisResult, model.result)
	assert.Nil(t, cmd)

	// Test lookup error message
	testError := assert.AnError
	updatedModel, cmd = model.Update(lookupErrorMsg{error: testError})
	model = updatedModel.(*Model)

	assert.Equal(t, StateError, model.state)
	assert.False(t, model.loading)
	assert.Equal(t, testError, model.error)
	assert.Nil(t, cmd)
}

func TestModel_View_States(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test input state view
	model.state = StateInput
	view := model.View()
	assert.Contains(t, view, "WHOIS Lookup Tool")
	assert.Contains(t, view, "Query:")
	assert.Contains(t, view, "enter: lookup")

	// Test loading state view
	model.state = StateLoading
	model.input.SetValue("example.com")
	view = model.View()
	assert.Contains(t, view, "Looking up WHOIS information")
	assert.Contains(t, view, "example.com")

	// Test result state view
	model.state = StateResult
	model.result = domain.WHOISResult{
		Domain:      "example.com",
		Registrar:   "Test Registrar",
		Created:     time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Expires:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
		Status:      []string{"clientTransferProhibited"},
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:         "John Doe",
				Organization: "Test Corp",
				Email:        "admin@example.com",
			},
		},
	}
	view = model.View()
	assert.Contains(t, view, "Domain Information")
	assert.Contains(t, view, "example.com")
	assert.Contains(t, view, "Test Registrar")
	assert.Contains(t, view, "Important Dates")
	assert.Contains(t, view, "Name Servers")
	assert.Contains(t, view, "ns1.example.com")
	assert.Contains(t, view, "Domain Status")
	assert.Contains(t, view, "clientTransferProhibited")
	assert.Contains(t, view, "Contacts")
	assert.Contains(t, view, "John Doe")
	assert.Contains(t, view, "esc: new lookup")

	// Test error state view
	model.state = StateError
	model.error = assert.AnError
	view = model.View()
	assert.Contains(t, view, "Error:")
	assert.Contains(t, view, "esc: new lookup")
}

func TestModel_renderSection(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	data := [][]string{
		{"Domain", "example.com"},
		{"Registrar", "Test Registrar"},
		{"", ""}, // Empty row should be skipped
		{"Created", "2020-01-01"},
	}

	section := model.renderSection("Test Section", data)

	assert.Contains(t, section, "Test Section")
	assert.Contains(t, section, "Domain")
	assert.Contains(t, section, "example.com")
	assert.Contains(t, section, "Registrar")
	assert.Contains(t, section, "Test Registrar")
	assert.Contains(t, section, "Created")
	assert.Contains(t, section, "2020-01-01")
}

func TestModel_renderContactsSection(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.result = domain.WHOISResult{
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:         "John Doe",
				Organization: "Test Corp",
				Email:        "john@example.com",
				Phone:        "+1-555-0123",
			},
			"admin": {
				Name:  "Jane Smith",
				Email: "jane@example.com",
			},
			"empty": {}, // Empty contact should be skipped
		},
	}

	section := model.renderContactsSection()

	assert.Contains(t, section, "Contacts")
	assert.Contains(t, section, "Registrant")
	assert.Contains(t, section, "John Doe")
	assert.Contains(t, section, "Test Corp")
	assert.Contains(t, section, "john@example.com")
	assert.Contains(t, section, "+1-555-0123")
	assert.Contains(t, section, "Admin")
	assert.Contains(t, section, "Jane Smith")
	assert.Contains(t, section, "jane@example.com")
}

func TestModel_performLookup(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set up mock expectations
	expectedResult := domain.WHOISResult{
		Domain:    "example.com",
		Registrar: "Test Registrar",
		RawData:   "Test data",
	}
	
	mockLogger.On("Info", "Executing WHOIS lookup", mock.Anything).Return()
	mockLogger.On("Info", "WHOIS lookup completed successfully", mock.Anything, mock.Anything, mock.Anything).Return()
	mockClient.On("WHOISLookup", mock.Anything, "example.com").Return(expectedResult, nil)

	// Set input value
	model.input.SetValue("example.com")

	// Perform lookup
	cmd := model.performLookup()

	// The command should be a batch command
	assert.NotNil(t, cmd)

	// Execute the batch command to test the async operations
	// Note: In a real test, you'd need to handle the async nature properly
	// This is a simplified test to verify the command structure
}

func TestModel_GetModel(t *testing.T) {
	mockClient := &MockNetworkClient{}
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	model := tool.GetModel()
	assert.NotNil(t, model)

	// Verify it's the correct type
	whoisModel, ok := model.(*Model)
	assert.True(t, ok)
	assert.Equal(t, tool, whoisModel.tool)
}