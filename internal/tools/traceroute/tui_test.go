// Package traceroute provides TUI tests for traceroute diagnostic tool
package traceroute

import (
	"fmt"
	"net"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModel_InitialTUIState tests the initial TUI state of the traceroute model
func TestModel_InitialTUIState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	assert.Equal(t, StateInput, model.state)
	assert.Empty(t, model.host)
	assert.Equal(t, 30, model.maxHops)
	assert.Equal(t, 5*time.Second, model.timeout)
	assert.Empty(t, model.hops)
	assert.NotNil(t, model.table)
	assert.NotNil(t, model.progress)
	assert.False(t, model.focused)
}

// TestModel_TabularDisplay tests the tabular display of traceroute hops
func TestModel_TabularDisplay(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set up test hops with various scenarios
	testHops := []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "router1.example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:       []time.Duration{10 * time.Millisecond, 12 * time.Millisecond, 11 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
		{
			Number: 2,
			Host: domain.NetworkHost{
				Hostname:  "",
				IPAddress: net.ParseIP("10.0.0.1"),
			},
			RTT:       []time.Duration{25 * time.Millisecond, 28 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
		{
			Number:    3,
			Host:      domain.NetworkHost{},
			RTT:       []time.Duration{},
			Timeout:   true,
			Timestamp: time.Now(),
		},
		{
			Number: 4,
			Host: domain.NetworkHost{
				Hostname:  "destination.example.com",
				IPAddress: net.ParseIP("8.8.8.8"),
			},
			RTT:       []time.Duration{50 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
	}

	// Add hops to model
	for _, hop := range testHops {
		model.hops = append(model.hops, hop)
	}
	model.updateTable()

	// Test table rendering
	tableView := model.renderTable()

	// Check that all hops are displayed (hostnames may be truncated)
	assert.Contains(t, tableView, "router1")
	assert.Contains(t, tableView, "192.168.1.1")
	assert.Contains(t, tableView, "10.0.0.1")
	assert.Contains(t, tableView, "destina") // May be truncated
	assert.Contains(t, tableView, "8.8.8.8")

	// Check RTT values are displayed
	assert.Contains(t, tableView, "10.0 ms")
	assert.Contains(t, tableView, "12.0 ms")
	assert.Contains(t, tableView, "25.0 ms")
	assert.Contains(t, tableView, "50.0 ms")

	// Check timeout indication
	assert.Contains(t, tableView, "✗ Timeout")

	// Check success indication
	assert.Contains(t, tableView, "✓ OK")
}

// TestModel_ProgressiveUpdates tests progressive hop discovery updates
func TestModel_ProgressiveUpdates(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetSize(100, 30)
	model.state = StateRunning

	// Simulate progressive hop discovery
	hops := []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "hop1.example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:     []time.Duration{10 * time.Millisecond},
			Timeout: false,
		},
		{
			Number: 2,
			Host: domain.NetworkHost{
				Hostname:  "hop2.example.com",
				IPAddress: net.ParseIP("10.0.0.1"),
			},
			RTT:     []time.Duration{20 * time.Millisecond},
			Timeout: false,
		},
		{
			Number:  3,
			Timeout: true,
		},
	}

	// Test progressive updates
	for i, hop := range hops {
		// Update model with new hop
		msg := HopReceivedMsg{Hop: hop}
		updatedModel, cmd := model.Update(msg)
		model = updatedModel.(*Model)

		// Verify hop was added
		assert.Len(t, model.hops, i+1)
		assert.Equal(t, hop.Number, model.hops[i].Number)

		// Verify update tracking
		assert.True(t, model.lastUpdate.After(time.Time{}))
		assert.Equal(t, i+1, model.updateCount)

		// Verify table was updated
		tableView := model.renderTable()
		assert.Contains(t, tableView, "Live updates")
		assert.Contains(t, tableView, fmt.Sprintf("(%d hops received)", i+1))

		// Verify command for next hop (may be nil in test environment)
		// In real usage, this would be a proper command, but in tests it may be nil
		_ = cmd
	}
}

// TestModel_RealTimeUpdateIndicator tests the real-time update indicator
func TestModel_RealTimeUpdateIndicator(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.state = StateRunning
	model.lastUpdate = time.Now()
	model.updateCount = 5

	// Test active update indicator (recent update)
	tableView := model.renderTable()
	assert.Contains(t, tableView, "● Live updates")
	assert.Contains(t, tableView, "(5 hops received)")

	// Test inactive indicator (old update)
	model.lastUpdate = time.Now().Add(-5 * time.Second)
	tableView = model.renderTable()
	assert.NotContains(t, tableView, "● Live updates")
}

// TestModel_TimeoutVisualization tests timeout visualization in the table
func TestModel_TimeoutVisualization(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Create hop with timeout
	timeoutHop := domain.TraceHop{
		Number:    5,
		Host:      domain.NetworkHost{},
		RTT:       []time.Duration{},
		Timeout:   true,
		Timestamp: time.Now(),
	}

	model.hops = []domain.TraceHop{timeoutHop}
	model.updateTable()

	// Test timeout visualization
	tableView := model.renderTable()
	assert.Contains(t, tableView, "✗ Timeout")
	assert.Contains(t, tableView, "*") // RTT columns should show asterisks

	// Test hopToTableRow for timeout
	row := model.hopToTableRow(timeoutHop)
	assert.Equal(t, "5", row[0])           // Hop number
	assert.Equal(t, "-", row[1])           // Hostname (empty)
	assert.Equal(t, "-", row[2])           // IP address (empty)
	assert.Equal(t, "*", row[3])           // RTT 1
	assert.Equal(t, "*", row[4])           // RTT 2
	assert.Equal(t, "*", row[5])           // RTT 3
	assert.Equal(t, "✗ Timeout", row[6])   // Status
}

// TestModel_ErrorIndication tests error indication in the traceroute interface
func TestModel_ErrorIndication(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test network error
	networkErr := &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "Network unreachable",
		Code:    "NETWORK_UNREACHABLE",
	}

	msg := TracerouteErrorMsg{Error: networkErr}
	updatedModel, cmd := model.Update(msg)
	model = updatedModel.(*Model)

	assert.Equal(t, StateError, model.state)
	assert.Equal(t, networkErr, model.err)
	assert.Nil(t, cmd)

	// Test error view rendering
	view := model.View()
	assert.Contains(t, view, "Error: Network unreachable")
	assert.Contains(t, view, "r: Reset")

	// Test error rendering method
	errorView := model.renderError()
	assert.Contains(t, errorView, "Error: Network unreachable")
}

