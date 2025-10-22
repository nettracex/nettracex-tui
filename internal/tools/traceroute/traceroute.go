// Package traceroute provides traceroute diagnostic functionality
package traceroute

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Tool implements the DiagnosticTool interface for traceroute operations
type Tool struct {
	client domain.NetworkClient
	logger domain.Logger
}

// NewTool creates a new traceroute diagnostic tool
func NewTool(client domain.NetworkClient, logger domain.Logger) *Tool {
	return &Tool{
		client: client,
		logger: logger,
	}
}

// Name returns the tool name
func (t *Tool) Name() string {
	return "traceroute"
}

// Description returns the tool description
func (t *Tool) Description() string {
	return "Trace the network path to a destination host and measure hop latency"
}

// Execute performs the traceroute operation
func (t *Tool) Execute(ctx context.Context, params domain.Parameters) (domain.Result, error) {
	t.logger.Info("Executing traceroute operation", "tool", t.Name())

	// Validate parameters
	if err := t.Validate(params); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "Traceroute parameter validation failed",
			Cause:     err,
			Context:   map[string]interface{}{"params": params.ToMap()},
			Timestamp: time.Now(),
			Code:      "TRACEROUTE_VALIDATION_FAILED",
		}
	}

	// Extract parameters
	host := params.Get("host").(string)
	maxHops := params.Get("max_hops").(int)
	timeout := params.Get("timeout").(time.Duration)
	packetSize := params.Get("packet_size").(int)
	queries := params.Get("queries").(int)
	ipv6 := params.Get("ipv6").(bool)

	opts := domain.TraceOptions{
		MaxHops:    maxHops,
		Timeout:    timeout,
		PacketSize: packetSize,
		Queries:    queries,
		IPv6:       ipv6,
	}

	// Perform traceroute operation
	resultChan, err := t.client.Traceroute(ctx, host, opts)
	if err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "Traceroute operation failed",
			Cause:     err,
			Context:   map[string]interface{}{"host": host, "options": opts},
			Timestamp: time.Now(),
			Code:      "TRACEROUTE_OPERATION_FAILED",
		}
	}

	// Collect all traceroute hops
	var hops []domain.TraceHop
	for hop := range resultChan {
		hops = append(hops, hop)
		t.logger.Debug("Received hop", "number", hop.Number, "host", hop.Host.Hostname, "timeout", hop.Timeout)
	}

	// Create result with metadata
	result := domain.NewResult(hops)
	result.SetMetadata("tool", t.Name())
	result.SetMetadata("host", host)
	result.SetMetadata("max_hops", maxHops)
	result.SetMetadata("total_hops", len(hops))
	result.SetMetadata("timestamp", time.Now())

	// Calculate statistics
	stats := t.calculateStatistics(hops)
	result.SetMetadata("statistics", stats)

	t.logger.Info("Traceroute operation completed", "host", host, "hops", len(hops))
	return result, nil
}

// Validate validates the parameters for traceroute operations
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

	// Validate max_hops
	if maxHops := params.Get("max_hops"); maxHops != nil {
		if hopsInt, ok := maxHops.(int); ok && (hopsInt <= 0 || hopsInt > 255) {
			return fmt.Errorf("max_hops must be between 1 and 255")
		}
	}

	// Validate packet size
	if packetSize := params.Get("packet_size"); packetSize != nil {
		if sizeInt, ok := packetSize.(int); ok && (sizeInt <= 0 || sizeInt > 65507) {
			return fmt.Errorf("packet_size must be between 1 and 65507")
		}
	}

	// Validate queries
	if queries := params.Get("queries"); queries != nil {
		if queriesInt, ok := queries.(int); ok && (queriesInt <= 0 || queriesInt > 10) {
			return fmt.Errorf("queries must be between 1 and 10")
		}
	}

	// Validate timeout
	if timeout := params.Get("timeout"); timeout != nil {
		if timeoutDur, ok := timeout.(time.Duration); ok && timeoutDur <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
	}

	return nil
}

// GetModel returns the Bubble Tea model for the traceroute tool
func (t *Tool) GetModel() tea.Model {
	return NewModel(t)
}

// TracerouteStatistics contains calculated traceroute statistics
type TracerouteStatistics struct {
	TotalHops       int           `json:"total_hops"`
	CompletedHops   int           `json:"completed_hops"`
	TimeoutHops     int           `json:"timeout_hops"`
	SuccessRate     float64       `json:"success_rate_percent"`
	MinRTT          time.Duration `json:"min_rtt"`
	MaxRTT          time.Duration `json:"max_rtt"`
	AvgRTT          time.Duration `json:"avg_rtt"`
	TotalTime       time.Duration `json:"total_time"`
	ReachedTarget   bool          `json:"reached_target"`
	FinalHop        int           `json:"final_hop"`
}

