// Package traceroute provides unit tests for traceroute TUI model
package traceroute

import (
	"net"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	model := NewModel(tool)

	assert.NotNil(t, model)
	assert.Equal(t, tool, model.tool)
	assert.Equal(t, StateInput, model.state)
	assert.Equal(t, 30, model.maxHops)
	assert.Equal(t, 5*time.Second, model.timeout)
	assert.Equal(t, 60, model.packetSize)
	assert.Equal(t, 3, model.queries)
	assert.False(t, model.ipv6)
	assert.Empty(t, model.hops)
	assert.NotNil(t, model.progress)
	assert.NotNil(t, model.styles)
}

func TestNewModelStyles(t *testing.T) {
	styles := NewModelStyles()

	assert.NotNil(t, styles.Base)
	assert.NotNil(t, styles.Header)
	assert.NotNil(t, styles.Table)
	assert.NotNil(t, styles.Progress)
	assert.NotNil(t, styles.Statistics)
	assert.NotNil(t, styles.Error)
	assert.NotNil(t, styles.Help)
	assert.NotNil(t, styles.Focused)
	assert.NotNil(t, styles.Blurred)
}

func TestModel_Init(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	cmd := model.Init()
	assert.Nil(t, cmd)
}

func TestModel_SetSize(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetSize(100, 50)

	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)
}

func TestModel_Focus_Blur(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test Focus
	model.Focus()
	assert.True(t, model.focused)

	// Test Blur
	model.Blur()
	assert.False(t, model.focused)
}

func TestModel_SetHost(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetHost("example.com")
	assert.Equal(t, "example.com", model.host)
}

func TestModel_SetOptions(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetOptions(20, 10*time.Second, 128, 5, true)

	assert.Equal(t, 20, model.maxHops)
	assert.Equal(t, 10*time.Second, model.timeout)
	assert.Equal(t, 128, model.packetSize)
	assert.Equal(t, 5, model.queries)
	assert.True(t, model.ipv6)
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, cmd := model.Update(msg)

	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)

	m := updatedModel.(*Model)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestModel_Update_KeyMsg_Quit(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	tests := []string{"ctrl+c", "q"}

	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			if key == "ctrl+c" {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			}

			_, cmd := model.Update(msg)
			assert.NotNil(t, cmd)
		})
	}
}

func TestModel_Update_KeyMsg_Enter(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set host and test enter key in input state
	model.SetHost("example.com")
	model.state = StateInput

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := model.Update(msg)

	assert.NotNil(t, cmd)
}

func TestModel_Update_KeyMsg_Reset(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set state to completed and test reset
	model.state = StateCompleted
	model.hops = []domain.TraceHop{{Number: 1}}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, cmd := model.Update(msg)

	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)

	m := updatedModel.(*Model)
	assert.Equal(t, StateInput, m.state)
	assert.Empty(t, m.hops)
}

func TestModel_Update_HopReceivedMsg(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	hop := domain.TraceHop{
		Number: 1,
		Host: domain.NetworkHost{
			Hostname:  "router.example.com",
			IPAddress: net.ParseIP("192.168.1.1"),
		},
		RTT:       []time.Duration{10 * time.Millisecond},
		Timeout:   false,
		Timestamp: time.Now(),
	}

	msg := HopReceivedMsg{Hop: hop}
	updatedModel, cmd := model.Update(msg)

	assert.NotNil(t, updatedModel)
	// cmd might be nil if resultChan is not set, which is expected in this test
	_ = cmd

	m := updatedModel.(*Model)
	assert.Len(t, m.hops, 1)
	assert.Equal(t, hop, m.hops[0])
}

func TestModel_Update_TracerouteCompleteMsg(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Add some hops first
	model.hops = []domain.TraceHop{
		{Number: 1, RTT: []time.Duration{10 * time.Millisecond}, Timeout: false},
		{Number: 2, RTT: []time.Duration{20 * time.Millisecond}, Timeout: false},
	}

	msg := TracerouteCompleteMsg{}
	updatedModel, cmd := model.Update(msg)

	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)

	m := updatedModel.(*Model)
	assert.Equal(t, StateCompleted, m.state)
	assert.NotEqual(t, TracerouteStatistics{}, m.statistics)
}

