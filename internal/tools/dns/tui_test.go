// Package dns provides TUI interaction tests for DNS diagnostic tool
package dns

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)

// TUITestHarness provides utilities for testing TUI interactions
type TUITestHarness struct {
	model   *Model
	client  *network.MockClient
	logger  *MockLogger
	tool    *Tool
}

// NewTUITestHarness creates a new test harness for TUI testing
func NewTUITestHarness() *TUITestHarness {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	return &TUITestHarness{
		model:  model,
		client: mockClient,
		logger: mockLogger,
		tool:   tool,
	}
}

// SendKey simulates sending a key press to the model
func (h *TUITestHarness) SendKey(key string) tea.Cmd {
	var keyMsg tea.KeyMsg
	
	switch key {
	case "enter":
		keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	case " ":
		keyMsg = tea.KeyMsg{Type: tea.KeySpace}
	case "ctrl+c":
		keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
	case "q":
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	default:
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	
	newModel, cmd := h.model.Update(keyMsg)
	h.model = newModel.(*Model)
	return cmd
}

// SendText simulates typing text into the input field
func (h *TUITestHarness) SendText(text string) {
	for _, char := range text {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := h.model.Update(keyMsg)
		h.model = newModel.(*Model)
	}
}

// GetView returns the current view of the model
func (h *TUITestHarness) GetView() string {
	return h.model.View()
}

// GetState returns the current state of the model
func (h *TUITestHarness) GetState() ModelState {
	return h.model.state
}

// SetupMockResponse sets up a mock DNS response
func (h *TUITestHarness) SetupMockResponse(domain string, recordType domain.DNSRecordType, result domain.DNSResult) {
	h.client.SetDNSResponse(domain, recordType, result)
}

// BuildResultTabs builds result tabs for testing
func (h *TUITestHarness) BuildResultTabs() {
	h.model.buildResultTabs()
}

func TestDNSTUI_BasicNavigation(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Test initial state
	if harness.GetState() != StateInput {
		t.Errorf("Expected initial state StateInput, got %v", harness.GetState())
	}
	
	view := harness.GetView()
	if !strings.Contains(view, "DNS Lookup Tool") {
		t.Error("View should contain tool title")
	}
	
	if !strings.Contains(view, "Domain:") {
		t.Error("View should contain domain input label")
	}
	
	// Test navigation to type selection
	harness.SendKey("tab")
	if harness.GetState() != StateTypeSelection {
		t.Errorf("Expected state StateTypeSelection after tab, got %v", harness.GetState())
	}
	
	view = harness.GetView()
	if !strings.Contains(view, "Record Types:") {
		t.Error("View should contain record types section")
	}
	
	// Test navigation back to input
	harness.SendKey("esc")
	if harness.GetState() != StateInput {
		t.Errorf("Expected state StateInput after escape, got %v", harness.GetState())
	}
}

func TestDNSTUI_TypeSelection(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Navigate to type selection
	harness.SendKey("tab")
	if harness.GetState() != StateTypeSelection {
		t.Errorf("Expected state StateTypeSelection, got %v", harness.GetState())
	}
	
	// Test navigation within type selection
	initialSelection := harness.model.typeSelection
	harness.SendKey("down")
	if harness.model.typeSelection != initialSelection+1 {
		t.Errorf("Expected typeSelection to increase, got %d", harness.model.typeSelection)
	}
	
	harness.SendKey("up")
	if harness.model.typeSelection != initialSelection {
		t.Errorf("Expected typeSelection to return to initial value, got %d", harness.model.typeSelection)
	}
	
	// Test toggling selection
	recordType := harness.model.getRecordTypeByIndex(harness.model.typeSelection)
	initialSelected := harness.model.selectedTypes[recordType]
	
	harness.SendKey(" ")
	if harness.model.selectedTypes[recordType] == initialSelected {
		t.Error("Expected record type selection to be toggled")
	}
	
	// Toggle back
	harness.SendKey(" ")
	if harness.model.selectedTypes[recordType] != initialSelected {
		t.Error("Expected record type selection to be toggled back")
	}
}

func TestDNSTUI_DNSLookupFlow(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Set up mock response
	mockResult := domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeA,
		Records: []domain.DNSRecord{
			{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeA,
				Value: "93.184.216.34",
				TTL:   300,
			},
		},
		ResponseTime: 50 * time.Millisecond,
		Server:       "system",
	}
	
	harness.SetupMockResponse("example.com", domain.DNSRecordTypeA, mockResult)
	harness.SetupMockResponse("example.com", domain.DNSRecordTypeAAAA, domain.DNSResult{
		Query:      "example.com",
		RecordType: domain.DNSRecordTypeAAAA,
		Records:    []domain.DNSRecord{},
		ResponseTime: 45 * time.Millisecond,
		Server:       "system",
	})
	
	// Enter domain name
	harness.SendText("example.com")
	
	// Verify input is captured
	if harness.model.input.Value() != "example.com" {
		t.Errorf("Expected input 'example.com', got '%s'", harness.model.input.Value())
	}
	
	// Start lookup
	cmd := harness.SendKey("enter")
	if cmd == nil {
		t.Error("Expected command from enter key")
	}
	
	// Simulate lookup start message
	startMsg := lookupStartMsg{}
	newModel, _ := harness.model.Update(startMsg)
	harness.model = newModel.(*Model)
	
	if harness.GetState() != StateLoading {
		t.Errorf("Expected state StateLoading, got %v", harness.GetState())
	}
	
	view := harness.GetView()
	if !strings.Contains(view, "Performing DNS lookups") {
		t.Error("Loading view should contain lookup message")
	}
	
	if !strings.Contains(view, "example.com") {
		t.Error("Loading view should contain domain name")
	}
	
	// Simulate lookup result message
	resultMsg := lookupResultMsg{result: mockResult}
	newModel, _ = harness.model.Update(resultMsg)
	harness.model = newModel.(*Model)
	
	if harness.GetState() != StateResult {
		t.Errorf("Expected state StateResult, got %v", harness.GetState())
	}
	
	view = harness.GetView()
	if !strings.Contains(view, "Query Information") {
		t.Error("Result view should contain query information")
	}
	
	if !strings.Contains(view, "example.com") {
		t.Error("Result view should contain domain name")
	}
	
	// With the new tabbed interface, the IP should be in the A Records tab
	if !strings.Contains(view, "93.184.216.34") {
		t.Errorf("Result view should contain IP address. View content:\n%s", view)
	}
}

