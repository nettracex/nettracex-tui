// Package ping provides TUI tests for ping diagnostic tool
package ping

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)



// TestModel_InitialState tests the initial state of the ping model
func TestModel_InitialState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	if model.state != StateInput {
		t.Errorf("Expected initial state to be StateInput, got %v", model.state)
	}

	if model.focusedInput != 0 {
		t.Errorf("Expected initial focused input to be 0, got %d", model.focusedInput)
	}

	if model.hostInput.Value() != "" {
		t.Errorf("Expected empty host input, got %s", model.hostInput.Value())
	}

	if model.countInput.Value() != "4" {
		t.Errorf("Expected count input to be '4', got %s", model.countInput.Value())
	}

	if model.intervalInput.Value() != "1" {
		t.Errorf("Expected interval input to be '1', got %s", model.intervalInput.Value())
	}
}

// TestModel_InputNavigation tests keyboard navigation between input fields
func TestModel_InputNavigation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test Tab navigation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updatedModel.(*Model)
	if model.focusedInput != 1 {
		t.Errorf("Expected focused input to be 1 after Tab, got %d", model.focusedInput)
	}

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updatedModel.(*Model)
	if model.focusedInput != 2 {
		t.Errorf("Expected focused input to be 2 after second Tab, got %d", model.focusedInput)
	}

	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updatedModel.(*Model)
	if model.focusedInput != 0 {
		t.Errorf("Expected focused input to wrap to 0 after third Tab, got %d", model.focusedInput)
	}

	// Test Shift+Tab navigation
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updatedModel.(*Model)
	if model.focusedInput != 2 {
		t.Errorf("Expected focused input to be 2 after Shift+Tab, got %d", model.focusedInput)
	}
}

// TestModel_InputValidation tests input validation
func TestModel_InputValidation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test empty host input - should not start ping
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)
	if cmd != nil {
		t.Error("Expected no command when host input is empty")
	}

	// Set valid host input
	model.hostInput.SetValue("google.com")
	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)
	if cmd == nil {
		t.Error("Expected command when host input is valid")
	}
}

// TestModel_RealTimeUpdates tests real-time statistics updates
func TestModel_RealTimeUpdates(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set up mock ping results
	mockResults := []domain.PingResult{
		{
			Host: domain.NetworkHost{
				Hostname:  "google.com",
				IPAddress: []byte{8, 8, 8, 8},
			},
			Sequence:   1,
			RTT:        10 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  time.Now(),
		},
		{
			Host: domain.NetworkHost{
				Hostname:  "google.com",
				IPAddress: []byte{8, 8, 8, 8},
			},
			Sequence:   2,
			RTT:        15 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  time.Now(),
		},
	}

	mockClient.SetPingResponse("google.com", mockResults)

	// Start ping operation
	model.hostInput.SetValue("google.com")
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)

	// Simulate ping start
	updatedModel, _ = model.Update(pingStartMsg{
		host:       "google.com",
		count:      2,
		interval:   time.Second,
		continuous: false,
	})
	model = updatedModel.(*Model)

	if model.state != StateRunning {
		t.Errorf("Expected state to be StateRunning, got %v", model.state)
	}

	// Simulate progress updates
	for i, result := range mockResults {
		model.updateLiveStats(result)
		
		// Check live statistics
		stats := model.liveStats
		if stats.PacketsSent != i+1 {
			t.Errorf("Expected PacketsSent to be %d, got %d", i+1, stats.PacketsSent)
		}
		
		if stats.PacketsReceived != i+1 {
			t.Errorf("Expected PacketsReceived to be %d, got %d", i+1, stats.PacketsReceived)
		}
		
		if stats.LastRTT != result.RTT {
			t.Errorf("Expected LastRTT to be %v, got %v", result.RTT, stats.LastRTT)
		}
	}

	// Check final statistics
	finalStats := model.liveStats
	if finalStats.MinRTT != 10*time.Millisecond {
		t.Errorf("Expected MinRTT to be 10ms, got %v", finalStats.MinRTT)
	}
	
	if finalStats.MaxRTT != 15*time.Millisecond {
		t.Errorf("Expected MaxRTT to be 15ms, got %v", finalStats.MaxRTT)
	}
	
	expectedAvg := 12*time.Millisecond + 500*time.Microsecond // (10+15)/2
	if finalStats.AvgRTT != expectedAvg {
		t.Errorf("Expected AvgRTT to be %v, got %v", expectedAvg, finalStats.AvgRTT)
	}
}