func TestModel_Update_TracerouteErrorMsg(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	testErr := &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "network unreachable",
		Code:    "NETWORK_UNREACHABLE",
	}

	msg := TracerouteErrorMsg{Error: testErr}
	updatedModel, cmd := model.Update(msg)

	assert.NotNil(t, updatedModel)
	assert.Nil(t, cmd)

	m := updatedModel.(*Model)
	assert.Equal(t, StateError, m.state)
	assert.Equal(t, testErr, m.err)
}

func TestModel_View_InputState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.state = StateInput
	model.SetHost("example.com")

	view := model.View()

	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "Enter target host for traceroute")
	assert.Contains(t, view, "Host: example.com")
	assert.Contains(t, view, "Max Hops: 30")
	assert.Contains(t, view, "Press Enter to start traceroute")
	assert.Contains(t, view, "Enter: Start traceroute")
}

func TestModel_View_RunningState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.state = StateRunning
	model.hops = []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "router.example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:     []time.Duration{10 * time.Millisecond},
			Timeout: false,
		},
	}

	view := model.View()

	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "Hop 1/30")
	assert.Contains(t, view, "Esc: Cancel")
}

func TestModel_View_CompletedState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.state = StateCompleted
	model.hops = []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "router.example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:     []time.Duration{10 * time.Millisecond},
			Timeout: false,
		},
	}
	model.statistics = TracerouteStatistics{
		TotalHops:     1,
		CompletedHops: 1,
		TimeoutHops:   0,
		SuccessRate:   100.0,
		ReachedTarget: true,
	}

	view := model.View()

	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "--- Traceroute Statistics ---")
	assert.Contains(t, view, "r: Reset")
}

func TestModel_View_ErrorState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.state = StateError
	model.err = &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "network unreachable",
		Code:    "NETWORK_UNREACHABLE",
	}

	view := model.View()

	assert.Contains(t, view, "NetTraceX - Traceroute")
	assert.Contains(t, view, "Error: network unreachable")
	assert.Contains(t, view, "r: Reset")
}

func TestModel_reset(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set up model with some state
	model.state = StateCompleted
	model.hops = []domain.TraceHop{{Number: 1}}
	model.statistics = TracerouteStatistics{TotalHops: 1}
	model.err = &domain.NetTraceError{Message: "test error"}

	// Reset the model
	model.reset()

	assert.Equal(t, StateInput, model.state)
	assert.Empty(t, model.hops)
	assert.Equal(t, TracerouteStatistics{}, model.statistics)
	assert.Nil(t, model.err)
	assert.Nil(t, model.ctx)
	assert.Nil(t, model.cancel)
	assert.Nil(t, model.resultChan)
}

func TestModel_updateTable(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Add test hops
	model.hops = []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "router.example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:     []time.Duration{10 * time.Millisecond, 12 * time.Millisecond},
			Timeout: false,
		},
		{
			Number:  2,
			Host:    domain.NetworkHost{},
			RTT:     []time.Duration{},
			Timeout: true,
		},
	}

	// Update table
	model.updateTable()

	// Verify hops are stored correctly
	assert.Len(t, model.hops, 2)

	// Check first hop (successful hop)
	assert.Equal(t, 1, model.hops[0].Number)
	assert.Equal(t, "router.example.com", model.hops[0].Host.Hostname)
	assert.Equal(t, "192.168.1.1", model.hops[0].Host.IPAddress.String())
	assert.False(t, model.hops[0].Timeout)

	// Check second hop (timeout hop)
	assert.Equal(t, 2, model.hops[1].Number)
	assert.True(t, model.hops[1].Timeout)
}

func TestModel_renderInputForm(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.SetHost("example.com")
	model.SetOptions(20, 10*time.Second, 128, 5, true)

	form := model.renderInputForm()

	assert.Contains(t, form, "Enter target host for traceroute")
	assert.Contains(t, form, "Host: example.com")
	assert.Contains(t, form, "Max Hops: 20")
	assert.Contains(t, form, "Timeout: 10s")
	assert.Contains(t, form, "Packet Size: 128 bytes")
	assert.Contains(t, form, "Queries per hop: 5")
	assert.Contains(t, form, "IPv6: true")
	assert.Contains(t, form, "Press Enter to start traceroute")
}

