// Package tui contains tests for diagnostic view models
package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDiagnosticTool implements domain.DiagnosticTool for testing
type MockDiagnosticTool struct {
	mock.Mock
}

func (m *MockDiagnosticTool) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDiagnosticTool) Description() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDiagnosticTool) Execute(ctx context.Context, params domain.Parameters) (domain.Result, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(domain.Result), args.Error(1)
}

func (m *MockDiagnosticTool) Validate(params domain.Parameters) error {
	args := m.Called(params)
	return args.Error(0)
}

func (m *MockDiagnosticTool) GetModel() tea.Model {
	args := m.Called()
	return args.Get(0).(tea.Model)
}

// MockResult implements domain.Result for testing
type MockResult struct {
	mock.Mock
	data     interface{}
	metadata map[string]interface{}
}

func (m *MockResult) Data() interface{} {
	return m.data
}

func (m *MockResult) Metadata() map[string]interface{} {
	return m.metadata
}

func (m *MockResult) Format(formatter domain.OutputFormatter) string {
	args := m.Called(formatter)
	return args.String(0)
}

func (m *MockResult) Export(format domain.ExportFormat) ([]byte, error) {
	args := m.Called(format)
	return args.Get(0).([]byte), args.Error(1)
}

func TestNewDiagnosticViewModel(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected DiagnosticViewState
	}{
		{
			name:     "WHOIS tool creates correct view model",
			toolName: "whois",
			expected: DiagnosticStateInput,
		},
		{
			name:     "Ping tool creates correct view model",
			toolName: "ping",
			expected: DiagnosticStateInput,
		},
		{
			name:     "DNS tool creates correct view model",
			toolName: "dns",
			expected: DiagnosticStateInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTool := &MockDiagnosticTool{}
			mockTool.On("Name").Return(tt.toolName)
			mockTool.On("Description").Return("Test description")

			viewModel := NewDiagnosticViewModel(mockTool)

			assert.NotNil(t, viewModel)
			assert.Equal(t, tt.expected, viewModel.GetState())
			assert.NotNil(t, viewModel.inputForm)
			assert.NotNil(t, viewModel.resultView)
			assert.Equal(t, mockTool, viewModel.GetTool())

			mockTool.AssertExpectations(t)
		})
	}
}

func TestDiagnosticViewModel_WHOISIntegration(t *testing.T) {
	// Create mock WHOIS tool
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")

	// Create WHOIS result
	whoisResult := domain.WHOISResult{
		Domain:      "example.com",
		Registrar:   "Test Registrar",
		Created:     time.Now().AddDate(-5, 0, 0),
		Expires:     time.Now().AddDate(1, 0, 0),
		NameServers: []string{"ns1.example.com", "ns2.example.com"},
		Contacts: map[string]domain.Contact{
			"registrant": {
				Name:  "John Doe",
				Email: "john@example.com",
			},
		},
		Status:  []string{"clientTransferProhibited"},
		RawData: "Raw WHOIS data...",
	}

	mockResult := &MockResult{
		data: whoisResult,
		metadata: map[string]interface{}{
			"tool":      "whois",
			"query":     "example.com",
			"timestamp": time.Now(),
		},
	}

	mockTool.On("Execute", mock.Anything, mock.Anything).Return(mockResult, nil)
	mockResult.On("Export", domain.ExportFormatJSON).Return([]byte(`{"domain":"example.com"}`), nil)

	// Create view model
	viewModel := NewDiagnosticViewModel(mockTool)
	viewModel.SetSize(80, 24)

	// Test initial state
	assert.Equal(t, DiagnosticStateInput, viewModel.GetState())
	assert.False(t, viewModel.IsLoading())

	// Test form submission - simulate the actual command execution
	formValues := map[string]string{
		"query": "example.com",
	}

	// Process form submission message
	updatedModel, cmd := viewModel.Update(FormSubmitMsg{Values: formValues})
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.NotNil(t, cmd)

	// Execute the batch command to simulate the async execution
	// The batch command contains two functions: start message and execution
	// We'll simulate both by calling the Update method with the expected messages

	// Process start message (first part of batch)
	updatedModel, _ = viewModel.Update(DiagnosticStartMsg{})
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.Equal(t, DiagnosticStateLoading, viewModel.GetState())
	assert.True(t, viewModel.IsLoading())

	// Process result message (second part of batch - this will call Execute)
	// We need to manually trigger the execution since we can't easily execute the tea.Cmd
	result, err := mockTool.Execute(context.Background(), domain.NewWHOISParameters("example.com"))
	assert.NoError(t, err)
	assert.NotNil(t, result)

	updatedModel, _ = viewModel.Update(DiagnosticResultMsg{Result: result})
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.Equal(t, DiagnosticStateResult, viewModel.GetState())
	assert.False(t, viewModel.IsLoading())
	assert.NotNil(t, viewModel.GetResult())

	// Test that the view renders correctly (this will trigger Export for raw mode)
	view := viewModel.View()
	assert.Contains(t, view, "example.com")

	// Test switching to raw mode to trigger Export
	viewModel.resultView.mode = ResultViewModeRaw
	rawView := viewModel.resultView.View()
	assert.Contains(t, rawView, "example.com")

	mockTool.AssertExpectations(t)
	mockResult.AssertExpectations(t)
}

