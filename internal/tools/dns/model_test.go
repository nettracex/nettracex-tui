// Package dns provides TUI model tests for DNS diagnostic tool
package dns

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)

func TestNewModel(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	
	model := NewModel(tool)
	
	if model == nil {
		t.Fatal("NewModel returned nil")
	}
	
	if model.tool != tool {
		t.Error("Model tool not set correctly")
	}
	
	if model.state != StateInput {
		t.Errorf("Expected initial state StateInput, got %v", model.state)
	}
	
	if model.loading {
		t.Error("Model should not be loading initially")
	}
	
	// Check default record type selections
	expectedTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}
	
	for _, recordType := range expectedTypes {
		if !model.selectedTypes[recordType] {
			t.Errorf("Expected record type %v to be selected by default", recordType)
		}
	}
}

func TestModel_Init(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	cmd := model.Init()
	
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestModel_Update_KeyboardNavigation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	tests := []struct {
		name           string
		initialState   ModelState
		key            string
		expectedState  ModelState
		expectedAction string
	}{
		{
			name:           "quit with ctrl+c",
			initialState:   StateInput,
			key:            "ctrl+c",
			expectedAction: "quit",
		},
		{
			name:           "quit with q",
			initialState:   StateInput,
			key:            "q",
			expectedAction: "quit",
		},
		{
			name:          "tab to type selection",
			initialState:  StateInput,
			key:           "tab",
			expectedState: StateTypeSelection,
		},
		{
			name:          "escape from type selection",
			initialState:  StateTypeSelection,
			key:           "esc",
			expectedState: StateInput,
		},
		{
			name:          "escape from result to input",
			initialState:  StateResult,
			key:           "esc",
			expectedState: StateInput,
		},
		{
			name:          "escape from error to input",
			initialState:  StateError,
			key:           "esc",
			expectedState: StateInput,
		},
		{
			name:          "enter from type selection to input",
			initialState:  StateTypeSelection,
			key:           "enter",
			expectedState: StateInput,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.state = tt.initialState
			if tt.initialState == StateTypeSelection {
				model.showTypeSelect = true
			}
			
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tt.key == "tab" {
				keyMsg = tea.KeyMsg{Type: tea.KeyTab}
			} else if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else if tt.key == "enter" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			} else if tt.key == "up" {
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			} else if tt.key == "down" {
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			} else if tt.key == " " {
				keyMsg = tea.KeyMsg{Type: tea.KeySpace}
			}
			
			newModel, cmd := model.Update(keyMsg)
			updatedModel := newModel.(*Model)
			
			if tt.expectedAction == "quit" {
				if cmd == nil {
					t.Error("Expected quit command but got nil")
				}
			} else if tt.expectedState != 0 {
				if updatedModel.state != tt.expectedState {
					t.Errorf("Expected state %v, got %v", tt.expectedState, updatedModel.state)
				}
			}
		})
	}
}

func TestModel_Update_TypeSelection(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	model.state = StateTypeSelection
	model.typeSelection = 0
	
	// Test navigation down
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(keyMsg)
	updatedModel := newModel.(*Model)
	
	if updatedModel.typeSelection != 1 {
		t.Errorf("Expected typeSelection 1, got %d", updatedModel.typeSelection)
	}
	
	// Test navigation up
	keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = updatedModel.Update(keyMsg)
	updatedModel = newModel.(*Model)
	
	if updatedModel.typeSelection != 0 {
		t.Errorf("Expected typeSelection 0, got %d", updatedModel.typeSelection)
	}
	
	// Test toggle selection
	initialSelection := updatedModel.selectedTypes[domain.DNSRecordTypeA]
	keyMsg = tea.KeyMsg{Type: tea.KeySpace}
	newModel, _ = updatedModel.Update(keyMsg)
	updatedModel = newModel.(*Model)
	
	if updatedModel.selectedTypes[domain.DNSRecordTypeA] == initialSelection {
		t.Error("Expected record type selection to be toggled")
	}
}