func TestDNSTUI_ErrorHandling(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Enter invalid domain
	harness.SendText("invalid..domain")
	
	// Start lookup (should fail validation)
	harness.SendKey("enter")
	
	// Simulate error message
	errorMsg := lookupErrorMsg{
		error: &domain.NetTraceError{
			Type:    domain.ErrorTypeValidation,
			Message: "invalid domain format",
			Code:    "DNS_VALIDATION_FAILED",
		},
	}
	
	newModel, _ := harness.model.Update(errorMsg)
	harness.model = newModel.(*Model)
	
	if harness.GetState() != StateError {
		t.Errorf("Expected state StateError, got %v", harness.GetState())
	}
	
	view := harness.GetView()
	if !strings.Contains(view, "Error:") {
		t.Error("Error view should contain error message")
	}
	
	if !strings.Contains(view, "invalid domain format") {
		t.Error("Error view should contain specific error message")
	}
	
	// Test recovery from error
	harness.SendKey("esc")
	if harness.GetState() != StateInput {
		t.Errorf("Expected state StateInput after escape from error, got %v", harness.GetState())
	}
	
	// Input should be cleared
	if harness.model.input.Value() != "" {
		t.Errorf("Expected input to be cleared, got '%s'", harness.model.input.Value())
	}
}

func TestDNSTUI_KeyboardShortcuts(t *testing.T) {
	harness := NewTUITestHarness()
	
	tests := []struct {
		name          string
		initialState  ModelState
		key           string
		expectedState ModelState
		expectQuit    bool
	}{
		{
			name:         "quit with q from input",
			initialState: StateInput,
			key:          "q",
			expectQuit:   true,
		},
		{
			name:         "quit with ctrl+c from input",
			initialState: StateInput,
			key:          "ctrl+c",
			expectQuit:   true,
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
			name:          "enter from type selection",
			initialState:  StateTypeSelection,
			key:           "enter",
			expectedState: StateInput,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset harness
			harness = NewTUITestHarness()
			harness.model.state = tt.initialState
			
			if tt.initialState == StateTypeSelection {
				harness.model.showTypeSelect = true
			}
			
			cmd := harness.SendKey(tt.key)
			
			if tt.expectQuit {
				if cmd == nil {
					t.Error("Expected quit command but got nil")
				}
				// Note: We can't easily test tea.Quit without more complex setup
			} else if tt.expectedState != 0 {
				if harness.GetState() != tt.expectedState {
					t.Errorf("Expected state %v, got %v", tt.expectedState, harness.GetState())
				}
			}
		})
	}
}