func TestDiagnosticViewModel_ErrorHandling(t *testing.T) {
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")
	mockTool.On("Execute", mock.Anything, mock.Anything).Return((*MockResult)(nil), assert.AnError)

	viewModel := NewDiagnosticViewModel(mockTool)

	// Test error handling
	formValues := map[string]string{
		"query": "invalid-domain",
	}

	// Process form submission
	updatedModel, cmd := viewModel.Update(FormSubmitMsg{Values: formValues})
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.NotNil(t, cmd)

	// Process start message
	updatedModel, _ = viewModel.Update(DiagnosticStartMsg{})
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.Equal(t, DiagnosticStateLoading, viewModel.GetState())

	// Manually trigger the execution to test the error path
	_, err := mockTool.Execute(context.Background(), domain.NewWHOISParameters("invalid-domain"))
	assert.Error(t, err)

	// Process error message
	updatedModel, _ = viewModel.Update(DiagnosticErrorMsg{Error: err})
	viewModel = updatedModel.(*DiagnosticViewModel)

	assert.Equal(t, DiagnosticStateError, viewModel.GetState())
	assert.False(t, viewModel.IsLoading())
	assert.Equal(t, err, viewModel.GetError())

	mockTool.AssertExpectations(t)
}

func TestDiagnosticViewModel_KeyboardNavigation(t *testing.T) {
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")

	viewModel := NewDiagnosticViewModel(mockTool)
	viewModel.SetSize(80, 24)

	// Test escape key navigation
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := viewModel.Update(keyMsg)
	viewModel = updatedModel.(*DiagnosticViewModel)

	// Should send navigation back message
	assert.NotNil(t, cmd)

	// Test quit key
	keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = viewModel.Update(keyMsg)

	// Should return quit command
	assert.NotNil(t, cmd)

	mockTool.AssertExpectations(t)
}

func TestDiagnosticViewModel_ResponsiveLayout(t *testing.T) {
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")

	viewModel := NewDiagnosticViewModel(mockTool)

	// Test different screen sizes
	sizes := []struct{ width, height int }{
		{60, 20},   // Small
		{100, 30},  // Medium
		{140, 40},  // Large
	}

	for _, size := range sizes {
		viewModel.SetSize(size.width, size.height)
		assert.Equal(t, size.width, viewModel.width)
		assert.Equal(t, size.height, viewModel.height)

		// Test that view renders without errors
		view := viewModel.View()
		assert.NotEmpty(t, view)
	}

	mockTool.AssertExpectations(t)
}

func TestDiagnosticViewModel_TUIComponentInterface(t *testing.T) {
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")

	viewModel := NewDiagnosticViewModel(mockTool)

	// Test TUIComponent interface methods
	viewModel.SetSize(100, 30)
	assert.Equal(t, 100, viewModel.width)
	assert.Equal(t, 30, viewModel.height)

	// Test theme setting (no panic)
	mockTheme := &MockDiagnosticTheme{}
	viewModel.SetTheme(mockTheme)

	// Test focus/blur
	viewModel.Focus()
	viewModel.Blur()

	mockTool.AssertExpectations(t)
}

// MockTheme implements domain.Theme for testing
type MockDiagnosticTheme struct {
	mock.Mock
}

func (m *MockDiagnosticTheme) GetColor(element string) string {
	args := m.Called(element)
	return args.String(0)
}

func (m *MockDiagnosticTheme) GetStyle(element string) map[string]interface{} {
	args := m.Called(element)
	return args.Get(0).(map[string]interface{})
}

func (m *MockDiagnosticTheme) SetColor(element, color string) {
	m.Called(element, color)
}

func TestDiagnosticViewModel_FormFieldConfiguration(t *testing.T) {
	tests := []struct {
		toolName      string
		expectedFields []string
	}{
		{
			toolName:      "whois",
			expectedFields: []string{"query"},
		},
		{
			toolName:      "ping",
			expectedFields: []string{"host", "count"},
		},
		{
			toolName:      "dns",
			expectedFields: []string{"domain", "record_type"},
		},
		{
			toolName:      "ssl",
			expectedFields: []string{"host", "port"},
		},
		{
			toolName:      "traceroute",
			expectedFields: []string{"host", "max_hops"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			mockTool := &MockDiagnosticTool{}
			mockTool.On("Name").Return(tt.toolName)
			mockTool.On("Description").Return("Test description")

			viewModel := NewDiagnosticViewModel(mockTool)

			// Verify form has expected fields
			assert.NotNil(t, viewModel.inputForm)
			assert.Equal(t, len(tt.expectedFields), len(viewModel.inputForm.fields))

			for i, expectedField := range tt.expectedFields {
				assert.Equal(t, expectedField, viewModel.inputForm.fields[i].Key)
			}

			mockTool.AssertExpectations(t)
		})
	}
}

func TestDiagnosticViewModel_StateTransitions(t *testing.T) {
	mockTool := &MockDiagnosticTool{}
	mockTool.On("Name").Return("whois")
	mockTool.On("Description").Return("WHOIS lookup tool")

	viewModel := NewDiagnosticViewModel(mockTool)

	// Test state transitions
	assert.Equal(t, DiagnosticStateInput, viewModel.GetState())

	// Transition to loading
	startMsg := DiagnosticStartMsg{}
	updatedModel, _ := viewModel.Update(startMsg)
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.Equal(t, DiagnosticStateLoading, viewModel.GetState())

	// Transition to result
	mockResult := &MockResult{
		data:     domain.WHOISResult{Domain: "example.com"},
		metadata: map[string]interface{}{},
	}
	resultMsg := DiagnosticResultMsg{Result: mockResult}
	updatedModel, _ = viewModel.Update(resultMsg)
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.Equal(t, DiagnosticStateResult, viewModel.GetState())

	// Transition back to input with escape
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = viewModel.Update(keyMsg)
	viewModel = updatedModel.(*DiagnosticViewModel)
	assert.Equal(t, DiagnosticStateInput, viewModel.GetState())

	mockTool.AssertExpectations(t)
}