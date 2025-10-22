// Package ping provides ping diagnostic functionality
package ping

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Tool implements the DiagnosticTool interface for ping operations
type Tool struct {
	client domain.NetworkClient
	logger domain.Logger
}

// NewTool creates a new ping diagnostic tool
func NewTool(client domain.NetworkClient, logger domain.Logger) *Tool {
	return &Tool{
		client: client,
		logger: logger,
	}
}

// Name returns the tool name
func (t *Tool) Name() string {
	return "ping"
}

// Description returns the tool description
func (t *Tool) Description() string {
	return "Test network connectivity and measure round-trip time to hosts"
}

// Execute performs the ping operation
func (t *Tool) Execute(ctx context.Context, params domain.Parameters) (domain.Result, error) {
	t.logger.Info("Executing ping operation", "tool", t.Name())

	// Validate parameters
	if err := t.Validate(params); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "Ping parameter validation failed",
			Cause:     err,
			Context:   map[string]interface{}{"params": params.ToMap()},
			Timestamp: time.Now(),
			Code:      "PING_VALIDATION_FAILED",
		}
	}

	// Extract parameters
	host := params.Get("host").(string)
	count := params.Get("count").(int)
	interval := params.Get("interval").(time.Duration)
	timeout := params.Get("timeout").(time.Duration)
	packetSize := params.Get("packet_size").(int)
	ttl := params.Get("ttl").(int)
	ipv6 := params.Get("ipv6").(bool)

	opts := domain.PingOptions{
		Count:      count,
		Interval:   interval,
		Timeout:    timeout,
		PacketSize: packetSize,
		TTL:        ttl,
		IPv6:       ipv6,
	}

	// Perform ping operation
	resultChan, err := t.client.Ping(ctx, host, opts)
	if err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "Ping operation failed",
			Cause:     err,
			Context:   map[string]interface{}{"host": host, "options": opts},
			Timestamp: time.Now(),
			Code:      "PING_OPERATION_FAILED",
		}
	}

	// Collect all ping results
	var results []domain.PingResult
	for result := range resultChan {
		results = append(results, result)
	}

	// Create result with metadata
	result := domain.NewResult(results)
	result.SetMetadata("tool", t.Name())
	result.SetMetadata("host", host)
	result.SetMetadata("count", count)
	result.SetMetadata("timestamp", time.Now())

	// Calculate statistics
	stats := t.calculateStatistics(results)
	result.SetMetadata("statistics", stats)

	t.logger.Info("Ping operation completed", "host", host, "count", len(results))
	return result, nil
}

// Validate validates the parameters for ping operations
func (t *Tool) Validate(params domain.Parameters) error {
	host := params.Get("host")
	if host == nil {
		return fmt.Errorf("host parameter is required")
	}

	hostStr, ok := host.(string)
	if !ok {
		return fmt.Errorf("host parameter must be a string")
	}

	if hostStr == "" {
		return fmt.Errorf("host parameter cannot be empty")
	}

	// Validate count
	if count := params.Get("count"); count != nil {
		if countInt, ok := count.(int); ok && countInt <= 0 {
			return fmt.Errorf("count must be positive")
		}
	}

	// Validate packet size
	if packetSize := params.Get("packet_size"); packetSize != nil {
		if sizeInt, ok := packetSize.(int); ok && (sizeInt <= 0 || sizeInt > 65507) {
			return fmt.Errorf("packet_size must be between 1 and 65507")
		}
	}

	// Validate TTL
	if ttl := params.Get("ttl"); ttl != nil {
		if ttlInt, ok := ttl.(int); ok && (ttlInt <= 0 || ttlInt > 255) {
			return fmt.Errorf("ttl must be between 1 and 255")
		}
	}

	return nil
}

// GetModel returns the Bubble Tea model for the ping tool
func (t *Tool) GetModel() tea.Model {
	return NewModel(t)
}

// PingStatistics contains calculated ping statistics
type PingStatistics struct {
	PacketsSent     int           `json:"packets_sent"`
	PacketsReceived int           `json:"packets_received"`
	PacketLoss      float64       `json:"packet_loss_percent"`
	MinRTT          time.Duration `json:"min_rtt"`
	MaxRTT          time.Duration `json:"max_rtt"`
	AvgRTT          time.Duration `json:"avg_rtt"`
	StdDevRTT       time.Duration `json:"stddev_rtt"`
	TotalTime       time.Duration `json:"total_time"`
}

// calculateStatistics calculates ping statistics from results
func (t *Tool) calculateStatistics(results []domain.PingResult) PingStatistics {
	stats := PingStatistics{
		PacketsSent: len(results),
	}

	if len(results) == 0 {
		return stats
	}

	var validResults []domain.PingResult
	var totalRTT time.Duration
	var minRTT, maxRTT time.Duration
	var startTime, endTime time.Time

	// Find first and last timestamps
	startTime = results[0].Timestamp
	endTime = results[0].Timestamp

	for _, result := range results {
		if result.Timestamp.Before(startTime) {
			startTime = result.Timestamp
		}
		if result.Timestamp.After(endTime) {
			endTime = result.Timestamp
		}

		// Only count successful pings
		if result.Error == nil {
			validResults = append(validResults, result)
			totalRTT += result.RTT

			if len(validResults) == 1 {
				minRTT = result.RTT
				maxRTT = result.RTT
			} else {
				if result.RTT < minRTT {
					minRTT = result.RTT
				}
				if result.RTT > maxRTT {
					maxRTT = result.RTT
				}
			}
		}
	}

	stats.PacketsReceived = len(validResults)
	stats.TotalTime = endTime.Sub(startTime)

	if stats.PacketsSent > 0 {
		stats.PacketLoss = float64(stats.PacketsSent-stats.PacketsReceived) / float64(stats.PacketsSent) * 100
	}

	if len(validResults) > 0 {
		stats.MinRTT = minRTT
		stats.MaxRTT = maxRTT
		stats.AvgRTT = totalRTT / time.Duration(len(validResults))

		// Calculate standard deviation
		var variance time.Duration
		for _, result := range validResults {
			diff := result.RTT - stats.AvgRTT
			variance += diff * diff / time.Duration(len(validResults))
		}
		stats.StdDevRTT = time.Duration(float64(variance) * 0.5) // Approximate square root
	}

	return stats
}

// FormatPingStatistics formats ping statistics for display
func FormatPingStatistics(stats PingStatistics) string {
	return fmt.Sprintf(
		"--- Ping Statistics ---\n"+
			"Packets: Sent = %d, Received = %d, Lost = %d (%.1f%% loss)\n"+
			"Round-trip times: Min = %v, Max = %v, Avg = %v\n"+
			"Total time: %v",
		stats.PacketsSent,
		stats.PacketsReceived,
		stats.PacketsSent-stats.PacketsReceived,
		stats.PacketLoss,
		stats.MinRTT,
		stats.MaxRTT,
		stats.AvgRTT,
		stats.TotalTime,
	)
}