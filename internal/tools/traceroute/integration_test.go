// Package traceroute provides integration tests for traceroute functionality
package traceroute

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTracerouteIntegration tests the complete traceroute workflow
func TestTracerouteIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create network configuration
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        10,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     100 * time.Millisecond,
	}

	// Create mock logger that captures log messages
	logger := &TestLogger{messages: make([]LogMessage, 0)}

	// Create network client
	client := network.NewClient(config, nil, logger)

	// Create traceroute tool
	tool := NewTool(client, logger)

	// Test parameters
	params := domain.NewTracerouteParameters("127.0.0.1", domain.TraceOptions{
		MaxHops:    5,
		Timeout:    2 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	// Execute traceroute
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check result data
	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)
	assert.NotEmpty(t, hops)

	// Verify metadata
	metadata := result.Metadata()
	assert.Equal(t, "traceroute", metadata["tool"])
	assert.Equal(t, "127.0.0.1", metadata["host"])
	assert.Equal(t, 5, metadata["max_hops"])

	// Verify statistics
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)
	assert.Greater(t, stats.TotalHops, 0)
	assert.GreaterOrEqual(t, stats.CompletedHops, 0)
	assert.GreaterOrEqual(t, stats.TimeoutHops, 0)
	assert.Equal(t, stats.TotalHops, stats.CompletedHops+stats.TimeoutHops)

	// Verify logger was called
	assert.NotEmpty(t, logger.messages)
	assert.Contains(t, logger.GetMessages("info"), "Executing traceroute operation")
	assert.Contains(t, logger.GetMessages("info"), "Traceroute operation completed")
}

// TestTracerouteWithRealHost tests traceroute to a real external host
func TestTracerouteWithRealHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create network configuration
	config := &domain.NetworkConfig{
		Timeout:        10 * time.Second,
		MaxHops:        15,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  3,
		RetryDelay:     200 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	// Test with a reliable public DNS server
	params := domain.NewTracerouteParameters("8.8.8.8", domain.TraceOptions{
		MaxHops:    15,
		Timeout:    5 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)
	assert.NotEmpty(t, hops)

	// Verify we have at least one hop
	assert.Greater(t, len(hops), 0)

	// Check that hops are numbered sequentially
	for i, hop := range hops {
		assert.Equal(t, i+1, hop.Number)
	}

	// Verify statistics make sense
	metadata := result.Metadata()
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)
	assert.Equal(t, len(hops), stats.TotalHops)
	assert.GreaterOrEqual(t, stats.SuccessRate, 0.0)
	assert.LessOrEqual(t, stats.SuccessRate, 100.0)
}

// TestTracerouteTimeout tests traceroute behavior with timeouts
func TestTracerouteTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &domain.NetworkConfig{
		Timeout:        1 * time.Second, // Very short timeout to force timeouts
		MaxHops:        5,
		PacketSize:     60,
		MaxConcurrency: 1,
		RetryAttempts:  1,
		RetryDelay:     50 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	// Use a non-routable IP to force timeouts
	params := domain.NewTracerouteParameters("192.0.2.1", domain.TraceOptions{
		MaxHops:    5,
		Timeout:    500 * time.Millisecond,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)

	// Should complete without error even with timeouts
	require.NoError(t, err)
	require.NotNil(t, result)

	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)

	// Verify statistics account for timeouts
	metadata := result.Metadata()
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)

	if len(hops) > 0 {
		// Should have some timeouts due to non-routable IP
		assert.GreaterOrEqual(t, stats.TimeoutHops, 0)
		assert.Equal(t, stats.TotalHops, stats.CompletedHops+stats.TimeoutHops)
	}
}

