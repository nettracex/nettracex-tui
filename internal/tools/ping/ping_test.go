// Package ping provides tests for ping diagnostic functionality
package ping

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
)

// TestTool_Name tests the tool name
func TestTool_Name(t *testing.T) {
	tool := &Tool{}
	expected := "ping"
	if got := tool.Name(); got != expected {
		t.Errorf("Tool.Name() = %v, want %v", got, expected)
	}
}

// TestTool_Description tests the tool description
func TestTool_Description(t *testing.T) {
	tool := &Tool{}
	description := tool.Description()
	if description == "" {
		t.Error("Tool.Description() should not be empty")
	}
}

// TestTool_Validate tests parameter validation
func TestTool_Validate(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name    string
		params  domain.Parameters
		wantErr bool
	}{
		{
			name:    "missing host parameter",
			params:  domain.NewParameters(),
			wantErr: true,
		},
		{
			name: "empty host parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "")
				return p
			}(),
			wantErr: true,
		},
		{
			name: "valid host parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "google.com")
				return p
			}(),
			wantErr: false,
		},
		{
			name: "invalid count parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "google.com")
				p.Set("count", -1)
				return p
			}(),
			wantErr: true,
		},
		{
			name: "invalid packet size parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "google.com")
				p.Set("packet_size", 70000)
				return p
			}(),
			wantErr: true,
		},
		{
			name: "invalid TTL parameter",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "google.com")
				p.Set("ttl", 300)
				return p
			}(),
			wantErr: true,
		},
		{
			name: "valid parameters",
			params: func() domain.Parameters {
				p := domain.NewParameters()
				p.Set("host", "google.com")
				p.Set("count", 4)
				p.Set("packet_size", 64)
				p.Set("ttl", 64)
				return p
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.Validate(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Tool.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// MockLogger implements a simple logger for testing
type MockLogger struct{}

func (l *MockLogger) Debug(msg string, fields ...interface{}) {}
func (l *MockLogger) Info(msg string, fields ...interface{})  {}
func (l *MockLogger) Warn(msg string, fields ...interface{})  {}
func (l *MockLogger) Error(msg string, fields ...interface{}) {}
func (l *MockLogger) Fatal(msg string, fields ...interface{}) {}

// TestTool_Execute tests ping execution with mock client
func TestTool_Execute(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &MockLogger{}
	tool := NewTool(mockClient, mockLogger)

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
			RTT:        12 * time.Millisecond,
			TTL:        64,
			PacketSize: 64,
			Timestamp:  time.Now(),
		},
	}

	mockClient.SetPingResponse("google.com", mockResults)

	// Create parameters
	params := domain.NewPingParameters("google.com", domain.PingOptions{
		Count:      2,
		Interval:   time.Second,
		Timeout:    5 * time.Second,
		PacketSize: 64,
		TTL:        64,
		IPv6:       false,
	})

	// Execute ping
	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Tool.Execute() error = %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Tool.Execute() returned nil result")
	}

	// Check result data
	data := result.Data()
	results, ok := data.([]domain.PingResult)
	if !ok {
		t.Fatalf("Tool.Execute() returned wrong data type: %T", data)
	}

	if len(results) != 2 {
		t.Errorf("Tool.Execute() returned %d results, want 2", len(results))
	}

	// Check metadata
	metadata := result.Metadata()
	if metadata["tool"] != "ping" {
		t.Errorf("Tool.Execute() metadata tool = %v, want ping", metadata["tool"])
	}

	if metadata["host"] != "google.com" {
		t.Errorf("Tool.Execute() metadata host = %v, want google.com", metadata["host"])
	}

	// Check statistics
	stats, ok := metadata["statistics"].(PingStatistics)
	if !ok {
		t.Fatal("Tool.Execute() metadata missing statistics")
	}

	if stats.PacketsSent != 2 {
		t.Errorf("Statistics PacketsSent = %d, want 2", stats.PacketsSent)
	}

	if stats.PacketsReceived != 2 {
		t.Errorf("Statistics PacketsReceived = %d, want 2", stats.PacketsReceived)
	}
}

