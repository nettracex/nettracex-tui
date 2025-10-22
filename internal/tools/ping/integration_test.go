// Package ping provides integration tests for ping diagnostic tool
package ping

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)

// MockErrorHandler implements a simple error handler for testing
type MockErrorHandler struct{}

func (h *MockErrorHandler) Handle(err error) error                                                    { return err }
func (h *MockErrorHandler) HandleWithContext(err error, ctx map[string]interface{}) error           { return err }
func (h *MockErrorHandler) CanRecover(err error) bool                                                { return false }
func (h *MockErrorHandler) Recover(err error) error                                                  { return err }

// TestPingTUI_RealTimeIntegration tests the ping TUI with real network operations
func TestPingTUI_RealTimeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        30,
		PacketSize:     64,
		DNSServers:     []string{"8.8.8.8", "8.8.4.4"},
		UserAgent:      "NetTraceX/1.0",
		MaxConcurrency: 10,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
	}
	logger := &MockLogger{}
	errorHandler := &MockErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	tool := NewTool(client, logger)
	model := NewModel(tool)

	// Set up for a quick ping test
	model.SetSize(80, 24)
	model.hostInput.SetValue("8.8.8.8") // Google DNS - should be reliable
	model.countInput.SetValue("3")       // Just 3 pings for quick test
	model.intervalInput.SetValue("0.5")  // 500ms interval

	// Start the ping operation
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)

	if cmd == nil {
		t.Fatal("Expected command to start ping operation")
	}

	// Execute the start command
	msg := cmd()
	if msg == nil {
		t.Fatal("Expected start message from command")
	}

	// Process the start message
	updatedModel, cmd = model.Update(msg)
	model = updatedModel.(*Model)

	if model.state != StateRunning {
		t.Errorf("Expected state to be StateRunning, got %v", model.state)
	}

	// Wait a reasonable time for ping to complete
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Ping operation timed out")
		case <-ticker.C:
			// Simulate tick updates
			updatedModel, _ := model.Update(tickMsg(time.Now()))
			model = updatedModel.(*Model)

			// Check if ping completed
			if model.state == StateResult || model.state == StateError {
				goto completed
			}
		}
	}

completed:
	// Verify results
	if model.state == StateError {
		t.Fatalf("Ping operation failed with error: %v", model.error)
	}

	if model.state != StateResult {
		t.Errorf("Expected final state to be StateResult, got %v", model.state)
	}

	// Check that we have some results
	if len(model.results) == 0 {
		t.Error("Expected some ping results")
	}

	// Check live statistics
	stats := model.liveStats
	if stats.PacketsSent == 0 {
		t.Error("Expected some packets to be sent")
	}

	// Verify view renders without errors
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view after ping completion")
	}
}

// TestPingTUI_ContinuousModeIntegration tests continuous ping mode
func TestPingTUI_ContinuousModeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        30,
		PacketSize:     64,
		DNSServers:     []string{"8.8.8.8", "8.8.4.4"},
		UserAgent:      "NetTraceX/1.0",
		MaxConcurrency: 10,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
	}
	logger := &MockLogger{}
	errorHandler := &MockErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	tool := NewTool(client, logger)
	model := NewModel(tool)

	// Set up for continuous ping
	model.SetSize(80, 24)
	model.hostInput.SetValue("8.8.8.8")
	model.countInput.SetValue("0")      // 0 = continuous
	model.intervalInput.SetValue("0.2") // 200ms interval for faster test

	// Start continuous ping
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)

	if cmd == nil {
		t.Fatal("Expected command to start continuous ping")
	}

	// Execute the start command
	msg := cmd()
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(*Model)

	if !model.continuousMode {
		t.Error("Expected continuous mode to be enabled")
	}

	if model.state != StateRunning {
		t.Errorf("Expected state to be StateRunning, got %v", model.state)
	}

	// Let it run for a short time
	time.Sleep(1 * time.Second)

	// Simulate some tick updates
	for i := 0; i < 5; i++ {
		updatedModel, _ := model.Update(tickMsg(time.Now()))
		model = updatedModel.(*Model)
		time.Sleep(100 * time.Millisecond)
	}

	// Stop the continuous ping
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model = updatedModel.(*Model)

	// Should transition to result state
	if model.state != StateResult {
		t.Errorf("Expected state to be StateResult after stopping, got %v", model.state)
	}

	// Should have some results
	if len(model.results) == 0 {
		t.Error("Expected some ping results from continuous mode")
	}

	// Verify statistics were updated
	stats := model.liveStats
	if stats.PacketsSent == 0 {
		t.Error("Expected some packets to be sent in continuous mode")
	}
}

// TestPingTUI_ErrorHandling tests error handling in real scenarios
func TestPingTUI_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create real network client
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        30,
		PacketSize:     64,
		DNSServers:     []string{"8.8.8.8", "8.8.4.4"},
		UserAgent:      "NetTraceX/1.0",
		MaxConcurrency: 10,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
	}
	logger := &MockLogger{}
	errorHandler := &MockErrorHandler{}
	client := network.NewClient(config, errorHandler, logger)
	tool := NewTool(client, logger)
	model := NewModel(tool)

	// Set up for ping to invalid host
	model.SetSize(80, 24)
	model.hostInput.SetValue("invalid.nonexistent.domain.test")
	model.countInput.SetValue("2")
	model.intervalInput.SetValue("1")

	// Start the ping operation
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updatedModel.(*Model)

	if cmd == nil {
		t.Fatal("Expected command to start ping operation")
	}

	// Execute the start command
	msg := cmd()
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(*Model)

	// Wait for operation to complete or error
	timeout := time.After(15 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Timeout is acceptable for invalid domains
			goto completed
		case <-ticker.C:
			// Simulate tick updates
			updatedModel, _ := model.Update(tickMsg(time.Now()))
			model = updatedModel.(*Model)

			// Check if operation completed
			if model.state == StateResult || model.state == StateError {
				goto completed
			}
		}
	}