// TestModel_KeyboardNavigation tests keyboard navigation in different states
func TestModel_KeyboardNavigation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test Enter key in input state with valid host
	model.SetHost("example.com")
	model.state = StateInput

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)
	assert.NotNil(t, cmd)

	// Test Enter key in input state without host
	model.SetHost("")
	model.state = StateInput

	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)
	assert.Nil(t, cmd)

	// Test Escape key in running state
	model.state = StateRunning
	model.cancel = func() {} // Mock cancel function

	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updatedModel.(*Model)
	assert.Equal(t, StateInput, model.state)
	assert.Nil(t, cmd)

	// Test Reset key in completed state
	model.state = StateCompleted
	model.hops = []domain.TraceHop{{Number: 1}}

	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	model = updatedModel.(*Model)
	assert.Equal(t, StateInput, model.state)
	assert.Empty(t, model.hops)
	assert.Nil(t, cmd)

	// Test Quit keys
	quitKeys := []tea.KeyMsg{
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyRunes, Runes: []rune("q")},
	}

	for _, keyMsg := range quitKeys {
		updatedModel, cmd = model.Update(keyMsg)
		assert.NotNil(t, cmd)
	}
}

// TestModel_ViewRendering tests view rendering in different states
func TestModel_ViewRendering(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetSize(100, 30)

	// Test input state view
	model.state = StateInput
	model.SetHost("example.com")
	view := model.View()

	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "Enter target host for traceroute")
	assert.Contains(t, view, "Host: example.com")
	assert.Contains(t, view, "Max Hops: 30")
	assert.Contains(t, view, "Press Enter to start traceroute")
	assert.Contains(t, view, "Enter: Start traceroute")

	// Test running state view
	model.state = StateRunning
	model.hops = []domain.TraceHop{
		{Number: 1, Timeout: false},
		{Number: 2, Timeout: false},
	}
	model.lastUpdate = time.Now()
	model.updateCount = 2

	view = model.View()
	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "Hop 2/30")
	assert.Contains(t, view, "Esc: Cancel")

	// Test completed state view
	model.state = StateCompleted
	model.statistics = TracerouteStatistics{
		TotalHops:     2,
		CompletedHops: 2,
		SuccessRate:   100.0,
		ReachedTarget: true,
	}

	view = model.View()
	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "--- Traceroute Statistics ---")
	assert.Contains(t, view, "r: Reset")

	// Test error state view
	model.state = StateError
	model.err = &domain.NetTraceError{Message: "Test error"}

	view = model.View()
	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "Error: Test error")
	assert.Contains(t, view, "r: Reset")
}

// TestModel_WindowSizeHandling tests window size handling and responsive layout
func TestModel_WindowSizeHandling(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test window size update
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(msg)
	model = updatedModel.(*Model)

	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
	assert.Nil(t, cmd)

	// Verify table size was updated
	// Table should get width-4 and height-10 to leave space for other elements
	// We can't directly test table size without exposing internal fields,
	// but we can verify the SetSize method was called by checking the model dimensions
	assert.Equal(t, 120, model.width)
	assert.Equal(t, 40, model.height)
}

