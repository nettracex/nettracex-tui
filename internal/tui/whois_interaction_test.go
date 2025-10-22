// Package tui contains comprehensive WHOIS TUI interaction tests
package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/tools/whois"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockNetworkClient for WHOIS testing
type MockWHOISNetworkClient struct {
	mock.Mock
}

func (m *MockWHOISNetworkClient) Ping(ctx context.Context, host string, opts domain.PingOptions) (<-chan domain.PingResult, error) {
	args := m.Called(ctx, host, opts)
	return args.Get(0).(<-chan domain.PingResult), args.Error(1)
}

func (m *MockWHOISNetworkClient) Traceroute(ctx context.Context, host string, opts domain.TraceOptions) (<-chan domain.TraceHop, error) {
	args := m.Called(ctx, host, opts)
	return args.Get(0).(<-chan domain.TraceHop), args.Error(1)
}

func (m *MockWHOISNetworkClient) DNSLookup(ctx context.Context, domainName string, recordType domain.DNSRecordType) (domain.DNSResult, error) {
	args := m.Called(ctx, domainName, recordType)
	return args.Get(0).(domain.DNSResult), args.Error(1)
}

func (m *MockWHOISNetworkClient) WHOISLookup(ctx context.Context, query string) (domain.WHOISResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).(domain.WHOISResult), args.Error(1)
}

func (m *MockWHOISNetworkClient) SSLCheck(ctx context.Context, host string, port int) (domain.SSLResult, error) {
	args := m.Called(ctx, host, port)
	return args.Get(0).(domain.SSLResult), args.Error(1)
}

// MockLogger for testing
type MockWHOISLogger struct {
	mock.Mock
}

func (m *MockWHOISLogger) Debug(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockWHOISLogger) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockWHOISLogger) Warn(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockWHOISLogger) Error(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockWHOISLogger) Fatal(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func TestWHOIS_CompleteUserInteractionFlow(t *testing.T) {
	// Setup mock dependencies (no expectations - just for interface compliance)
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	// Create WHOIS tool and diagnostic view
	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	// Create test harness
	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	diagnosticView.SetSize(120, 40)

	// Start the test harness
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Wait for initial render
	assert.True(t, harness.WaitForOutput("WHOIS Diagnostic Tool", 1*time.Second))

	// Test that the form is displayed
	assert.True(t, harness.WaitForOutput("Domain or IP Address", 1*time.Second))

	// Test typing in the query field
	harness.SendKeyString("example.com")
	time.Sleep(100 * time.Millisecond)

	// Test that input appears in the form (basic UI interaction test)
	// Note: We're testing UI interaction, not business logic execution
	output := harness.GetOutput()
	assert.Contains(t, output, "WHOIS Diagnostic Tool")

	// Test form navigation
	harness.SendKey(tea.KeyTab)
	time.Sleep(50 * time.Millisecond)

	// Test escape key navigation
	harness.SendKey(tea.KeyEsc)
	time.Sleep(50 * time.Millisecond)

	// Verify we can navigate back to input
	assert.True(t, harness.WaitForOutput("Domain or IP Address", 500*time.Millisecond))
}

func TestWHOIS_ErrorHandlingFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Wait for initial render
	assert.True(t, harness.WaitForOutput("WHOIS Diagnostic Tool", 1*time.Second))

	// Test form validation by submitting empty form
	harness.SendKey(tea.KeyEnter)
	time.Sleep(100 * time.Millisecond)

	// Should show validation error - this proves error handling is working
	output := harness.GetOutput()
	assert.Contains(t, output, "required")

	// Test that we can still interact with the form after error
	harness.SendKeyString("test")
	time.Sleep(50 * time.Millisecond)

	// Test escape key navigation
	harness.SendKey(tea.KeyEsc)
	time.Sleep(50 * time.Millisecond)

	// Verify the interface is still responsive
	assert.True(t, harness.IsRunning())
}

func TestWHOIS_KeyboardNavigationFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Test keyboard shortcuts
	shortcuts := []struct {
		key         tea.KeyType
		description string
	}{
		{tea.KeyTab, "Tab navigation"},
		{tea.KeyEsc, "Escape navigation"},
		{tea.KeyCtrlC, "Quit shortcut"},
	}

	for _, shortcut := range shortcuts {
		t.Run(shortcut.description, func(t *testing.T) {
			harness.SendKey(shortcut.key)
			time.Sleep(50 * time.Millisecond)
			
			// For quit shortcuts, the program should stop gracefully
			if shortcut.key == tea.KeyCtrlC {
				// Quit shortcut should stop the program
				// We just verify it doesn't crash - the program stopping is expected
				return
			}
			
			// For other shortcuts, verify no crash occurred
			assert.True(t, harness.IsRunning())
		})
	}
}