completed:
	// Verify error handling
	if model.state == StateError {
		// Error state is expected for invalid domain
		if model.error == nil {
			t.Error("Expected error to be set in error state")
		}

		// Verify error view renders
		view := model.View()
		if view == "" {
			t.Error("Expected non-empty error view")
		}
	} else if model.state == StateResult {
		// If it somehow succeeded, check that packet loss is handled
		stats := model.liveStats
		if stats.PacketsSent > 0 && stats.PacketsReceived == 0 {
			// All packets lost - this is acceptable for invalid domain
			if stats.PacketLoss != 100.0 {
				t.Errorf("Expected 100%% packet loss for invalid domain, got %.1f%%", stats.PacketLoss)
			}
		}
	}
}

// TestPingTUI_PerformanceUnderLoad tests TUI performance with rapid updates
func TestPingTUI_PerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create mock client for controlled testing
	client := network.NewMockClient()
	logger := &MockLogger{}
	tool := NewTool(client, logger)
	model := NewModel(tool)

	// Set up rapid ping simulation
	model.SetSize(120, 40) // Larger size for more complex rendering
	model.updateInterval = 50 * time.Millisecond // High frequency updates

	// Create many mock results for performance testing
	var mockResults []domain.PingResult
	for i := 0; i < 100; i++ {
		result := domain.PingResult{
			Host: domain.NetworkHost{
				Hostname:  "test.example.com",
				IPAddress: []byte{192, 168, 1, 1},
			},
			Sequence:   i + 1,
			RTT:        time.Duration(10+i%50) * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  time.Now().Add(time.Duration(i) * 100 * time.Millisecond),
		}
		mockResults = append(mockResults, result)
	}

	client.SetPingResponse("test.example.com", mockResults)

	start := time.Now()

	// Simulate rapid updates
	for i, result := range mockResults {
		model.updateLiveStats(result)

		// Simulate tick updates every few results
		if i%5 == 0 {
			updatedModel, _ := model.Update(tickMsg(time.Now()))
			model = updatedModel.(*Model)

			// Render view to test rendering performance
			view := model.View()
			if view == "" {
				t.Errorf("View rendering failed at update %d", i)
			}
		}
	}

	elapsed := time.Since(start)

	// Performance check - should handle 100 updates quickly
	if elapsed > 2*time.Second {
		t.Errorf("Performance test took too long: %v (expected < 2s)", elapsed)
	}

	// Verify data integrity after rapid updates
	stats := model.liveStats
	if stats.PacketsSent != 100 {
		t.Errorf("Expected 100 packets sent, got %d", stats.PacketsSent)
	}

	if stats.PacketsReceived != 100 {
		t.Errorf("Expected 100 packets received, got %d", stats.PacketsReceived)
	}

	// Verify graph and indicators are properly bounded
	if len(model.latencyGraph.Values) > model.latencyGraph.MaxValues {
		t.Errorf("Latency graph exceeded max values: %d > %d",
			len(model.latencyGraph.Values), model.latencyGraph.MaxValues)
	}

	if len(model.packetLoss.RecentResults) > model.packetLoss.MaxResults {
		t.Errorf("Packet loss indicator exceeded max results: %d > %d",
			len(model.packetLoss.RecentResults), model.packetLoss.MaxResults)
	}
}

// TestPingTUI_ResponsiveLayout tests responsive layout at different sizes
func TestPingTUI_ResponsiveLayout(t *testing.T) {
	client := network.NewMockClient()
	logger := &MockLogger{}
	tool := NewTool(client, logger)
	model := NewModel(tool)

	// Test different screen sizes
	sizes := []struct {
		width, height int
		name          string
	}{
		{40, 10, "very small"},
		{80, 24, "standard"},
		{120, 40, "large"},
		{200, 60, "very large"},
	}

	for _, size := range sizes {
		t.Run(size.name, func(t *testing.T) {
			model.SetSize(size.width, size.height)

			// Test input state
			view := model.View()
			if view == "" {
				t.Errorf("Empty view for %s size in input state", size.name)
			}

			// Add some test data and test running state
			model.state = StateRunning
			model.hostInput.SetValue("test.example.com")
			model.liveStats = LiveStatistics{
				PacketsSent:     10,
				PacketsReceived: 8,
				PacketLoss:      20.0,
				MinRTT:          5 * time.Millisecond,
				MaxRTT:          50 * time.Millisecond,
				AvgRTT:          25 * time.Millisecond,
				LastRTT:         30 * time.Millisecond,
				ElapsedTime:     10 * time.Second,
			}

			// Add some graph data
			for i := 0; i < 10; i++ {
				model.latencyGraph.Values = append(model.latencyGraph.Values,
					time.Duration(10+i*5)*time.Millisecond)
			}

			// Add packet loss data
			for i := 0; i < 10; i++ {
				model.packetLoss.RecentResults = append(model.packetLoss.RecentResults, i%4 != 0)
			}

			view = model.View()
			if view == "" {
				t.Errorf("Empty view for %s size in running state", size.name)
			}

			// Verify graph adapts to size
			if model.latencyGraph.Width > size.width-8 && size.width > 20 {
				t.Errorf("Graph width not properly constrained for %s size", size.name)
			}
		})
	}
}