func TestDNSTUI_ViewContent(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Test input state view content
	harness.model.state = StateInput
	view := harness.GetView()
	
	expectedContent := []string{
		"DNS Lookup Tool",
		"Query DNS records for domains",
		"Domain:",
		"Enter a domain name",
		"enter: lookup",
		"tab: select record types",
		"q: quit",
	}
	
	for _, content := range expectedContent {
		if !strings.Contains(view, content) {
			t.Errorf("Input view should contain '%s'", content)
		}
	}
	
	// Test type selection view content
	harness.model.state = StateTypeSelection
	harness.model.showTypeSelect = true
	view = harness.GetView()
	
	typeSelectionContent := []string{
		"Record Types:",
		"A - IPv4 address records",
		"AAAA - IPv6 address records",
		"MX - Mail exchange records",
		"TXT - Text records",
		"CNAME - Canonical name records",
		"NS - Name server records",
		"↑/↓ to navigate",
		"space to toggle",
		"enter to confirm",
	}
	
	for _, content := range typeSelectionContent {
		if !strings.Contains(view, content) {
			t.Errorf("Type selection view should contain '%s'", content)
		}
	}
	
	// Test result view content
	harness.model.state = StateResult
	harness.model.result = domain.DNSResult{
		Query:  "example.com",
		Server: "system",
		Records: []domain.DNSRecord{
			{
				Name:  "example.com",
				Type:  domain.DNSRecordTypeA,
				Value: "93.184.216.34",
				TTL:   300,
			},
			{
				Name:     "example.com",
				Type:     domain.DNSRecordTypeMX,
				Value:    "mail.example.com",
				TTL:      300,
				Priority: 10,
			},
		},
		ResponseTime: 50 * time.Millisecond,
	}
	harness.model.buildResultTabs()
	harness.model.SetSize(80, 24)
	harness.model.calculateMaxScroll()
	view = harness.GetView()
	
	resultContent := []string{
		"Query Information",
		"example.com",
		"system",
		"A (1)",  // Tab format instead of "A Records:"
		"93.184.216.34",
		"esc: new lookup",
	}
	
	for _, content := range resultContent {
		if !strings.Contains(view, content) {
			t.Errorf("Result view should contain '%s'", content)
		}
	}
	
	// Test switching to MX tab
	harness.SendKey("right")
	view = harness.GetView()
	
	mxContent := []string{
		"MX (1)",  // Tab format
		"mail.example.com",
		"Priority: 10",
	}
	
	for _, content := range mxContent {
		if !strings.Contains(view, content) {
			t.Errorf("MX tab view should contain '%s'", content)
		}
	}
}

func TestDNSTUI_ResponsiveLayout(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Test different screen sizes
	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},   // Standard terminal
		{120, 30},  // Wide terminal
		{60, 20},   // Narrow terminal
		{200, 50},  // Very wide terminal
	}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%dx%d", size.width, size.height), func(t *testing.T) {
			harness.model.SetSize(size.width, size.height)
			
			if harness.model.width != size.width {
				t.Errorf("Expected width %d, got %d", size.width, harness.model.width)
			}
			
			if harness.model.height != size.height {
				t.Errorf("Expected height %d, got %d", size.height, harness.model.height)
			}
			
			// Verify input width is adjusted
			expectedInputWidth := size.width - 4
			if harness.model.input.Width != expectedInputWidth {
				t.Errorf("Expected input width %d, got %d", expectedInputWidth, harness.model.input.Width)
			}
			
			// Verify view renders without panic
			view := harness.GetView()
			if view == "" {
				t.Error("View should not be empty")
			}
		})
	}
}

