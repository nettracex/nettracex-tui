// Package dns provides DNS diagnostic functionality
package dns

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Tool implements the DiagnosticTool interface for DNS operations
type Tool struct {
	client domain.NetworkClient
	logger domain.Logger
}

// NewTool creates a new DNS diagnostic tool
func NewTool(client domain.NetworkClient, logger domain.Logger) *Tool {
	return &Tool{
		client: client,
		logger: logger,
	}
}

// Name returns the tool name
func (t *Tool) Name() string {
	return "dns"
}

// Description returns the tool description
func (t *Tool) Description() string {
	return "Performs DNS lookups for multiple record types (A, AAAA, MX, TXT, CNAME, NS) with concurrent queries"
}

// Execute performs the DNS lookup operation
func (t *Tool) Execute(ctx context.Context, params domain.Parameters) (domain.Result, error) {
	t.logger.Info("Executing DNS lookup", "tool", t.Name())

	// Validate parameters
	if err := t.Validate(params); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "DNS parameter validation failed",
			Cause:     err,
			Context:   map[string]interface{}{"params": params.ToMap()},
			Timestamp: time.Now(),
			Code:      "DNS_VALIDATION_FAILED",
		}
	}

	domainName := params.Get("domain").(string)
	recordTypes := t.getRecordTypes(params)

	// Perform concurrent DNS lookups for multiple record types
	results, err := t.performConcurrentLookups(ctx, domainName, recordTypes)
	if err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "DNS lookup operation failed",
			Cause:     err,
			Context:   map[string]interface{}{"domain": domainName, "record_types": recordTypes},
			Timestamp: time.Now(),
			Code:      "DNS_LOOKUP_FAILED",
		}
	}

	// Create consolidated result
	consolidatedResult := t.consolidateResults(domainName, results)

	// Create result with metadata
	result := domain.NewResult(consolidatedResult)
	result.SetMetadata("tool", t.Name())
	result.SetMetadata("domain", domainName)
	result.SetMetadata("timestamp", time.Now())
	result.SetMetadata("record_types", recordTypes)
	result.SetMetadata("total_records", len(consolidatedResult.Records))

	t.logger.Info("DNS lookup completed successfully", "domain", domainName, "record_types", len(recordTypes), "total_records", len(consolidatedResult.Records))
	return result, nil
}

// Validate validates the parameters for DNS operations
func (t *Tool) Validate(params domain.Parameters) error {
	domainParam := params.Get("domain")
	if domainParam == nil {
		return fmt.Errorf("domain parameter is required")
	}

	domainName, ok := domainParam.(string)
	if !ok {
		return fmt.Errorf("domain parameter must be a string")
	}

	if strings.TrimSpace(domainName) == "" {
		return fmt.Errorf("domain parameter cannot be empty")
	}

	// Validate domain format
	if !t.isValidDomain(domainName) {
		return fmt.Errorf("domain must be a valid domain name")
	}

	// Validate record types if specified
	if recordTypesParam := params.Get("record_types"); recordTypesParam != nil {
		recordTypes, ok := recordTypesParam.([]domain.DNSRecordType)
		if !ok {
			return fmt.Errorf("record_types parameter must be a slice of DNSRecordType")
		}

		for _, recordType := range recordTypes {
			if !t.isValidRecordType(recordType) {
				return fmt.Errorf("invalid DNS record type: %v", recordType)
			}
		}
	}

	return nil
}

// GetModel returns the Bubble Tea model for the DNS tool
func (t *Tool) GetModel() tea.Model {
	return NewModel(t)
}

// isValidDomain validates if the string is a valid domain name
func (t *Tool) isValidDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	
	// Basic length validation
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	// Must contain at least one dot for TLD (except for special cases like localhost)
	if !strings.Contains(domain, ".") && domain != "localhost" {
		return false
	}
	
	// Domain regex pattern - allows for subdomains and international domains
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return false
	}
	
	// Check that each label is valid
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		// Labels cannot start or end with hyphens
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	
	return true
}

// isValidRecordType validates if the record type is supported
func (t *Tool) isValidRecordType(recordType domain.DNSRecordType) bool {
	switch recordType {
	case domain.DNSRecordTypeA, domain.DNSRecordTypeAAAA, domain.DNSRecordTypeMX,
		 domain.DNSRecordTypeTXT, domain.DNSRecordTypeCNAME, domain.DNSRecordTypeNS:
		return true
	default:
		return false
	}
}