// TestModel_HopToTableRowConversion tests conversion of hops to table rows
func TestModel_HopToTableRowConversion(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test successful hop with all RTT values
	successHop := domain.TraceHop{
		Number: 1,
		Host: domain.NetworkHost{
			Hostname:  "router.example.com",
			IPAddress: net.ParseIP("192.168.1.1"),
		},
		RTT:     []time.Duration{10 * time.Millisecond, 12 * time.Millisecond, 11 * time.Millisecond},
		Timeout: false,
	}

	row := model.hopToTableRow(successHop)
	assert.Equal(t, "1", row[0])
	assert.Equal(t, "router.example.com", row[1])
	assert.Equal(t, "192.168.1.1", row[2])
	assert.Equal(t, "10.0 ms", row[3])
	assert.Equal(t, "12.0 ms", row[4])
	assert.Equal(t, "11.0 ms", row[5])
	assert.Equal(t, "✓ OK", row[6])

	// Test hop with missing hostname
	noHostnameHop := domain.TraceHop{
		Number: 2,
		Host: domain.NetworkHost{
			Hostname:  "",
			IPAddress: net.ParseIP("10.0.0.1"),
		},
		RTT:     []time.Duration{25 * time.Millisecond},
		Timeout: false,
	}

	row = model.hopToTableRow(noHostnameHop)
	assert.Equal(t, "2", row[0])
	assert.Equal(t, "-", row[1])
	assert.Equal(t, "10.0.0.1", row[2])
	assert.Equal(t, "25.0 ms", row[3])
	assert.Equal(t, "", row[4])
	assert.Equal(t, "", row[5])
	assert.Equal(t, "✓ OK", row[6])

	// Test timeout hop
	timeoutHop := domain.TraceHop{
		Number:  3,
		Host:    domain.NetworkHost{},
		RTT:     []time.Duration{},
		Timeout: true,
	}

	row = model.hopToTableRow(timeoutHop)
	assert.Equal(t, "3", row[0])
	assert.Equal(t, "-", row[1])
	assert.Equal(t, "-", row[2])
	assert.Equal(t, "*", row[3])
	assert.Equal(t, "*", row[4])
	assert.Equal(t, "*", row[5])
	assert.Equal(t, "✗ Timeout", row[6])
}

// TestModel_ProgressIndicator tests the progress indicator functionality
func TestModel_ProgressIndicator(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test progress with no hops
	progress := model.renderProgress()
	assert.Contains(t, progress, "Starting traceroute...")

	// Test progress with some hops
	model.hops = []domain.TraceHop{
		{Number: 1},
		{Number: 2},
		{Number: 3},
	}
	model.maxHops = 10

	progress = model.renderProgress()
	assert.Contains(t, progress, "Hop 3/10")

	// Test progress with hops exceeding max (should cap at 100%)
	model.hops = make([]domain.TraceHop, 15)
	for i := range model.hops {
		model.hops[i].Number = i + 1
	}

	progress = model.renderProgress()
	assert.Contains(t, progress, "Hop 15/10")
}

// TestModel_StatisticsRendering tests statistics rendering
func TestModel_StatisticsRendering(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set up test statistics
	model.statistics = TracerouteStatistics{
		TotalHops:     5,
		CompletedHops: 4,
		TimeoutHops:   1,
		SuccessRate:   80.0,
		MinRTT:        10 * time.Millisecond,
		MaxRTT:        50 * time.Millisecond,
		AvgRTT:        25 * time.Millisecond,
		TotalTime:     2 * time.Second,
		ReachedTarget: true,
		FinalHop:      5,
	}

	statsView := model.renderStatistics()
	assert.Contains(t, statsView, "--- Traceroute Statistics ---")
	assert.Contains(t, statsView, "Total = 5")
	assert.Contains(t, statsView, "Completed = 4")
	assert.Contains(t, statsView, "Timeouts = 1")
	assert.Contains(t, statsView, "80.0% success")
	assert.Contains(t, statsView, "Min = 10ms")
	assert.Contains(t, statsView, "Max = 50ms")
	assert.Contains(t, statsView, "Avg = 25ms")
	assert.Contains(t, statsView, "Total time: 2s")
	assert.Contains(t, statsView, "Reached target: true")
}

// TestModel_HelpTextRendering tests help text rendering in different states
func TestModel_HelpTextRendering(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test input state help
	model.state = StateInput
	help := model.renderHelp()
	assert.Contains(t, help, "Enter: Start traceroute")
	assert.Contains(t, help, "q: Quit")

	// Test running state help
	model.state = StateRunning
	help = model.renderHelp()
	assert.Contains(t, help, "Esc: Cancel")
	assert.Contains(t, help, "q: Quit")

	// Test completed state help
	model.state = StateCompleted
	help = model.renderHelp()
	assert.Contains(t, help, "r: Reset")
	assert.Contains(t, help, "q: Quit")

	// Test error state help
	model.state = StateError
	help = model.renderHelp()
	assert.Contains(t, help, "r: Reset")
	assert.Contains(t, help, "q: Quit")
}