func TestDNSTUI_MultipleRecordTypes(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Set up mock responses for multiple record types
	recordTypes := []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
	}
	
	for _, recordType := range recordTypes {
		mockResult := domain.DNSResult{
			Query:      "example.com",
			RecordType: recordType,
			Records: []domain.DNSRecord{
				{
					Name:  "example.com",
					Type:  recordType,
					Value: fmt.Sprintf("value-for-%s", GetRecordTypeString(recordType)),
					TTL:   300,
				},
			},
			ResponseTime: 50 * time.Millisecond,
			Server:       "system",
		}
		harness.SetupMockResponse("example.com", recordType, mockResult)
	}
	
	// Navigate to type selection and select specific types
	harness.SendKey("tab")
	
	// Deselect all types first
	for i := 0; i < 6; i++ {
		harness.model.typeSelection = i
		recordType := harness.model.getRecordTypeByIndex(i)
		if harness.model.selectedTypes[recordType] {
			harness.SendKey(" ") // Toggle off
		}
	}
	
	// Select only the types we want
	for i, recordType := range recordTypes {
		harness.model.typeSelection = i
		if !harness.model.selectedTypes[recordType] {
			harness.SendKey(" ") // Toggle on
		}
	}
	
	// Return to input and perform lookup
	harness.SendKey("enter")
	harness.SendText("example.com")
	harness.SendKey("enter")
	
	// Simulate successful lookup with consolidated results
	consolidatedResult := domain.DNSResult{
		Query:  "example.com",
		Server: "system",
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
			{Name: "example.com", Type: domain.DNSRecordTypeAAAA, Value: "2606:2800:220:1:248:1893:25c8:1946", TTL: 300},
			{Name: "example.com", Type: domain.DNSRecordTypeMX, Value: "mail.example.com", TTL: 300, Priority: 10},
		},
		ResponseTime: 50 * time.Millisecond,
	}
	
	resultMsg := lookupResultMsg{result: consolidatedResult}
	newModel, _ := harness.model.Update(resultMsg)
	harness.model = newModel.(*Model)
	
	view := harness.GetView()
	
	// Verify tabs are created
	if len(harness.model.resultTabs) != 3 {
		t.Errorf("Expected 3 result tabs, got %d", len(harness.model.resultTabs))
	}
	
	// Verify tab names and record counts
	expectedTabs := map[string]int{
		"A":    1,
		"AAAA": 1,
		"MX":   1,
	}
	
	for _, tab := range harness.model.resultTabs {
		expectedCount, exists := expectedTabs[tab.Name]
		if !exists {
			t.Errorf("Unexpected tab: %s", tab.Name)
		}
		if len(tab.Records) != expectedCount {
			t.Errorf("Tab %s should have %d records, got %d", tab.Name, expectedCount, len(tab.Records))
		}
	}
	
	// Verify first tab is active and displayed
	if !strings.Contains(view, "A Records (1)") {
		t.Error("View should contain A Records tab")
	}
	
	if !strings.Contains(view, "93.184.216.34") {
		t.Error("View should contain A record value")
	}
}

func TestDNSTUI_TabbedNavigation(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Set up result with multiple record types
	consolidatedResult := domain.DNSResult{
		Query:  "example.com",
		Server: "system",
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
			{Name: "example.com", Type: domain.DNSRecordTypeAAAA, Value: "2606:2800:220:1:248:1893:25c8:1946", TTL: 300},
			{Name: "example.com", Type: domain.DNSRecordTypeMX, Value: "mail.example.com", TTL: 300, Priority: 10},
		},
		ResponseTime: 50 * time.Millisecond,
	}
	
	// Set model to result state with data
	harness.model.state = StateResult
	harness.model.result = consolidatedResult
	harness.model.buildResultTabs()
	harness.model.SetSize(80, 24)
	harness.model.calculateMaxScroll()
	
	// Test initial state - should be on first tab (A)
	if harness.model.resultTab != 0 {
		t.Errorf("Expected initial tab to be 0, got %d", harness.model.resultTab)
	}
	
	view := harness.GetView()
	if !strings.Contains(view, "A Records (1)") {
		t.Error("View should show A Records tab as active")
	}
	
	// Test navigation to next tab
	harness.SendKey("right")
	if harness.model.resultTab != 1 {
		t.Errorf("Expected tab to be 1 after right arrow, got %d", harness.model.resultTab)
	}
	
	view = harness.GetView()
	if !strings.Contains(view, "AAAA Records (1)") {
		t.Error("View should show AAAA Records tab as active")
	}
	
	// Test navigation to next tab
	harness.SendKey("right")
	if harness.model.resultTab != 2 {
		t.Errorf("Expected tab to be 2 after right arrow, got %d", harness.model.resultTab)
	}
	
	view = harness.GetView()
	if !strings.Contains(view, "MX Records (1)") {
		t.Error("View should show MX Records tab as active")
	}
	
	// Test navigation past last tab (should stay at last)
	harness.SendKey("right")
	if harness.model.resultTab != 2 {
		t.Errorf("Expected tab to stay at 2 when at end, got %d", harness.model.resultTab)
	}
	
	// Test navigation back
	harness.SendKey("left")
	if harness.model.resultTab != 1 {
		t.Errorf("Expected tab to be 1 after left arrow, got %d", harness.model.resultTab)
	}
	
	// Test navigation to first tab
	harness.SendKey("left")
	if harness.model.resultTab != 0 {
		t.Errorf("Expected tab to be 0 after left arrow, got %d", harness.model.resultTab)
	}
	
	// Test navigation past first tab (should stay at first)
	harness.SendKey("left")
	if harness.model.resultTab != 0 {
		t.Errorf("Expected tab to stay at 0 when at beginning, got %d", harness.model.resultTab)
	}
}