// getRecordTypes extracts record types from parameters or returns default set
func (t *Tool) getRecordTypes(params domain.Parameters) []domain.DNSRecordType {
	if recordTypesParam := params.Get("record_types"); recordTypesParam != nil {
		if recordTypes, ok := recordTypesParam.([]domain.DNSRecordType); ok {
			return recordTypes
		}
	}

	// Default to all supported record types
	return []domain.DNSRecordType{
		domain.DNSRecordTypeA,
		domain.DNSRecordTypeAAAA,
		domain.DNSRecordTypeMX,
		domain.DNSRecordTypeTXT,
		domain.DNSRecordTypeCNAME,
		domain.DNSRecordTypeNS,
	}
}

// performConcurrentLookups performs DNS lookups for multiple record types concurrently
func (t *Tool) performConcurrentLookups(ctx context.Context, domainName string, recordTypes []domain.DNSRecordType) (map[domain.DNSRecordType]domain.DNSResult, error) {
	results := make(map[domain.DNSRecordType]domain.DNSResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstError error
	var errorOnce sync.Once

	// Create a channel to limit concurrent operations
	semaphore := make(chan struct{}, 3) // Limit to 3 concurrent DNS queries

	for _, recordType := range recordTypes {
		wg.Add(1)
		go func(rt domain.DNSRecordType) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Perform DNS lookup
			result, err := t.client.DNSLookup(ctx, domainName, rt)
			if err != nil {
				t.logger.Warn("DNS lookup failed for record type", "domain", domainName, "record_type", rt, "error", err)
				// Store first error but continue with other lookups
				errorOnce.Do(func() {
					firstError = err
				})
				return
			}

			// Store result
			mu.Lock()
			results[rt] = result
			mu.Unlock()

			t.logger.Debug("DNS lookup completed for record type", "domain", domainName, "record_type", rt, "record_count", len(result.Records))
		}(recordType)
	}

	wg.Wait()

	// If no results were obtained and we have an error, return the error
	if len(results) == 0 && firstError != nil {
		return nil, firstError
	}

	return results, nil
}

// consolidateResults consolidates multiple DNS results into a single result
func (t *Tool) consolidateResults(domainName string, results map[domain.DNSRecordType]domain.DNSResult) domain.DNSResult {
	consolidated := domain.DNSResult{
		Query:        domainName,
		RecordType:   domain.DNSRecordTypeA, // Default, will be overridden in multi-type queries
		Records:      []domain.DNSRecord{},
		Authority:    []domain.DNSRecord{},
		Additional:   []domain.DNSRecord{},
		ResponseTime: 0,
		Server:       "system",
	}

	var totalResponseTime time.Duration
	recordCount := 0

	// Consolidate all records from different types
	for _, result := range results {
		consolidated.Records = append(consolidated.Records, result.Records...)
		consolidated.Authority = append(consolidated.Authority, result.Authority...)
		consolidated.Additional = append(consolidated.Additional, result.Additional...)
		
		totalResponseTime += result.ResponseTime
		recordCount++
	}

	// Calculate average response time
	if recordCount > 0 {
		consolidated.ResponseTime = totalResponseTime / time.Duration(recordCount)
	}

	return consolidated
}

// GetRecordTypeString returns a human-readable string for a DNS record type
func GetRecordTypeString(recordType domain.DNSRecordType) string {
	switch recordType {
	case domain.DNSRecordTypeA:
		return "A"
	case domain.DNSRecordTypeAAAA:
		return "AAAA"
	case domain.DNSRecordTypeMX:
		return "MX"
	case domain.DNSRecordTypeTXT:
		return "TXT"
	case domain.DNSRecordTypeCNAME:
		return "CNAME"
	case domain.DNSRecordTypeNS:
		return "NS"
	case domain.DNSRecordTypeSOA:
		return "SOA"
	case domain.DNSRecordTypePTR:
		return "PTR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", recordType)
	}
}