// TestCalculateStatistics tests ping statistics calculation
func TestCalculateStatistics(t *testing.T) {
	tool := &Tool{}

	tests := []struct {
		name     string
		results  []domain.PingResult
		expected PingStatistics
	}{
		{
			name:    "empty results",
			results: []domain.PingResult{},
			expected: PingStatistics{
				PacketsSent: 0,
			},
		},
		{
			name: "all successful pings",
			results: []domain.PingResult{
				{
					Sequence:  1,
					RTT:       10 * time.Millisecond,
					Timestamp: time.Now(),
				},
				{
					Sequence:  2,
					RTT:       20 * time.Millisecond,
					Timestamp: time.Now().Add(time.Second),
				},
			},
			expected: PingStatistics{
				PacketsSent:     2,
				PacketsReceived: 2,
				PacketLoss:      0.0,
				MinRTT:          10 * time.Millisecond,
				MaxRTT:          20 * time.Millisecond,
				AvgRTT:          15 * time.Millisecond,
			},
		},
		{
			name: "mixed success and failure",
			results: []domain.PingResult{
				{
					Sequence:  1,
					RTT:       10 * time.Millisecond,
					Timestamp: time.Now(),
				},
				{
					Sequence:  2,
					Error:     fmt.Errorf("timeout"),
					Timestamp: time.Now().Add(time.Second),
				},
			},
			expected: PingStatistics{
				PacketsSent:     2,
				PacketsReceived: 1,
				PacketLoss:      50.0,
				MinRTT:          10 * time.Millisecond,
				MaxRTT:          10 * time.Millisecond,
				AvgRTT:          10 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tool.calculateStatistics(tt.results)

			if stats.PacketsSent != tt.expected.PacketsSent {
				t.Errorf("PacketsSent = %d, want %d", stats.PacketsSent, tt.expected.PacketsSent)
			}

			if stats.PacketsReceived != tt.expected.PacketsReceived {
				t.Errorf("PacketsReceived = %d, want %d", stats.PacketsReceived, tt.expected.PacketsReceived)
			}

			if stats.PacketLoss != tt.expected.PacketLoss {
				t.Errorf("PacketLoss = %f, want %f", stats.PacketLoss, tt.expected.PacketLoss)
			}

			if len(tt.results) > 0 && tt.expected.PacketsReceived > 0 {
				if stats.MinRTT != tt.expected.MinRTT {
					t.Errorf("MinRTT = %v, want %v", stats.MinRTT, tt.expected.MinRTT)
				}

				if stats.MaxRTT != tt.expected.MaxRTT {
					t.Errorf("MaxRTT = %v, want %v", stats.MaxRTT, tt.expected.MaxRTT)
				}

				if stats.AvgRTT != tt.expected.AvgRTT {
					t.Errorf("AvgRTT = %v, want %v", stats.AvgRTT, tt.expected.AvgRTT)
				}
			}
		})
	}
}

// TestFormatPingStatistics tests statistics formatting
func TestFormatPingStatistics(t *testing.T) {
	stats := PingStatistics{
		PacketsSent:     4,
		PacketsReceived: 3,
		PacketLoss:      25.0,
		MinRTT:          10 * time.Millisecond,
		MaxRTT:          30 * time.Millisecond,
		AvgRTT:          20 * time.Millisecond,
		TotalTime:       3 * time.Second,
	}

	formatted := FormatPingStatistics(stats)
	if formatted == "" {
		t.Error("FormatPingStatistics() returned empty string")
	}

	// Check that key information is included
	expectedStrings := []string{
		"Packets: Sent = 4",
		"Received = 3",
		"Lost = 1",
		"25.0% loss",
		"Min = 10ms",
		"Max = 30ms",
		"Avg = 20ms",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(formatted, expected) {
			t.Errorf("FormatPingStatistics() missing expected string: %s", expected)
		}
	}
}