func TestModel_Update_AsyncMessages(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test lookup start message
	startMsg := lookupStartMsg{}
	newModel, _ := model.Update(startMsg)
	updatedModel := newModel.(*Model)
	
	if updatedModel.state != StateLoading {
		t.Errorf("Expected state StateLoading, got %v", updatedModel.state)
	}
	
	if !updatedModel.loading {
		t.Error("Expected loading to be true")
	}
	
	// Test lookup result message
	resultMsg := lookupResultMsg{
		result: domain.DNSResult{
			Query: "example.com",
			Records: []domain.DNSRecord{
				{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
			},
		},
	}
	newModel, _ = updatedModel.Update(resultMsg)
	updatedModel = newModel.(*Model)
	
	if updatedModel.state != StateResult {
		t.Errorf("Expected state StateResult, got %v", updatedModel.state)
	}
	
	if updatedModel.loading {
		t.Error("Expected loading to be false")
	}
	
	if updatedModel.result.Query != "example.com" {
		t.Errorf("Expected result query 'example.com', got '%s'", updatedModel.result.Query)
	}
	
	// Test lookup error message
	model.state = StateLoading
	errorMsg := lookupErrorMsg{
		error: &domain.NetTraceError{
			Message: "DNS lookup failed",
		},
	}
	newModel, _ = model.Update(errorMsg)
	updatedModel = newModel.(*Model)
	
	if updatedModel.state != StateError {
		t.Errorf("Expected state StateError, got %v", updatedModel.state)
	}
	
	if updatedModel.loading {
		t.Error("Expected loading to be false")
	}
	
	if updatedModel.error == nil {
		t.Error("Expected error to be set")
	}
}

func TestModel_View(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test input state view
	model.state = StateInput
	view := model.View()
	
	if !strings.Contains(view, "DNS Lookup Tool") {
		t.Error("View should contain tool title")
	}
	
	if !strings.Contains(view, "Domain:") {
		t.Error("View should contain domain input label")
	}
	
	// Test type selection view
	model.state = StateTypeSelection
	model.showTypeSelect = true
	view = model.View()
	
	if !strings.Contains(view, "Record Types:") {
		t.Error("View should contain record types section")
	}
	
	if !strings.Contains(view, "A - IPv4 address records") {
		t.Error("View should contain A record description")
	}
	
	// Test loading state view
	model.state = StateLoading
	model.input.SetValue("example.com")
	view = model.View()
	
	if !strings.Contains(view, "Performing DNS lookups") {
		t.Error("View should contain loading message")
	}
	
	if !strings.Contains(view, "example.com") {
		t.Error("View should contain domain being looked up")
	}
	
	// Test result state view
	model.state = StateResult
	model.result = domain.DNSResult{
		Query:  "example.com",
		Server: "system",
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
		},
	}
	model.buildResultTabs()
	model.SetSize(80, 24)
	model.calculateMaxScroll()
	view = model.View()
	
	if !strings.Contains(view, "Query Information") {
		t.Error("View should contain query information section")
	}
	
	if !strings.Contains(view, "example.com") {
		t.Error("View should contain query domain")
	}
	
	if !strings.Contains(view, "93.184.216.34") {
		t.Error("View should contain DNS record value")
	}
	
	// Test error state view
	model.state = StateError
	model.error = &domain.NetTraceError{
		Message: "DNS lookup failed",
	}
	view = model.View()
	
	if !strings.Contains(view, "Error:") {
		t.Error("View should contain error message")
	}
	
	if !strings.Contains(view, "DNS lookup failed") {
		t.Error("View should contain specific error message")
	}
}

func TestModel_SetSize(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	width, height := 100, 50
	model.SetSize(width, height)
	
	if model.width != width {
		t.Errorf("Expected width %d, got %d", width, model.width)
	}
	
	if model.height != height {
		t.Errorf("Expected height %d, got %d", height, model.height)
	}
	
	expectedInputWidth := width - 4
	if model.input.Width != expectedInputWidth {
		t.Errorf("Expected input width %d, got %d", expectedInputWidth, model.input.Width)
	}
}

func TestModel_SetTheme(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Create a mock theme
	mockTheme := &MockTheme{}
	model.SetTheme(mockTheme)
	
	if model.theme != mockTheme {
		t.Error("Theme not set correctly")
	}
}

func TestModel_Focus(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	model.state = StateInput
	model.Focus()
	
	// Note: We can't easily test if the input is actually focused
	// without access to internal state, but we can verify the method doesn't panic
}

func TestModel_Blur(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	model.Blur()
	
	// Note: We can't easily test if the input is actually blurred
	// without access to internal state, but we can verify the method doesn't panic
}

func TestModel_GetRecordTypeByIndex(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	tests := []struct {
		index    int
		expected domain.DNSRecordType
	}{
		{0, domain.DNSRecordTypeA},
		{1, domain.DNSRecordTypeAAAA},
		{2, domain.DNSRecordTypeMX},
		{3, domain.DNSRecordTypeTXT},
		{4, domain.DNSRecordTypeCNAME},
		{5, domain.DNSRecordTypeNS},
		{-1, domain.DNSRecordTypeA}, // Invalid index should return default
		{10, domain.DNSRecordTypeA}, // Invalid index should return default
	}
	
	for _, tt := range tests {
		t.Run(string(rune(tt.index)), func(t *testing.T) {
			result := model.getRecordTypeByIndex(tt.index)
			if result != tt.expected {
				t.Errorf("getRecordTypeByIndex(%d) = %v, want %v", tt.index, result, tt.expected)
			}
		})
	}
}

func TestModel_GetRecordTypeDescription(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	tests := []struct {
		recordType  domain.DNSRecordType
		expectedKey string
	}{
		{domain.DNSRecordTypeA, "IPv4 address"},
		{domain.DNSRecordTypeAAAA, "IPv6 address"},
		{domain.DNSRecordTypeMX, "Mail exchange"},
		{domain.DNSRecordTypeTXT, "Text records"},
		{domain.DNSRecordTypeCNAME, "Canonical name"},
		{domain.DNSRecordTypeNS, "Name server"},
		{domain.DNSRecordType(999), "Unknown record type"},
	}
	
	for _, tt := range tests {
		t.Run(GetRecordTypeString(tt.recordType), func(t *testing.T) {
			result := model.getRecordTypeDescription(tt.recordType)
			if !strings.Contains(result, tt.expectedKey) {
				t.Errorf("getRecordTypeDescription(%v) should contain '%s', got '%s'", tt.recordType, tt.expectedKey, result)
			}
		})
	}
}

func TestModel_PerformLookup(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Set up input
	model.input.SetValue("example.com")
	
	// Set up mock response
	mockClient.SetDNSResponse("example.com", domain.DNSRecordTypeA, domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeA,
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
		},
	})
	
	cmd := model.performLookup()
	
	if cmd == nil {
		t.Error("performLookup should return a command")
	}
	
	// Note: Testing the actual execution of the command would require
	// more complex setup and is better covered by integration tests
}

// MockTheme implements domain.Theme for testing
type MockTheme struct{}

func (m *MockTheme) GetColor(element string) string {
	return "#ffffff"
}

func (m *MockTheme) GetStyle(element string) map[string]interface{} {
	return map[string]interface{}{}
}

func (m *MockTheme) SetColor(element, color string) {
	// Mock implementation
}