func TestDNSTUI_ScrollingInResults(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Create many records to test scrolling
	var records []domain.DNSRecord
	for i := 0; i < 20; i++ {
		records = append(records, domain.DNSRecord{
			Name:  fmt.Sprintf("record%d.example.com", i),
			Type:  domain.DNSRecordTypeA,
			Value: fmt.Sprintf("192.168.1.%d", i+1),
			TTL:   300,
		})
	}
	
	consolidatedResult := domain.DNSResult{
		Query:        "example.com",
		Server:       "system",
		Records:      records,
		ResponseTime: 50 * time.Millisecond,
	}
	
	// Set model to result state with data
	harness.model.state = StateResult
	harness.model.result = consolidatedResult
	harness.model.buildResultTabs()
	harness.model.SetSize(80, 15) // Small height to force scrolling
	harness.model.calculateMaxScroll()
	
	// Test initial scroll position
	if harness.model.scrollOffset != 0 {
		t.Errorf("Expected initial scroll offset to be 0, got %d", harness.model.scrollOffset)
	}
	
	// Test scrolling down
	initialOffset := harness.model.scrollOffset
	harness.SendKey("down")
	if harness.model.scrollOffset <= initialOffset {
		t.Error("Expected scroll offset to increase after down arrow")
	}
	
	// Test scrolling up
	currentOffset := harness.model.scrollOffset
	harness.SendKey("up")
	if harness.model.scrollOffset >= currentOffset {
		t.Error("Expected scroll offset to decrease after up arrow")
	}
	
	// Test scrolling to maximum
	for i := 0; i < 50; i++ { // Scroll more than needed
		harness.SendKey("down")
	}
	
	if harness.model.scrollOffset > harness.model.maxScroll {
		t.Errorf("Scroll offset should not exceed maxScroll: %d > %d", 
			harness.model.scrollOffset, harness.model.maxScroll)
	}
	
	// Test scrolling to minimum
	for i := 0; i < 50; i++ { // Scroll more than needed
		harness.SendKey("up")
	}
	
	if harness.model.scrollOffset < 0 {
		t.Errorf("Scroll offset should not be negative: %d", harness.model.scrollOffset)
	}
}

func TestDNSTUI_SingleRecordTypeDisplay(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Set up result with single record type
	consolidatedResult := domain.DNSResult{
		Query:  "example.com",
		Server: "system",
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
			{Name: "www.example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.35", TTL: 300},
		},
		ResponseTime: 50 * time.Millisecond,
	}
	
	// Set model to result state with data
	harness.model.state = StateResult
	harness.model.result = consolidatedResult
	harness.model.buildResultTabs()
	
	// Should have only one tab
	if len(harness.model.resultTabs) != 1 {
		t.Errorf("Expected 1 result tab, got %d", len(harness.model.resultTabs))
	}
	
	view := harness.GetView()
	
	// Should not show tab navigation for single tab
	if strings.Contains(view, "←/→: switch tabs") {
		t.Error("Should not show tab navigation help for single tab")
	}
	
	// Should show records directly
	if !strings.Contains(view, "A Records (2)") {
		t.Error("View should show A Records section")
	}
	
	if !strings.Contains(view, "93.184.216.34") {
		t.Error("View should contain first A record")
	}
	
	if !strings.Contains(view, "93.184.216.35") {
		t.Error("View should contain second A record")
	}
}