// TestTracerouteIPv6 tests IPv6 traceroute functionality
func TestTracerouteIPv6(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &domain.NetworkConfig{
		Timeout:        10 * time.Second,
		MaxHops:        10,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     100 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	// Test with IPv6 loopback
	params := domain.NewTracerouteParameters("::1", domain.TraceOptions{
		MaxHops:    5,
		Timeout:    5 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)

	// IPv6 might not be available on all systems, so we handle both cases
	if err != nil {
		// If IPv6 is not available, the error should be network-related
		netErr, ok := err.(*domain.NetTraceError)
		if ok {
			assert.Equal(t, domain.ErrorTypeNetwork, netErr.Type)
		}
		t.Skipf("IPv6 not available or supported: %v", err)
		return
	}

	require.NotNil(t, result)

	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)

	// If IPv6 works, verify the results
	if len(hops) > 0 {
		// Check that we got IPv6 addresses
		for _, hop := range hops {
			if !hop.Timeout && hop.Host.IPAddress != nil {
				// Should be IPv6 address (16 bytes)
				assert.Equal(t, 16, len(hop.Host.IPAddress))
			}
		}
	}
}

// TestTracerouteCancel tests cancellation of traceroute operations
func TestTracerouteCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &domain.NetworkConfig{
		Timeout:        30 * time.Second, // Long timeout
		MaxHops:        30,
		PacketSize:     60,
		MaxConcurrency: 1,
		RetryAttempts:  1,
		RetryDelay:     100 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	params := domain.NewTracerouteParameters("8.8.8.8", domain.TraceOptions{
		MaxHops:    30,
		Timeout:    10 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Start traceroute in goroutine
	resultChan := make(chan struct {
		result domain.Result
		err    error
	}, 1)

	go func() {
		result, err := tool.Execute(ctx, params)
		resultChan <- struct {
			result domain.Result
			err    error
		}{result, err}
	}()

	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for result
	select {
	case res := <-resultChan:
		// Should either complete quickly or return context cancelled error
		if res.err != nil {
			assert.Contains(t, res.err.Error(), "context")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Traceroute did not respond to cancellation within timeout")
	}
}

// TestTracerouteValidation tests parameter validation in integration context
func TestTracerouteValidation(t *testing.T) {
	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        30,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     100 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	tests := []struct {
		name        string
		host        string
		options     domain.TraceOptions
		expectError bool
	}{
		{
			name: "valid parameters",
			host: "example.com",
			options: domain.TraceOptions{
				MaxHops:    15,
				Timeout:    5 * time.Second,
				PacketSize: 60,
				Queries:    3,
				IPv6:       false,
			},
			expectError: false,
		},
		{
			name: "empty host",
			host: "",
			options: domain.TraceOptions{
				MaxHops:    15,
				Timeout:    5 * time.Second,
				PacketSize: 60,
				Queries:    3,
				IPv6:       false,
			},
			expectError: true,
		},
		{
			name: "invalid max hops",
			host: "example.com",
			options: domain.TraceOptions{
				MaxHops:    0,
				Timeout:    5 * time.Second,
				PacketSize: 60,
				Queries:    3,
				IPv6:       false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := domain.NewTracerouteParameters(tt.host, tt.options)
			ctx := context.Background()

			result, err := tool.Execute(ctx, params)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				// Note: We don't assert NoError here because network operations
				// might fail for legitimate reasons in integration tests
				if err == nil {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

// TestLogger implements domain.Logger for testing
type TestLogger struct {
	messages []LogMessage
}

type LogMessage struct {
	Level  string
	Msg    string
	Fields []interface{}
}

func (l *TestLogger) Debug(msg string, fields ...interface{}) {
	l.messages = append(l.messages, LogMessage{Level: "debug", Msg: msg, Fields: fields})
}

func (l *TestLogger) Info(msg string, fields ...interface{}) {
	l.messages = append(l.messages, LogMessage{Level: "info", Msg: msg, Fields: fields})
}

func (l *TestLogger) Warn(msg string, fields ...interface{}) {
	l.messages = append(l.messages, LogMessage{Level: "warn", Msg: msg, Fields: fields})
}

func (l *TestLogger) Error(msg string, fields ...interface{}) {
	l.messages = append(l.messages, LogMessage{Level: "error", Msg: msg, Fields: fields})
}

func (l *TestLogger) Fatal(msg string, fields ...interface{}) {
	l.messages = append(l.messages, LogMessage{Level: "fatal", Msg: msg, Fields: fields})
}

func (l *TestLogger) GetMessages(level string) []string {
	var msgs []string
	for _, msg := range l.messages {
		if msg.Level == level {
			msgs = append(msgs, msg.Msg)
		}
	}
	return msgs
}

func (l *TestLogger) GetAllMessages() []LogMessage {
	return l.messages
}

// TestTracerouteResultExport tests result export functionality
func TestTracerouteResultExport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        5,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     100 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	params := domain.NewTracerouteParameters("127.0.0.1", domain.TraceOptions{
		MaxHops:    3,
		Timeout:    2 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test JSON export
	jsonData, err := result.Export(domain.ExportFormatJSON)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)
	assert.Contains(t, string(jsonData), "data")
	assert.Contains(t, string(jsonData), "metadata")

	// Test CSV export
	csvData, err := result.Export(domain.ExportFormatCSV)
	require.NoError(t, err)
	assert.NotEmpty(t, csvData)
	assert.Contains(t, string(csvData), "hop")
	assert.Contains(t, string(csvData), "hostname")

	// Test text export
	textData, err := result.Export(domain.ExportFormatText)
	require.NoError(t, err)
	assert.NotEmpty(t, textData)
	assert.Contains(t, string(textData), "NetTraceX Result")
	assert.Contains(t, string(textData), "Hop")
}

// TestTracerouteStatisticsAccuracy tests the accuracy of calculated statistics
func TestTracerouteStatisticsAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &domain.NetworkConfig{
		Timeout:        5 * time.Second,
		MaxHops:        10,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  2,
		RetryDelay:     100 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	params := domain.NewTracerouteParameters("127.0.0.1", domain.TraceOptions{
		MaxHops:    5,
		Timeout:    3 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)

	metadata := result.Metadata()
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)

	// Verify statistics consistency
	assert.Equal(t, len(hops), stats.TotalHops)
	assert.Equal(t, stats.TotalHops, stats.CompletedHops+stats.TimeoutHops)

	// Count actual completed and timeout hops
	actualCompleted := 0
	actualTimeouts := 0
	for _, hop := range hops {
		if hop.Timeout {
			actualTimeouts++
		} else {
			actualCompleted++
		}
	}

	assert.Equal(t, actualCompleted, stats.CompletedHops)
	assert.Equal(t, actualTimeouts, stats.TimeoutHops)

	// Verify success rate calculation
	expectedSuccessRate := float64(actualCompleted) / float64(len(hops)) * 100
	assert.InDelta(t, expectedSuccessRate, stats.SuccessRate, 0.01)

	// Verify reached target determination
	if len(hops) > 0 {
		lastHop := hops[len(hops)-1]
		assert.Equal(t, !lastHop.Timeout, stats.ReachedTarget)
	}
}

// Benchmark tests for performance
func BenchmarkTracerouteExecution(b *testing.B) {
	config := &domain.NetworkConfig{
		Timeout:        2 * time.Second,
		MaxHops:        5,
		PacketSize:     60,
		MaxConcurrency: 3,
		RetryAttempts:  1,
		RetryDelay:     50 * time.Millisecond,
	}

	logger := &TestLogger{messages: make([]LogMessage, 0)}
	client := network.NewClient(config, nil, logger)
	tool := NewTool(client, logger)

	params := domain.NewTracerouteParameters("127.0.0.1", domain.TraceOptions{
		MaxHops:    3,
		Timeout:    1 * time.Second,
		PacketSize: 60,
		Queries:    1,
		IPv6:       false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := tool.Execute(ctx, params)
		cancel()
		if err != nil {
			b.Fatalf("Traceroute execution failed: %v", err)
		}
	}
}

func BenchmarkTracerouteStatistics(b *testing.B) {
	tool := &Tool{}

	// Create test hops for benchmarking
	hops := make([]domain.TraceHop, 30)
	for i := 0; i < 30; i++ {
		hops[i] = domain.TraceHop{
			Number: i + 1,
			Host: domain.NetworkHost{
				Hostname:  "hop" + string(rune(i+1)) + ".example.com",
				IPAddress: net.ParseIP("192.168.1." + string(rune(i+1))),
			},
			RTT:       []time.Duration{time.Duration(i+1) * time.Millisecond, time.Duration(i+2) * time.Millisecond},
			Timeout:   i%5 == 0, // Every 5th hop times out
			Timestamp: time.Now().Add(time.Duration(i) * 100 * time.Millisecond),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.calculateStatistics(hops)
	}
}