// TestModel_LatencyGraph tests latency graph functionality
func TestModel_LatencyGraph(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Add some latency values
	latencies := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		15 * time.Millisecond,
		25 * time.Millisecond,
		12 * time.Millisecond,
	}

	for _, latency := range latencies {
		result := domain.PingResult{
			RTT: latency,
		}
		model.updateLiveStats(result)
	}

	// Check that graph values are stored
	graph := model.latencyGraph
	if len(graph.Values) != len(latencies) {
		t.Errorf("Expected %d graph values, got %d", len(latencies), len(graph.Values))
	}

	// Check that values match
	for i, expected := range latencies {
		if graph.Values[i] != expected {
			t.Errorf("Expected graph value %d to be %v, got %v", i, expected, graph.Values[i])
		}
	}

	// Test graph overflow (should keep only MaxValues)
	model.latencyGraph.MaxValues = 3
	
	// Clear existing values and reset stats
	model.latencyGraph.Values = make([]time.Duration, 0)
	model.liveStats = LiveStatistics{}
	
	// Add more values
	for i := 0; i < 5; i++ {
		result := domain.PingResult{
			RTT: time.Duration(i+30) * time.Millisecond,
		}
		model.updateLiveStats(result)
	}

	// Should only keep the last 3 values
	if len(model.latencyGraph.Values) != 3 {
		t.Errorf("Expected graph to keep only 3 values, got %d", len(model.latencyGraph.Values))
	}
}

// TestModel_PacketLossIndicator tests packet loss tracking
func TestModel_PacketLossIndicator(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Simulate mixed success and failure
	results := []domain.PingResult{
		{RTT: 10 * time.Millisecond}, // Success
		{Error: context.DeadlineExceeded}, // Failure
		{RTT: 15 * time.Millisecond}, // Success
		{RTT: 12 * time.Millisecond}, // Success
		{Error: context.DeadlineExceeded}, // Failure
	}

	for _, result := range results {
		model.updateLiveStats(result)
	}

	// Check packet loss statistics
	stats := model.liveStats
	if stats.PacketsSent != 5 {
		t.Errorf("Expected PacketsSent to be 5, got %d", stats.PacketsSent)
	}
	
	if stats.PacketsReceived != 3 {
		t.Errorf("Expected PacketsReceived to be 3, got %d", stats.PacketsReceived)
	}
	
	expectedLoss := 40.0 // 2/5 * 100
	if stats.PacketLoss != expectedLoss {
		t.Errorf("Expected PacketLoss to be %.1f%%, got %.1f%%", expectedLoss, stats.PacketLoss)
	}

	// Check packet loss indicator
	indicator := model.packetLoss
	if len(indicator.RecentResults) != 5 {
		t.Errorf("Expected 5 recent results, got %d", len(indicator.RecentResults))
	}

	expectedResults := []bool{true, false, true, true, false}
	for i, expected := range expectedResults {
		if indicator.RecentResults[i] != expected {
			t.Errorf("Expected recent result %d to be %v, got %v", i, expected, indicator.RecentResults[i])
		}
	}
}

// TestModel_ContinuousMode tests continuous ping mode
func TestModel_ContinuousMode(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set count to 0 for continuous mode
	model.countInput.SetValue("0")
	model.hostInput.SetValue("google.com")

	// Start ping
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)

	// Simulate ping start message
	updatedModel, _ = model.Update(pingStartMsg{
		host:       "google.com",
		count:      0,
		interval:   time.Second,
		continuous: true,
	})
	model = updatedModel.(*Model)

	if !model.continuousMode {
		t.Error("Expected continuous mode to be enabled")
	}

	if model.state != StateRunning {
		t.Errorf("Expected state to be StateRunning, got %v", model.state)
	}
}

