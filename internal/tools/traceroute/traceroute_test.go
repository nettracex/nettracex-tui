// Package traceroute provides unit tests for traceroute functionality
package traceroute

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// We'll use the existing MockClient from the network package

// MockLogger implements domain.Logger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLogger) Fatal(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func TestNewTool(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}

	tool := NewTool(mockClient, mockLogger)

	assert.NotNil(t, tool)
	assert.Equal(t, "traceroute", tool.Name())
	assert.Equal(t, "Trace the network path to a destination host and measure hop latency", tool.Description())
	assert.Equal(t, mockClient, tool.client)
	assert.Equal(t, mockLogger, tool.logger)
}

func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	assert.Equal(t, "traceroute", tool.Name())
}

func TestTool_Description(t *testing.T) {
	tool := &Tool{}
	expected := "Trace the network path to a destination host and measure hop latency"
	assert.Equal(t, expected, tool.Description())
}

func TestTool_Validate(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name        string
		params      domain.Parameters
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid parameters",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("max_hops", 30)
				p.Set("timeout", 5*time.Second)
				p.Set("packet_size", 60)
				p.Set("queries", 3)
				p.Set("ipv6", false)
				return p
			}(),
			expectError: false,
		},
		{
			name: "missing host parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("max_hops", 30)
				return p
			}(),
			expectError: true,
			errorMsg:    "host parameter is required",
		},
		{
			name: "empty host parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "")
				return p
			}(),
			expectError: true,
			errorMsg:    "host parameter cannot be empty",
		},
		{
			name: "invalid host type",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", 123)
				return p
			}(),
			expectError: true,
			errorMsg:    "host parameter must be a string",
		},
		{
			name: "invalid max_hops - zero",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("max_hops", 0)
				return p
			}(),
			expectError: true,
			errorMsg:    "max_hops must be between 1 and 255",
		},
		{
			name: "invalid max_hops - too large",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("max_hops", 256)
				return p
			}(),
			expectError: true,
			errorMsg:    "max_hops must be between 1 and 255",
		},
		{
			name: "invalid packet_size - zero",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("packet_size", 0)
				return p
			}(),
			expectError: true,
			errorMsg:    "packet_size must be between 1 and 65507",
		},
		{
			name: "invalid packet_size - too large",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("packet_size", 65508)
				return p
			}(),
			expectError: true,
			errorMsg:    "packet_size must be between 1 and 65507",
		},
		{
			name: "invalid queries - zero",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("queries", 0)
				return p
			}(),
			expectError: true,
			errorMsg:    "queries must be between 1 and 10",
		},
		{
			name: "invalid queries - too large",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("queries", 11)
				return p
			}(),
			expectError: true,
			errorMsg:    "queries must be between 1 and 10",
		},
		{
			name: "invalid timeout - negative",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "example.com")
				p.Set("timeout", -1*time.Second)
				return p
			}(),
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.Validate(tt.params)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTool_Execute_Success(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Create test hops
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
				Hostname:  "router2.example.com",
				IPAddress: net.ParseIP("10.0.0.1"),
			},
			RTT:       []time.Duration{25 * time.Millisecond, 27 * time.Millisecond, 26 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
		{
			Number: 3,
			Host: domain.NetworkHost{
				Hostname:  "example.com",
				IPAddress: net.ParseIP("93.184.216.34"),
			},
			RTT:       []time.Duration{45 * time.Millisecond, 47 * time.Millisecond, 46 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
	}

	// Set up mock client with test hops
	mockClient.SetTraceResponse("example.com", testHops)

	// Set up mock logger expectations
	mockLogger.On("Info", "Executing traceroute operation", mock.Anything).Return()
	mockLogger.On("Debug", "Received hop", mock.Anything).Return()
	mockLogger.On("Info", "Traceroute operation completed", mock.Anything).Return()

	// Create parameters
	params := domain.NewTracerouteParameters("example.com", domain.TraceOptions{
		MaxHops:    30,
		Timeout:    5 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	// Execute traceroute
	ctx := context.Background()
	result, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check result data
	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)
	assert.Len(t, hops, 3)

	// Check metadata
	metadata := result.Metadata()
	assert.Equal(t, "traceroute", metadata["tool"])
	assert.Equal(t, "example.com", metadata["host"])
	assert.Equal(t, 30, metadata["max_hops"])
	assert.Equal(t, 3, metadata["total_hops"])

	// Check statistics
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)
	assert.Equal(t, 3, stats.TotalHops)
	assert.Equal(t, 3, stats.CompletedHops)
	assert.Equal(t, 0, stats.TimeoutHops)
	assert.Equal(t, 100.0, stats.SuccessRate)
	assert.True(t, stats.ReachedTarget)

	// Verify mock logger expectations
	mockLogger.AssertExpectations(t)
}

func TestTool_Execute_WithTimeouts(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Create test hops with timeouts
	testHops := []domain.TraceHop{
		{
			Number:    1,
			Host:      domain.NetworkHost{Hostname: "router1.example.com", IPAddress: net.ParseIP("192.168.1.1")},
			RTT:       []time.Duration{10 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
		{
			Number:    2,
			Host:      domain.NetworkHost{},
			RTT:       []time.Duration{},
			Timeout:   true,
			Timestamp: time.Now(),
		},
		{
			Number:    3,
			Host:      domain.NetworkHost{Hostname: "example.com", IPAddress: net.ParseIP("93.184.216.34")},
			RTT:       []time.Duration{45 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
	}

	// Set up mock client with test hops
	mockClient.SetTraceResponse("example.com", testHops)

	// Set up mock expectations
	mockLogger.On("Info", "Executing traceroute operation", mock.Anything).Return()
	mockLogger.On("Debug", "Received hop", mock.Anything).Return()
	mockLogger.On("Info", "Traceroute operation completed", mock.Anything).Return()

	// Create parameters
	params := domain.NewTracerouteParameters("example.com", domain.TraceOptions{
		MaxHops:    30,
		Timeout:    5 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	// Execute traceroute
	ctx := context.Background()
	result, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check statistics
	metadata := result.Metadata()
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)
	assert.Equal(t, 3, stats.TotalHops)
	assert.Equal(t, 2, stats.CompletedHops)
	assert.Equal(t, 1, stats.TimeoutHops)
	assert.InDelta(t, 66.7, stats.SuccessRate, 0.1)
	assert.True(t, stats.ReachedTarget) // Last hop was successful

	// Verify mock expectations
	mockLogger.AssertExpectations(t)
}

func TestTool_Execute_ValidationError(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Set up mock expectations
	mockLogger.On("Info", "Executing traceroute operation", mock.Anything).Return()

	// Create invalid parameters (empty host)
	params := domain.NewParameters()
	params.Set("host", "")

	// Execute traceroute
	ctx := context.Background()
	result, err := tool.Execute(ctx, params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, result)

	netErr, ok := err.(*domain.NetTraceError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrorTypeValidation, netErr.Type)
	assert.Equal(t, "TRACEROUTE_VALIDATION_FAILED", netErr.Code)

	// Verify mock expectations
	mockLogger.AssertExpectations(t)
}

func TestTool_Execute_NetworkError(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	// Set up mock expectations
	mockLogger.On("Info", "Executing traceroute operation", mock.Anything).Return()

	networkErr := &domain.NetTraceError{
		Type:    domain.ErrorTypeNetwork,
		Message: "network unreachable",
		Code:    "NETWORK_UNREACHABLE",
	}

	mockClient.SetTraceError("example.com", networkErr)

	// Create parameters
	params := domain.NewTracerouteParameters("example.com", domain.TraceOptions{
		MaxHops:    30,
		Timeout:    5 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	// Execute traceroute
	ctx := context.Background()
	result, err := tool.Execute(ctx, params)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, result)

	netErr, ok := err.(*domain.NetTraceError)
	require.True(t, ok)
	assert.Equal(t, domain.ErrorTypeNetwork, netErr.Type)
	assert.Equal(t, "TRACEROUTE_OPERATION_FAILED", netErr.Code)

	// Verify mock expectations
	mockLogger.AssertExpectations(t)
}

func TestTool_GetModel(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

	model := tool.GetModel()
	assert.NotNil(t, model)

	// Verify it's the correct type
	tracerouteModel, ok := model.(*Model)
	assert.True(t, ok)
	assert.Equal(t, tool, tracerouteModel.tool)
}

func TestCalculateStatistics(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name     string
		hops     []domain.TraceHop
		expected TracerouteStatistics
	}{
		{
			name: "empty hops",
			hops: []domain.TraceHop{},
			expected: TracerouteStatistics{
				TotalHops:     0,
				CompletedHops: 0,
				TimeoutHops:   0,
				SuccessRate:   0,
				ReachedTarget: false,
				FinalHop:      0,
			},
		},
		{
			name: "all successful hops",
			hops: []domain.TraceHop{
				{
					Number:    1,
					RTT:       []time.Duration{10 * time.Millisecond, 12 * time.Millisecond},
					Timeout:   false,
					Timestamp: time.Now(),
				},
				{
					Number:    2,
					RTT:       []time.Duration{20 * time.Millisecond, 22 * time.Millisecond},
					Timeout:   false,
					Timestamp: time.Now().Add(1 * time.Second),
				},
			},
			expected: TracerouteStatistics{
				TotalHops:     2,
				CompletedHops: 2,
				TimeoutHops:   0,
				SuccessRate:   100.0,
				MinRTT:        10 * time.Millisecond,
				MaxRTT:        22 * time.Millisecond,
				AvgRTT:        16 * time.Millisecond,
				ReachedTarget: true,
				FinalHop:      2,
			},
		},
		{
			name: "mixed successful and timeout hops",
			hops: []domain.TraceHop{
				{
					Number:    1,
					RTT:       []time.Duration{10 * time.Millisecond},
					Timeout:   false,
					Timestamp: time.Now(),
				},
				{
					Number:    2,
					RTT:       []time.Duration{},
					Timeout:   true,
					Timestamp: time.Now().Add(1 * time.Second),
				},
				{
					Number:    3,
					RTT:       []time.Duration{30 * time.Millisecond},
					Timeout:   false,
					Timestamp: time.Now().Add(2 * time.Second),
				},
			},
			expected: TracerouteStatistics{
				TotalHops:     3,
				CompletedHops: 2,
				TimeoutHops:   1,
				SuccessRate:   66.66666666666667,
				MinRTT:        10 * time.Millisecond,
				MaxRTT:        30 * time.Millisecond,
				AvgRTT:        20 * time.Millisecond,
				ReachedTarget: true, // Last hop was successful
				FinalHop:      3,
			},
		},
		{
			name: "ending with timeout",
			hops: []domain.TraceHop{
				{
					Number:    1,
					RTT:       []time.Duration{10 * time.Millisecond},
					Timeout:   false,
					Timestamp: time.Now(),
				},
				{
					Number:    2,
					RTT:       []time.Duration{},
					Timeout:   true,
					Timestamp: time.Now().Add(1 * time.Second),
				},
			},
			expected: TracerouteStatistics{
				TotalHops:     2,
				CompletedHops: 1,
				TimeoutHops:   1,
				SuccessRate:   50.0,
				MinRTT:        10 * time.Millisecond,
				MaxRTT:        10 * time.Millisecond,
				AvgRTT:        10 * time.Millisecond,
				ReachedTarget: false, // Last hop was timeout
				FinalHop:      2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tool.calculateStatistics(tt.hops)

			assert.Equal(t, tt.expected.TotalHops, stats.TotalHops)
			assert.Equal(t, tt.expected.CompletedHops, stats.CompletedHops)
			assert.Equal(t, tt.expected.TimeoutHops, stats.TimeoutHops)
			assert.InDelta(t, tt.expected.SuccessRate, stats.SuccessRate, 0.001)
			assert.Equal(t, tt.expected.ReachedTarget, stats.ReachedTarget)
			assert.Equal(t, tt.expected.FinalHop, stats.FinalHop)

			if tt.expected.MinRTT > 0 {
				assert.Equal(t, tt.expected.MinRTT, stats.MinRTT)
				assert.Equal(t, tt.expected.MaxRTT, stats.MaxRTT)
				assert.Equal(t, tt.expected.AvgRTT, stats.AvgRTT)
			}
		})
	}
}

func TestFormatTracerouteStatistics(t *testing.T) {
	stats := TracerouteStatistics{
		TotalHops:     5,
		CompletedHops: 4,
		TimeoutHops:   1,
		SuccessRate:   80.0,
		MinRTT:        10 * time.Millisecond,
		MaxRTT:        50 * time.Millisecond,
		AvgRTT:        25 * time.Millisecond,
		TotalTime:     5 * time.Second,
		ReachedTarget: true,
		FinalHop:      5,
	}

	formatted := FormatTracerouteStatistics(stats)

	assert.Contains(t, formatted, "--- Traceroute Statistics ---")
	assert.Contains(t, formatted, "Total = 5")
	assert.Contains(t, formatted, "Completed = 4")
	assert.Contains(t, formatted, "Timeouts = 1")
	assert.Contains(t, formatted, "80.0% success")
	assert.Contains(t, formatted, "Min = 10ms")
	assert.Contains(t, formatted, "Max = 50ms")
	assert.Contains(t, formatted, "Avg = 25ms")
	assert.Contains(t, formatted, "Total time: 5s")
	assert.Contains(t, formatted, "Final hop: 5")
	assert.Contains(t, formatted, "Reached target: true")
}

func TestGetHopSummary(t *testing.T) {
	tests := []struct {
		name     string
		hop      domain.TraceHop
		expected string
	}{
		{
			name: "timeout hop",
			hop: domain.TraceHop{
				Number:  1,
				Timeout: true,
			},
			expected: " 1  * * * Request timed out",
		},
		{
			name: "successful hop with hostname",
			hop: domain.TraceHop{
				Number: 2,
				Host: domain.NetworkHost{
					Hostname:  "router.example.com",
					IPAddress: net.ParseIP("192.168.1.1"),
				},
				RTT:     []time.Duration{10 * time.Millisecond, 12 * time.Millisecond},
				Timeout: false,
			},
			expected: " 2  router.example.com (192.168.1.1)  [[10.000 ms 12.000 ms]]",
		},
		{
			name: "successful hop without hostname",
			hop: domain.TraceHop{
				Number: 3,
				Host: domain.NetworkHost{
					IPAddress: net.ParseIP("10.0.0.1"),
				},
				RTT:     []time.Duration{25 * time.Millisecond},
				Timeout: false,
			},
			expected: " 3  10.0.0.1 (10.0.0.1)  [[25.000 ms]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetHopSummary(tt.hop)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"127.0.0.1", true},
		{"8.8.8.8", false},
		{"93.184.216.34", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := IsPrivateIP(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test with real network client (using mock for network operations)
func TestTool_Integration(t *testing.T) {
	// Create a mock network client that simulates real traceroute behavior
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}

	// Set up mock expectations for a realistic traceroute
	testHops := []domain.TraceHop{
		{
			Number: 1,
			Host: domain.NetworkHost{
				Hostname:  "gateway.local",
				IPAddress: net.ParseIP("192.168.1.1"),
			},
			RTT:       []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 1 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now(),
		},
		{
			Number: 2,
			Host: domain.NetworkHost{
				Hostname:  "isp-router.example.net",
				IPAddress: net.ParseIP("10.1.1.1"),
			},
			RTT:       []time.Duration{15 * time.Millisecond, 16 * time.Millisecond, 14 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now().Add(100 * time.Millisecond),
		},
		{
			Number:    3,
			Host:      domain.NetworkHost{},
			RTT:       []time.Duration{},
			Timeout:   true,
			Timestamp: time.Now().Add(200 * time.Millisecond),
		},
		{
			Number: 4,
			Host: domain.NetworkHost{
				Hostname:  "example.com",
				IPAddress: net.ParseIP("93.184.216.34"),
			},
			RTT:       []time.Duration{45 * time.Millisecond, 47 * time.Millisecond, 46 * time.Millisecond},
			Timeout:   false,
			Timestamp: time.Now().Add(300 * time.Millisecond),
		},
	}

	mockClient.SetTraceResponse("example.com", testHops)

	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()

	// Create tool and execute
	tool := NewTool(mockClient, mockLogger)
	params := domain.NewTracerouteParameters("example.com", domain.TraceOptions{
		MaxHops:    30,
		Timeout:    5 * time.Second,
		PacketSize: 60,
		Queries:    3,
		IPv6:       false,
	})

	ctx := context.Background()
	result, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, result)

	hops, ok := result.Data().([]domain.TraceHop)
	require.True(t, ok)
	assert.Len(t, hops, 4)

	// Verify statistics
	metadata := result.Metadata()
	stats, ok := metadata["statistics"].(TracerouteStatistics)
	require.True(t, ok)
	assert.Equal(t, 4, stats.TotalHops)
	assert.Equal(t, 3, stats.CompletedHops)
	assert.Equal(t, 1, stats.TimeoutHops)
	assert.Equal(t, 75.0, stats.SuccessRate)
	assert.True(t, stats.ReachedTarget)

	// Verify mock expectations
	mockLogger.AssertExpectations(t)
}