// ParseRecordTypeString parses a string into a DNS record type
func ParseRecordTypeString(recordTypeStr string) (domain.DNSRecordType, error) {
	switch strings.ToUpper(strings.TrimSpace(recordTypeStr)) {
	case "A":
		return domain.DNSRecordTypeA, nil
	case "AAAA":
		return domain.DNSRecordTypeAAAA, nil
	case "MX":
		return domain.DNSRecordTypeMX, nil
	case "TXT":
		return domain.DNSRecordTypeTXT, nil
	case "CNAME":
		return domain.DNSRecordTypeCNAME, nil
	case "NS":
		return domain.DNSRecordTypeNS, nil
	case "SOA":
		return domain.DNSRecordTypeSOA, nil
	case "PTR":
		return domain.DNSRecordTypePTR, nil
	default:
		return domain.DNSRecordTypeA, fmt.Errorf("unknown DNS record type: %s", recordTypeStr)
	}
}

// FormatDNSResult formats DNS result for display
func FormatDNSResult(result domain.DNSResult) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("DNS Query: %s\n", result.Query))
	builder.WriteString(fmt.Sprintf("Server: %s\n", result.Server))
	builder.WriteString(fmt.Sprintf("Response Time: %v\n", result.ResponseTime))
	builder.WriteString(fmt.Sprintf("Total Records: %d\n", len(result.Records)))
	
	if len(result.Records) > 0 {
		builder.WriteString("\nRecords:\n")
		
		// Group records by type for better display
		recordsByType := make(map[domain.DNSRecordType][]domain.DNSRecord)
		for _, record := range result.Records {
			recordsByType[record.Type] = append(recordsByType[record.Type], record)
		}
		
		// Display records grouped by type
		for recordType, records := range recordsByType {
			builder.WriteString(fmt.Sprintf("\n%s Records:\n", GetRecordTypeString(recordType)))
			for _, record := range records {
				if record.Priority > 0 {
					builder.WriteString(fmt.Sprintf("  %s %d %s (Priority: %d)\n", 
						record.Name, record.TTL, record.Value, record.Priority))
				} else {
					builder.WriteString(fmt.Sprintf("  %s %d %s\n", 
						record.Name, record.TTL, record.Value))
				}
			}
		}
	}
	
	if len(result.Authority) > 0 {
		builder.WriteString("\nAuthority Records:\n")
		for _, record := range result.Authority {
			builder.WriteString(fmt.Sprintf("  %s %d %s\n", 
				record.Name, record.TTL, record.Value))
		}
	}
	
	if len(result.Additional) > 0 {
		builder.WriteString("\nAdditional Records:\n")
		for _, record := range result.Additional {
			builder.WriteString(fmt.Sprintf("  %s %d %s\n", 
				record.Name, record.TTL, record.Value))
		}
	}
	
	return builder.String()
}

// ValidateDNSResult validates that a DNS result contains expected data
func ValidateDNSResult(result domain.DNSResult) error {
	if result.Query == "" {
		return fmt.Errorf("DNS result missing query")
	}
	
	if result.ResponseTime <= 0 {
		return fmt.Errorf("DNS result has invalid response time")
	}
	
	// At least one of records, authority, or additional should have data
	if len(result.Records) == 0 && len(result.Authority) == 0 && len(result.Additional) == 0 {
		return fmt.Errorf("DNS result contains no records")
	}
	
	// Validate individual records
	for _, record := range result.Records {
		if err := ValidateDNSRecord(record); err != nil {
			return fmt.Errorf("invalid DNS record: %w", err)
		}
	}
	
	return nil
}

// ValidateDNSRecord validates a single DNS record
func ValidateDNSRecord(record domain.DNSRecord) error {
	if record.Name == "" {
		return fmt.Errorf("DNS record missing name")
	}
	
	if record.Value == "" {
		return fmt.Errorf("DNS record missing value")
	}
	
	if record.TTL == 0 {
		return fmt.Errorf("DNS record has zero TTL")
	}
	
	// Validate record type
	if !isValidRecordTypeForValidation(record.Type) {
		return fmt.Errorf("DNS record has invalid type: %v", record.Type)
	}
	
	return nil
}

// isValidRecordTypeForValidation checks if a record type is valid for validation
func isValidRecordTypeForValidation(recordType domain.DNSRecordType) bool {
	switch recordType {
	case domain.DNSRecordTypeA, domain.DNSRecordTypeAAAA, domain.DNSRecordTypeMX,
		 domain.DNSRecordTypeTXT, domain.DNSRecordTypeCNAME, domain.DNSRecordTypeNS,
		 domain.DNSRecordTypeSOA, domain.DNSRecordTypePTR:
		return true
	default:
		return false
	}
}