// calculateStatistics calculates traceroute statistics from hops
func (t *Tool) calculateStatistics(hops []domain.TraceHop) TracerouteStatistics {
	stats := TracerouteStatistics{
		TotalHops: len(hops),
	}

	if len(hops) == 0 {
		return stats
	}

	var validHops []domain.TraceHop
	var allRTTs []time.Duration
	var startTime, endTime time.Time

	// Find first and last timestamps
	startTime = hops[0].Timestamp
	endTime = hops[0].Timestamp

	for _, hop := range hops {
		if hop.Timestamp.Before(startTime) {
			startTime = hop.Timestamp
		}
		if hop.Timestamp.After(endTime) {
			endTime = hop.Timestamp
		}

		if hop.Timeout {
			stats.TimeoutHops++
		} else {
			validHops = append(validHops, hop)
			stats.CompletedHops++
			
			// Collect all RTT measurements for this hop
			for _, rtt := range hop.RTT {
				allRTTs = append(allRTTs, rtt)
			}
		}
	}

	stats.TotalTime = endTime.Sub(startTime)
	stats.FinalHop = len(hops)

	if stats.TotalHops > 0 {
		stats.SuccessRate = float64(stats.CompletedHops) / float64(stats.TotalHops) * 100
	}

	// Calculate RTT statistics from all measurements
	if len(allRTTs) > 0 {
		stats.MinRTT = allRTTs[0]
		stats.MaxRTT = allRTTs[0]
		var totalRTT time.Duration

		for _, rtt := range allRTTs {
			totalRTT += rtt
			if rtt < stats.MinRTT {
				stats.MinRTT = rtt
			}
			if rtt > stats.MaxRTT {
				stats.MaxRTT = rtt
			}
		}

		stats.AvgRTT = totalRTT / time.Duration(len(allRTTs))
	}

	// Determine if we reached the target (last hop is not a timeout)
	if len(hops) > 0 {
		lastHop := hops[len(hops)-1]
		stats.ReachedTarget = !lastHop.Timeout
	}

	return stats
}

// FormatTracerouteStatistics formats traceroute statistics for display
func FormatTracerouteStatistics(stats TracerouteStatistics) string {
	return fmt.Sprintf(
		"--- Traceroute Statistics ---\n"+
			"Hops: Total = %d, Completed = %d, Timeouts = %d (%.1f%% success)\n"+
			"Round-trip times: Min = %v, Max = %v, Avg = %v\n"+
			"Total time: %v, Final hop: %d, Reached target: %t",
		stats.TotalHops,
		stats.CompletedHops,
		stats.TimeoutHops,
		stats.SuccessRate,
		stats.MinRTT,
		stats.MaxRTT,
		stats.AvgRTT,
		stats.TotalTime,
		stats.FinalHop,
		stats.ReachedTarget,
	)
}

// GetHopSummary returns a formatted summary of a single hop
func GetHopSummary(hop domain.TraceHop) string {
	if hop.Timeout {
		return fmt.Sprintf("%2d  * * * Request timed out", hop.Number)
	}

	var rttStrs []string
	for _, rtt := range hop.RTT {
		rttStrs = append(rttStrs, fmt.Sprintf("%.3f ms", float64(rtt.Nanoseconds())/1000000.0))
	}

	hostname := hop.Host.Hostname
	if hostname == "" {
		hostname = hop.Host.IPAddress.String()
	}

	return fmt.Sprintf("%2d  %s (%s)  %s",
		hop.Number,
		hostname,
		hop.Host.IPAddress.String(),
		fmt.Sprintf("[%s]", fmt.Sprintf("%s", rttStrs)))
}

// IsPrivateIP checks if an IP address is in a private range
func IsPrivateIP(ip string) bool {
	// This is a simplified check - in production you'd use net.IP methods
	// Common private IP ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	if len(ip) == 0 {
		return false
	}
	
	// Simple string-based check for common private ranges
	return (len(ip) >= 3 && ip[:3] == "10.") ||
		   (len(ip) >= 7 && ip[:7] == "172.16.") ||
		   (len(ip) >= 8 && ip[:8] == "192.168.") ||
		   (len(ip) >= 9 && ip[:9] == "127.0.0.1")
}