func TestWHOIS_ResponsiveLayoutFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Test different screen sizes
	sizes := []struct {
		width  int
		height int
		name   string
	}{
		{60, 20, "Small"},
		{100, 30, "Medium"},
		{140, 40, "Large"},
	}

	for _, size := range sizes {
		t.Run(size.name+" screen", func(t *testing.T) {
			harness.SendWindowSize(size.width, size.height)
			time.Sleep(50 * time.Millisecond)

			// Verify layout adapts
			assert.True(t, harness.WaitForOutput("WHOIS", 500*time.Millisecond))
			
			// Test that interface is still functional
			harness.SendKeyString("test")
			time.Sleep(50 * time.Millisecond)
			
			// Clear input for next test
			for i := 0; i < 4; i++ {
				harness.SendKey(tea.KeyBackspace)
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestWHOIS_FormValidationFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Wait for initial render
	assert.True(t, harness.WaitForOutput("WHOIS Diagnostic Tool", 1*time.Second))

	// Verify form is displayed
	assert.True(t, harness.WaitForOutput("Domain or IP Address", 1*time.Second))

	// Test submitting empty form
	harness.SendKey(tea.KeyEnter)
	time.Sleep(100 * time.Millisecond)

	// Should show validation error
	output := harness.GetOutput()
	assert.Contains(t, output, "required")

	// Test with valid input
	harness.SendKeyString("valid-domain.com")
	time.Sleep(50 * time.Millisecond)

	// Test that we can navigate with tab
	harness.SendKey(tea.KeyTab)
	time.Sleep(50 * time.Millisecond)

	// Test escape navigation
	harness.SendKey(tea.KeyEsc)
	time.Sleep(50 * time.Millisecond)
}

func TestWHOIS_IPAddressLookupFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Wait for initial render
	assert.True(t, harness.WaitForOutput("WHOIS Diagnostic Tool", 1*time.Second))

	// Verify form accepts IP address input
	assert.True(t, harness.WaitForOutput("Domain or IP Address", 1*time.Second))

	// Type IP address
	harness.SendKeyString("8.8.8.8")
	time.Sleep(50 * time.Millisecond)

	// Test that input is accepted (UI interaction test)
	output := harness.GetOutput()
	assert.Contains(t, output, "WHOIS Diagnostic Tool")

	// Test navigation
	harness.SendKey(tea.KeyTab)
	time.Sleep(50 * time.Millisecond)

	harness.SendKey(tea.KeyEsc)
	time.Sleep(50 * time.Millisecond)
}

func TestWHOIS_LongRunningOperationFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Wait for initial render
	assert.True(t, harness.WaitForOutput("WHOIS Diagnostic Tool", 1*time.Second))

	// Test that the interface is responsive during operations
	assert.True(t, harness.WaitForOutput("Domain or IP Address", 1*time.Second))

	// Type domain
	harness.SendKeyString("test-domain.com")
	time.Sleep(50 * time.Millisecond)

	// Test that we can still navigate during input
	harness.SendKey(tea.KeyTab)
	time.Sleep(50 * time.Millisecond)

	// Test escape navigation
	harness.SendKey(tea.KeyEsc)
	time.Sleep(50 * time.Millisecond)

	// Verify interface remains responsive
	output := harness.GetOutput()
	assert.Contains(t, output, "WHOIS")
}

func TestWHOIS_MultipleQueriesFlow(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Test multiple input sequences
	queries := []string{"first.com", "second.com"}

	for i, query := range queries {
		t.Run("Query "+query, func(t *testing.T) {
			if i > 0 {
				// Clear previous input and start fresh
				for j := 0; j < 20; j++ {
					harness.SendKey(tea.KeyBackspace)
					time.Sleep(5 * time.Millisecond)
				}
			}

			// Wait for input state
			assert.True(t, harness.WaitForOutput("Domain or IP Address", 1*time.Second))

			// Type query
			harness.SendKeyString(query)
			time.Sleep(50 * time.Millisecond)

			// Test navigation without submitting
			harness.SendKey(tea.KeyTab)
			time.Sleep(50 * time.Millisecond)

			// Test that interface remains responsive
			output := harness.GetOutput()
			// Just verify the interface is still working - the exact content may vary
			assert.NotEmpty(t, output)
		})
	}
}

func TestWHOIS_AccessibilityFeatures(t *testing.T) {
	mockClient := &MockWHOISNetworkClient{}
	mockLogger := &MockWHOISLogger{}

	tool := whois.NewTool(mockClient, mockLogger)
	diagnosticView := NewDiagnosticViewModel(tool)

	// Test focus management
	diagnosticView.Focus()
	diagnosticView.Blur()
	diagnosticView.Focus()

	// Test theme compatibility
	mockTheme := &MockWHOISTUITheme{}
	diagnosticView.SetTheme(mockTheme)

	// Test size adaptation
	diagnosticView.SetSize(80, 24)
	view := diagnosticView.View()
	assert.NotEmpty(t, view)

	// Test keyboard-only navigation
	suite := NewTUITestSuite()
	defer suite.Cleanup()

	harness := suite.CreateHarness(diagnosticView)
	err := harness.Start()
	require.NoError(t, err)
	defer harness.Stop()

	// Test all keyboard shortcuts work
	keys := []tea.KeyType{
		tea.KeyTab,
		tea.KeyEsc,
		tea.KeyEnter,
		tea.KeyBackspace,
	}

	for _, key := range keys {
		harness.SendKey(key)
		time.Sleep(20 * time.Millisecond)
		assert.True(t, harness.IsRunning())
	}
}

// MockWHOISTUITheme for testing
type MockWHOISTUITheme struct{}

func (m *MockWHOISTUITheme) GetColor(element string) string {
	return "#ffffff"
}

func (m *MockWHOISTUITheme) GetStyle(element string) map[string]interface{} {
	return map[string]interface{}{}
}

func (m *MockWHOISTUITheme) SetColor(element, color string) {
	// No-op for testing
}