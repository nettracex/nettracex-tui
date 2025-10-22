// Package domain contains result implementations
package domain

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// BaseResult provides a basic implementation of the Result interface
type BaseResult struct {
	data     interface{}
	metadata map[string]interface{}
}

// NewResult creates a new BaseResult instance
func NewResult(data interface{}) *BaseResult {
	return &BaseResult{
		data:     data,
		metadata: make(map[string]interface{}),
	}
}

// Data returns the result data
func (r *BaseResult) Data() interface{} {
	return r.data
}

// Metadata returns the result metadata
func (r *BaseResult) Metadata() map[string]interface{} {
	return r.metadata
}

// SetMetadata sets a metadata value
func (r *BaseResult) SetMetadata(key string, value interface{}) {
	r.metadata[key] = value
}

// Format formats the result using the provided formatter
func (r *BaseResult) Format(formatter OutputFormatter) string {
	return formatter.Format(r.data)
}

// Export exports the result in the specified format
func (r *BaseResult) Export(format ExportFormat) ([]byte, error) {
	switch format {
	case ExportFormatJSON:
		return r.exportJSON()
	case ExportFormatCSV:
		return r.exportCSV()
	case ExportFormatText:
		return r.exportText()
	default:
		return nil, fmt.Errorf("unsupported export format: %d", format)
	}
}

// exportJSON exports the result as JSON
func (r *BaseResult) exportJSON() ([]byte, error) {
	exportData := map[string]interface{}{
		"data":      r.data,
		"metadata":  r.metadata,
		"timestamp": time.Now(),
	}
	return json.MarshalIndent(exportData, "", "  ")
}

// exportCSV exports the result as CSV
func (r *BaseResult) exportCSV() ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write metadata as header comments
	for key, value := range r.metadata {
		writer.Write([]string{fmt.Sprintf("# %s: %v", key, value)})
	}
	
	// Convert data to CSV format based on type
	switch data := r.data.(type) {
	case []PingResult:
		return r.exportPingResultsCSV(data)
	case []TraceHop:
		return r.exportTraceHopsCSV(data)
	case DNSResult:
		return r.exportDNSResultCSV(data)
	case WHOISResult:
		return r.exportWHOISResultCSV(data)
	case SSLResult:
		return r.exportSSLResultCSV(data)
	default:
		// Fallback to JSON for unknown types
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		writer.Write([]string{"data"})
		writer.Write([]string{string(jsonData)})
	}
	
	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

// exportText exports the result as plain text
func (r *BaseResult) exportText() ([]byte, error) {
	var buf strings.Builder
	
	// Write metadata
	buf.WriteString("=== NetTraceX Result ===\n")
	buf.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format(time.RFC3339)))
	for key, value := range r.metadata {
		buf.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}
	buf.WriteString("\n=== Data ===\n")
	
	// Format data based on type
	switch data := r.data.(type) {
	case []PingResult:
		for _, result := range data {
			buf.WriteString(fmt.Sprintf("Ping %s: seq=%d time=%v ttl=%d\n",
				result.Host.Hostname, result.Sequence, result.RTT, result.TTL))
		}
	case []TraceHop:
		for _, hop := range data {
			buf.WriteString(fmt.Sprintf("Hop %d: %s (%s) %v\n",
				hop.Number, hop.Host.Hostname, hop.Host.IPAddress, hop.RTT))
		}
	case DNSResult:
		buf.WriteString(fmt.Sprintf("DNS Query: %s (Type: %d)\n", data.Query, data.RecordType))
		for _, record := range data.Records {
			buf.WriteString(fmt.Sprintf("  %s %d %s\n", record.Name, record.TTL, record.Value))
		}
	case WHOISResult:
		buf.WriteString(fmt.Sprintf("Domain: %s\n", data.Domain))
		buf.WriteString(fmt.Sprintf("Registrar: %s\n", data.Registrar))
		buf.WriteString(fmt.Sprintf("Created: %s\n", data.Created.Format(time.RFC3339)))
		buf.WriteString(fmt.Sprintf("Expires: %s\n", data.Expires.Format(time.RFC3339)))
	case SSLResult:
		buf.WriteString(fmt.Sprintf("SSL Certificate for %s:%d\n", data.Host, data.Port))
		buf.WriteString(fmt.Sprintf("Subject: %s\n", data.Subject))
		buf.WriteString(fmt.Sprintf("Issuer: %s\n", data.Issuer))
		buf.WriteString(fmt.Sprintf("Valid: %t\n", data.Valid))
		buf.WriteString(fmt.Sprintf("Expires: %s\n", data.Expiry.Format(time.RFC3339)))
	default:
		buf.WriteString(fmt.Sprintf("%+v\n", data))
	}
	
	return []byte(buf.String()), nil
}