func TestModel_renderProgress(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test with no hops
	progress := model.renderProgress()
	assert.Contains(t, progress, "Starting traceroute...")

	// Test with some hops
	model.hops = []domain.TraceHop{
		{Number: 1},
		{Number: 2},
		{Number: 3},
	}
	model.maxHops = 10

	progress = model.renderProgress()
	assert.Contains(t, progress, "Hop 3/10")
}

func TestModel_renderError(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	model.err = &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "network unreachable",
		Code:    "NETWORK_UNREACHABLE",
	}

	errorView := model.renderError()
	assert.Contains(t, errorView, "Error: network unreachable")
}

func TestModel_renderHelp(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	tests := []struct {
		state    ModelState
		expected []string
	}{
		{
			state:    StateInput,
			expected: []string{"Enter: Start traceroute", "q: Quit"},
		},
		{
			state:    StateRunning,
			expected: []string{"Esc: Cancel", "q: Quit"},
		},
		{
			state:    StateCompleted,
			expected: []string{"r: Reset", "q: Quit"},
		},
		{
			state:    StateError,
			expected: []string{"r: Reset", "q: Quit"},
		},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.state)), func(t *testing.T) {
			model.state = tt.state
			help := model.renderHelp()

			for _, expected := range tt.expected {
				assert.Contains(t, help, expected)
			}
		})
	}
}

// Test custom message types
func TestHopReceivedMsg(t *testing.T) {
	hop := domain.TraceHop{
		Number: 1,
		Host: domain.NetworkHost{
			Hostname:  "router.example.com",
			IPAddress: net.ParseIP("192.168.1.1"),
		},
		RTT:     []time.Duration{10 * time.Millisecond},
		Timeout: false,
	}

	msg := HopReceivedMsg{Hop: hop}
	assert.Equal(t, hop, msg.Hop)
}

func TestTracerouteCompleteMsg(t *testing.T) {
	msg := TracerouteCompleteMsg{}
	assert.NotNil(t, msg)
}

func TestTracerouteErrorMsg(t *testing.T) {
	testErr := &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "test error",
		Code:    "TEST_ERROR",
	}

	msg := TracerouteErrorMsg{Error: testErr}
	assert.Equal(t, testErr, msg.Error)
}

// Test model state transitions
func TestModel_StateTransitions(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Initial state should be Input
	assert.Equal(t, StateInput, model.state)

	// Simulate hop received (would transition to Running in real scenario)
	hop := domain.TraceHop{Number: 1, Timeout: false}
	msg := HopReceivedMsg{Hop: hop}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(*Model)
	assert.Len(t, m.hops, 1)

	// Simulate completion
	completeMsg := TracerouteCompleteMsg{}
	updatedModel, _ = m.Update(completeMsg)
	m = updatedModel.(*Model)
	assert.Equal(t, StateCompleted, m.state)

	// Simulate error
	model.state = StateInput
	errorMsg := TracerouteErrorMsg{Error: &domain.NetTraceError{Message: "test error"}}
	updatedModel, _ = model.Update(errorMsg)
	m = updatedModel.(*Model)
	assert.Equal(t, StateError, m.state)
	assert.NotNil(t, m.err)
}

// Test table rendering (since we removed the table component)
func TestModel_TableRendering(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Add test hops
	model.hops = []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "router.example.com",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:     []time.Duration{10 * time.Millisecond},
			Timeout: false,
		},
	}

	// Update table with the hops
	model.updateTable()

	// Test table rendering
	tableView := model.renderTable()
	assert.Contains(t, tableView, "Hop")
	assert.Contains(t, tableView, "Hostname")
	assert.Contains(t, tableView, "router")
	assert.Contains(t, tableView, "192.168.1.1")
}

// Test model implements tea.Model interface
func TestModel_ImplementsTeaModel(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Verify it implements tea.Model interface
	var _ tea.Model = model

	// Test interface methods
	// Init() can return nil, which is valid
	_ = model.Init()
	assert.NotNil(t, model.View())

	// Test Update with various message types
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	_, cmd := model.Update(windowMsg)
	assert.Nil(t, cmd)

	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd = model.Update(keyMsg)
	assert.Nil(t, cmd)
}