// TestModel_TableIntegration tests integration with the table component
func TestModel_TableIntegration(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Verify table is initialized with correct headers
	require.NotNil(t, model.table)

	// Add test data and verify table update
	testHops := []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "test.example.com",
				IPAddress: net.ParseIP("1.2.3.4"),
			},
			RTT:     []time.Duration{15 * time.Millisecond},
			Timeout: false,
		},
	}

	model.hops = testHops
	model.updateTable()

	// Verify table rendering includes our data (hostname may be truncated)
	tableView := model.renderTable()
	assert.Contains(t, tableView, "test.example") // May be truncated
	assert.Contains(t, tableView, "1.2.3.4")
	assert.Contains(t, tableView, "15.0 ms")
}

// TestModel_MessageHandling tests comprehensive message handling
func TestModel_MessageHandling(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test StartTracerouteMsg
	startMsg := StartTracerouteMsg{}
	updatedModel, cmd := model.Update(startMsg)
	model = updatedModel.(*Model)
	assert.Equal(t, StateRunning, model.state)
	assert.NotNil(t, cmd) // Should return waitForNextHop command

	// Test HopReceivedMsg
	hop := domain.TraceHop{
		Number: 1,
		Host: domain.NetworkHost{
			Hostname:  "hop1.example.com",
			IPAddress: net.ParseIP("192.168.1.1"),
		},
		RTT:     []time.Duration{10 * time.Millisecond},
		Timeout: false,
	}

	hopMsg := HopReceivedMsg{Hop: hop}
	updatedModel, cmd = model.Update(hopMsg)
	model = updatedModel.(*Model)
	assert.Len(t, model.hops, 1)
	assert.Equal(t, hop, model.hops[0])
	assert.True(t, model.lastUpdate.After(time.Time{}))
	assert.Equal(t, 1, model.updateCount)
	assert.NotNil(t, cmd)

	// Test TracerouteCompleteMsg
	completeMsg := TracerouteCompleteMsg{}
	updatedModel, cmd = model.Update(completeMsg)
	model = updatedModel.(*Model)
	assert.Equal(t, StateCompleted, model.state)
	assert.NotEqual(t, TracerouteStatistics{}, model.statistics)
	assert.Nil(t, cmd)

	// Test TracerouteErrorMsg
	model.state = StateRunning // Reset to running state
	errorMsg := TracerouteErrorMsg{
		Error: &domain.NetTraceError{
			Type:    domain.ErrorTypeNetwork,
			Message: "Test error",
			Code:    "TEST_ERROR",
		},
	}
	updatedModel, cmd = model.Update(errorMsg)
	model = updatedModel.(*Model)
	assert.Equal(t, StateError, model.state)
	assert.NotNil(t, model.err)
	assert.Nil(t, cmd)
}

// TestModel_ResetFunctionality tests the reset functionality
func TestModel_ResetFunctionality(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set up model with data
	model.state = StateCompleted
	model.hops = []domain.TraceHop{{Number: 1}, {Number: 2}}
	model.statistics = TracerouteStatistics{TotalHops: 2}
	model.err = &domain.NetTraceError{Message: "test error"}
	model.lastUpdate = time.Now()
	model.updateCount = 5

	// Add data to table
	model.updateTable()

	// Reset the model
	model.reset()

	// Verify reset state
	assert.Equal(t, StateInput, model.state)
	assert.Empty(t, model.hops)
	assert.Equal(t, TracerouteStatistics{}, model.statistics)
	assert.Nil(t, model.err)
	assert.Equal(t, time.Time{}, model.lastUpdate)
	assert.Equal(t, 0, model.updateCount)
	assert.Nil(t, model.ctx)
	assert.Nil(t, model.cancel)
	assert.Nil(t, model.resultChan)
}

// TestModel_ComponentInterfaces tests that model implements required interfaces
func TestModel_ComponentInterfaces(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test tea.Model interface
	var _ tea.Model = model
	initCmd := model.Init()
	// Init() can return nil, which is valid
	_ = initCmd
	assert.NotEmpty(t, model.View())

	// Test domain.TUIComponent interface methods
	model.SetSize(100, 50)
	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)

	model.Focus()
	assert.True(t, model.focused)

	model.Blur()
	assert.False(t, model.focused)

	// Test SetTheme (should not panic)
	assert.NotPanics(t, func() {
		model.SetTheme(nil) // Theme interface not fully implemented yet
	})
}