// Helper methods for specific CSV exports
func (r *BaseResult) exportPingResultsCSV(results []PingResult) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	writer.Write([]string{"timestamp", "host", "sequence", "rtt_ms", "ttl", "packet_size"})
	
	// Write data
	for _, result := range results {
		rttMs := float64(result.RTT.Nanoseconds()) / 1000000.0
		writer.Write([]string{
			result.Timestamp.Format(time.RFC3339),
			result.Host.Hostname,
			fmt.Sprintf("%d", result.Sequence),
			fmt.Sprintf("%.3f", rttMs),
			fmt.Sprintf("%d", result.TTL),
			fmt.Sprintf("%d", result.PacketSize),
		})
	}
	
	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

func (r *BaseResult) exportTraceHopsCSV(hops []TraceHop) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	writer.Write([]string{"hop", "hostname", "ip_address", "rtt1_ms", "rtt2_ms", "rtt3_ms", "timeout"})
	
	// Write data
	for _, hop := range hops {
		rttStrs := make([]string, 3)
		for i := 0; i < 3; i++ {
			if i < len(hop.RTT) {
				rttMs := float64(hop.RTT[i].Nanoseconds()) / 1000000.0
				rttStrs[i] = fmt.Sprintf("%.3f", rttMs)
			} else {
				rttStrs[i] = ""
			}
		}
		
		writer.Write([]string{
			fmt.Sprintf("%d", hop.Number),
			hop.Host.Hostname,
			hop.Host.IPAddress.String(),
			rttStrs[0],
			rttStrs[1],
			rttStrs[2],
			fmt.Sprintf("%t", hop.Timeout),
		})
	}
	
	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

func (r *BaseResult) exportDNSResultCSV(result DNSResult) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	writer.Write([]string{"name", "type", "value", "ttl", "priority"})
	
	// Write records
	for _, record := range result.Records {
		writer.Write([]string{
			record.Name,
			fmt.Sprintf("%d", record.Type),
			record.Value,
			fmt.Sprintf("%d", record.TTL),
			fmt.Sprintf("%d", record.Priority),
		})
	}
	
	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

func (r *BaseResult) exportWHOISResultCSV(result WHOISResult) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	writer.Write([]string{"field", "value"})
	
	// Write data
	writer.Write([]string{"domain", result.Domain})
	writer.Write([]string{"registrar", result.Registrar})
	writer.Write([]string{"created", result.Created.Format(time.RFC3339)})
	writer.Write([]string{"updated", result.Updated.Format(time.RFC3339)})
	writer.Write([]string{"expires", result.Expires.Format(time.RFC3339)})
	
	for _, ns := range result.NameServers {
		writer.Write([]string{"nameserver", ns})
	}
	
	writer.Flush()
	return []byte(buf.String()), writer.Error()
}

func (r *BaseResult) exportSSLResultCSV(result SSLResult) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	writer.Write([]string{"field", "value"})
	
	// Write data
	writer.Write([]string{"host", result.Host})
	writer.Write([]string{"port", fmt.Sprintf("%d", result.Port)})
	writer.Write([]string{"subject", result.Subject})
	writer.Write([]string{"issuer", result.Issuer})
	writer.Write([]string{"valid", fmt.Sprintf("%t", result.Valid)})
	writer.Write([]string{"expires", result.Expiry.Format(time.RFC3339)})
	
	for _, san := range result.SANs {
		writer.Write([]string{"san", san})
	}
	
	writer.Flush()
	return []byte(buf.String()), writer.Error()
}