// TestModel_ViewRendering tests that views render without errors
func TestModel_ViewRendering(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set reasonable dimensions
	model.SetSize(80, 24)

	// Test input state view
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view for input state")
	}

	if !strings.Contains(view, "Target Host") {
		t.Error("Expected input view to contain 'Target Host'")
	}

	// Test running state view
	model.state = StateRunning
	model.hostInput.SetValue("google.com")
	
	// Add some test data
	model.liveStats = LiveStatistics{
		PacketsSent:     5,
		PacketsReceived: 4,
		PacketLoss:      20.0,
		MinRTT:          10 * time.Millisecond,
		MaxRTT:          25 * time.Millisecond,
		AvgRTT:          15 * time.Millisecond,
		LastRTT:         12 * time.Millisecond,
	}

	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for running state")
	}

	if !strings.Contains(view, "Pinging google.com") {
		t.Error("Expected running view to contain ping target")
	}

	if !strings.Contains(view, "Live Statistics") {
		t.Error("Expected running view to contain live statistics")
	}

	// Test result state view
	model.state = StateResult
	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for result state")
	}

	// Test error state view
	model.state = StateError
	model.error = context.DeadlineExceeded
	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for error state")
	}

	if !strings.Contains(view, "Error") {
		t.Error("Expected error view to contain 'Error'")
	}
}

// TestModel_KeyboardShortcuts tests keyboard shortcuts in different states
func TestModel_KeyboardShortcuts(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Test escape key in running state
	model.state = StateRunning
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updatedModel.(*Model)
	if model.state != StateInput {
		t.Errorf("Expected escape to return to input state, got %v", model.state)
	}

	// Test quit key
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model = updatedModel.(*Model)
	if cmd == nil {
		t.Error("Expected quit command when 'q' is pressed")
	}

	// Test ctrl+c
	updatedModel, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = updatedModel.(*Model)
	if cmd == nil {
		t.Error("Expected quit command when Ctrl+C is pressed")
	}
}

// TestModel_PerformanceWithHighFrequencyUpdates tests performance with rapid updates
func TestModel_PerformanceWithHighFrequencyUpdates(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Set small update interval for high frequency
	model.updateInterval = 10 * time.Millisecond

	start := time.Now()

	// Simulate many rapid updates
	for i := 0; i < 100; i++ {
		result := domain.PingResult{
			Sequence: i + 1,
			RTT:      time.Duration(10+i%20) * time.Millisecond,
		}
		model.updateLiveStats(result)

		// Simulate tick updates
		updatedModel, _ := model.Update(tickMsg(time.Now()))
		model = updatedModel.(*Model)
	}

	elapsed := time.Since(start)

	// Should complete quickly (under 1 second for 100 updates)
	if elapsed > time.Second {
		t.Errorf("Performance test took too long: %v", elapsed)
	}

	// Verify data integrity after rapid updates
	stats := model.liveStats
	if stats.PacketsSent != 100 {
		t.Errorf("Expected 100 packets sent, got %d", stats.PacketsSent)
	}

	if stats.PacketsReceived != 100 {
		t.Errorf("Expected 100 packets received, got %d", stats.PacketsReceived)
	}
}

// TestModel_MemoryCleanup tests proper cleanup of resources
func TestModel_MemoryCleanup(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)

	// Fill up with data
	for i := 0; i < 200; i++ {
		result := domain.PingResult{
			RTT: time.Duration(i) * time.Millisecond,
		}
		model.updateLiveStats(result)
	}

	// Check that arrays don't grow unbounded
	if len(model.latencyGraph.Values) > model.latencyGraph.MaxValues {
		t.Errorf("Latency graph values not properly limited: %d > %d",
			len(model.latencyGraph.Values), model.latencyGraph.MaxValues)
	}

	if len(model.packetLoss.RecentResults) > model.packetLoss.MaxResults {
		t.Errorf("Packet loss results not properly limited: %d > %d",
			len(model.packetLoss.RecentResults), model.packetLoss.MaxResults)
	}

	// Test reset cleanup
	model.resetToInput()

	if len(model.results) != 0 {
		t.Errorf("Expected results to be cleared after reset, got %d", len(model.results))
	}

	if len(model.latencyGraph.Values) != 0 {
		t.Errorf("Expected graph values to be cleared after reset, got %d", len(model.latencyGraph.Values))
	}

	if len(model.packetLoss.RecentResults) != 0 {
		t.Errorf("Expected packet loss results to be cleared after reset, got %d", len(model.packetLoss.RecentResults))
	}
}