func TestDNSTUI_EmptyResultsDisplay(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Set up result with no records
	consolidatedResult := domain.DNSResult{
		Query:        "nonexistent.example.com",
		Server:       "system",
		Records:      []domain.DNSRecord{},
		ResponseTime: 50 * time.Millisecond,
	}
	
	// Set model to result state with empty data
	harness.model.state = StateResult
	harness.model.result = consolidatedResult
	harness.model.buildResultTabs()
	
	// Should have no tabs
	if len(harness.model.resultTabs) != 0 {
		t.Errorf("Expected 0 result tabs for empty result, got %d", len(harness.model.resultTabs))
	}
	
	view := harness.GetView()
	
	// Should show appropriate message
	if !strings.Contains(view, "No DNS records found") {
		t.Error("View should show 'No DNS records found' message")
	}
	
	// Should not show tab navigation
	if strings.Contains(view, "←/→: switch tabs") {
		t.Error("Should not show tab navigation for empty results")
	}
}

func TestDNSTUI_ErrorHandlingAndValidation(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Test various error scenarios
	errorTests := []struct {
		name        string
		domain      string
		errorMsg    string
		expectError bool
	}{
		{
			name:        "empty domain",
			domain:      "",
			expectError: false, // Should not trigger lookup
		},
		{
			name:        "invalid domain format",
			domain:      "invalid..domain",
			errorMsg:    "invalid domain format",
			expectError: true,
		},
		{
			name:        "network timeout",
			domain:      "timeout.example.com",
			errorMsg:    "network timeout",
			expectError: true,
		},
		{
			name:        "dns server error",
			domain:      "error.example.com",
			errorMsg:    "DNS server error",
			expectError: true,
		},
	}
	
	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset harness
			harness = NewTUITestHarness()
			
			// Enter domain
			if tt.domain != "" {
				harness.SendText(tt.domain)
			}
			
			// Try to start lookup
			harness.SendKey("enter")
			
			if tt.expectError {
				// Simulate error message
				errorMsg := lookupErrorMsg{
					error: &domain.NetTraceError{
						Type:    domain.ErrorTypeValidation,
						Message: tt.errorMsg,
						Code:    "DNS_ERROR",
					},
				}
				
				newModel, _ := harness.model.Update(errorMsg)
				harness.model = newModel.(*Model)
				
				if harness.GetState() != StateError {
					t.Errorf("Expected state StateError, got %v", harness.GetState())
				}
				
				view := harness.GetView()
				if !strings.Contains(view, "Error:") {
					t.Error("Error view should contain error message")
				}
				
				if !strings.Contains(view, tt.errorMsg) {
					t.Errorf("Error view should contain specific error message: %s", tt.errorMsg)
				}
				
				// Test recovery from error
				harness.SendKey("esc")
				if harness.GetState() != StateInput {
					t.Errorf("Expected state StateInput after escape from error, got %v", harness.GetState())
				}
				
				// Input should be cleared
				if harness.model.input.Value() != "" {
					t.Errorf("Expected input to be cleared, got '%s'", harness.model.input.Value())
				}
			}
		})
	}
}

func TestDNSTUI_ResponsiveLayoutWithTabs(t *testing.T) {
	harness := NewTUITestHarness()
	
	// Set up result with multiple record types
	consolidatedResult := domain.DNSResult{
		Query:  "example.com",
		Server: "system",
		Records: []domain.DNSRecord{
			{Name: "example.com", Type: domain.DNSRecordTypeA, Value: "93.184.216.34", TTL: 300},
			{Name: "example.com", Type: domain.DNSRecordTypeAAAA, Value: "2606:2800:220:1:248:1893:25c8:1946", TTL: 300},
		},
		ResponseTime: 50 * time.Millisecond,
	}
	
	harness.model.state = StateResult
	harness.model.result = consolidatedResult
	harness.model.buildResultTabs()
	
	// Test different screen sizes
	sizes := []struct {
		width  int
		height int
	}{
		{80, 24},   // Standard terminal
		{120, 30},  // Wide terminal
		{60, 15},   // Narrow/short terminal
		{200, 50},  // Very wide terminal
	}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%dx%d", size.width, size.height), func(t *testing.T) {
			harness.model.SetSize(size.width, size.height)
			harness.model.calculateMaxScroll()
			
			// Verify view renders without panic
			view := harness.GetView()
			if view == "" {
				t.Error("View should not be empty")
			}
			
			// Verify tabs are displayed for multi-record results
			if len(harness.model.resultTabs) > 1 {
				if !strings.Contains(view, "A (1)") && !strings.Contains(view, "A Records") {
					t.Error("View should contain A record tab or section")
				}
			}
			
			// Verify scroll calculations are reasonable
			if harness.model.maxScroll < 0 {
				t.Errorf("maxScroll should not be negative: %d", harness.model.maxScroll)
			}
		})
	}
}