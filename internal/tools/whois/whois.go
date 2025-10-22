// Package whois provides WHOIS diagnostic functionality
package whois

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Tool implements the DiagnosticTool interface for WHOIS operations
type Tool struct {
	client domain.NetworkClient
	logger domain.Logger
}

// NewTool creates a new WHOIS diagnostic tool
func NewTool(client domain.NetworkClient, logger domain.Logger) *Tool {
	return &Tool{
		client: client,
		logger: logger,
	}
}

// Name returns the tool name
func (t *Tool) Name() string {
	return "whois"
}

// Description returns the tool description
func (t *Tool) Description() string {
	return "Performs WHOIS lookups for domains and IP addresses to retrieve registration information"
}

// Execute performs the WHOIS lookup operation
func (t *Tool) Execute(ctx context.Context, params domain.Parameters) (domain.Result, error) {
	t.logger.Info("Executing WHOIS lookup", "tool", t.Name())

	// Validate parameters
	if err := t.Validate(params); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "WHOIS parameter validation failed",
			Cause:     err,
			Context:   map[string]interface{}{"params": params.ToMap()},
			Timestamp: time.Now(),
			Code:      "WHOIS_VALIDATION_FAILED",
		}
	}

	query := params.Get("query").(string)

	// Perform WHOIS lookup
	whoisResult, err := t.client.WHOISLookup(ctx, query)
	if err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "WHOIS lookup operation failed",
			Cause:     err,
			Context:   map[string]interface{}{"query": query},
			Timestamp: time.Now(),
			Code:      "WHOIS_LOOKUP_FAILED",
		}
	}

	// Create result with metadata
	result := domain.NewResult(whoisResult)
	result.SetMetadata("tool", t.Name())
	result.SetMetadata("query", query)
	result.SetMetadata("timestamp", time.Now())
	result.SetMetadata("query_type", t.determineQueryType(query))

	t.logger.Info("WHOIS lookup completed successfully", "query", query, "domain", whoisResult.Domain)
	return result, nil
}

// Validate validates the parameters for WHOIS operations
func (t *Tool) Validate(params domain.Parameters) error {
	query := params.Get("query")
	if query == nil {
		return fmt.Errorf("query parameter is required")
	}

	queryStr, ok := query.(string)
	if !ok {
		return fmt.Errorf("query parameter must be a string")
	}

	if strings.TrimSpace(queryStr) == "" {
		return fmt.Errorf("query parameter cannot be empty")
	}

	// Validate query format (domain or IP)
	if !t.isValidQuery(queryStr) {
		return fmt.Errorf("query must be a valid domain name or IP address")
	}

	return nil
}

// GetModel returns the Bubble Tea model for the WHOIS tool
func (t *Tool) GetModel() tea.Model {
	return NewModel(t)
}

// isValidQuery validates if the query is a valid domain or IP address
func (t *Tool) isValidQuery(query string) bool {
	query = strings.TrimSpace(query)
	
	// Check if it's a valid IP address
	if net.ParseIP(query) != nil {
		return true
	}
	
	// Check if it's a valid domain name
	return t.isValidDomain(query)
}

// isValidDomain validates if the string is a valid domain name
func (t *Tool) isValidDomain(domain string) bool {
	// Basic domain validation
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	
	// Must contain at least one dot for TLD
	if !strings.Contains(domain, ".") {
		return false
	}
	
	// Domain regex pattern - must have at least 2 parts (domain.tld)
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)+$`)
	if !domainRegex.MatchString(domain) {
		return false
	}
	
	// Check that it has at least 2 parts after splitting by dot
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}
	
	// Each part should be at least 1 character
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}
	
	return true
}

// determineQueryType determines if the query is a domain or IP address
func (t *Tool) determineQueryType(query string) string {
	if net.ParseIP(query) != nil {
		return "ip"
	}
	return "domain"
}

// ParseWHOISData parses raw WHOIS data into structured format
func ParseWHOISData(rawData string, query string) domain.WHOISResult {
	result := domain.WHOISResult{
		Domain:      query,
		RawData:     rawData,
		Contacts:    make(map[string]domain.Contact),
		NameServers: []string{},
		Status:      []string{},
	}

	lines := strings.Split(rawData, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key-value pairs
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "domain name", "domain":
			result.Domain = value
		case "registrar":
			result.Registrar = value
		case "creation date", "created", "registered":
			if date, err := parseDate(value); err == nil {
				result.Created = date
			}
		case "updated date", "last updated", "modified":
			if date, err := parseDate(value); err == nil {
				result.Updated = date
			}
		case "expiry date", "expires", "expiration date":
			if date, err := parseDate(value); err == nil {
				result.Expires = date
			}
		case "name server", "nameserver", "nserver":
			if value != "" {
				result.NameServers = append(result.NameServers, value)
			}
		case "status", "domain status":
			if value != "" {
				result.Status = append(result.Status, value)
			}
		case "registrant name":
			contact := result.Contacts["registrant"]
			contact.Name = value
			result.Contacts["registrant"] = contact
		case "registrant organization":
			contact := result.Contacts["registrant"]
			contact.Organization = value
			result.Contacts["registrant"] = contact
		case "registrant email":
			contact := result.Contacts["registrant"]
			contact.Email = value
			result.Contacts["registrant"] = contact
		case "admin name":
			contact := result.Contacts["admin"]
			contact.Name = value
			result.Contacts["admin"] = contact
		case "admin email":
			contact := result.Contacts["admin"]
			contact.Email = value
			result.Contacts["admin"] = contact
		case "tech name":
			contact := result.Contacts["tech"]
			contact.Name = value
			result.Contacts["tech"] = contact
		case "tech email":
			contact := result.Contacts["tech"]
			contact.Email = value
			result.Contacts["tech"] = contact
		}
	}

	return result
}

// parseDate attempts to parse various date formats commonly found in WHOIS data
func parseDate(dateStr string) (time.Time, error) {
	// Common WHOIS date formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02-Jan-2006",
		"2006.01.02",
		"01/02/2006",
		"2006/01/02",
	}

	// Clean the date string
	dateStr = strings.TrimSpace(dateStr)
	
	// Remove common suffixes
	dateStr = strings.Replace(dateStr, " UTC", "", -1)
	dateStr = strings.Replace(dateStr, " GMT", "", -1)
	
	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// FormatWHOISResult formats WHOIS result for display
func FormatWHOISResult(result domain.WHOISResult) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("Domain: %s\n", result.Domain))
	
	if result.Registrar != "" {
		builder.WriteString(fmt.Sprintf("Registrar: %s\n", result.Registrar))
	}
	
	if !result.Created.IsZero() {
		builder.WriteString(fmt.Sprintf("Created: %s\n", result.Created.Format("2006-01-02 15:04:05")))
	}
	
	if !result.Updated.IsZero() {
		builder.WriteString(fmt.Sprintf("Updated: %s\n", result.Updated.Format("2006-01-02 15:04:05")))
	}
	
	if !result.Expires.IsZero() {
		builder.WriteString(fmt.Sprintf("Expires: %s\n", result.Expires.Format("2006-01-02 15:04:05")))
		
		// Add expiration warning if within 30 days
		daysUntilExpiry := time.Until(result.Expires).Hours() / 24
		if daysUntilExpiry <= 30 && daysUntilExpiry > 0 {
			builder.WriteString(fmt.Sprintf("⚠️  WARNING: Domain expires in %.0f days!\n", daysUntilExpiry))
		} else if daysUntilExpiry <= 0 {
			builder.WriteString("🚨 WARNING: Domain has expired!\n")
		}
	}
	
	if len(result.NameServers) > 0 {
		builder.WriteString("\nName Servers:\n")
		for _, ns := range result.NameServers {
			builder.WriteString(fmt.Sprintf("  %s\n", ns))
		}
	}
	
	if len(result.Status) > 0 {
		builder.WriteString("\nStatus:\n")
		for _, status := range result.Status {
			builder.WriteString(fmt.Sprintf("  %s\n", status))
		}
	}
	
	if len(result.Contacts) > 0 {
		builder.WriteString("\nContacts:\n")
		for contactType, contact := range result.Contacts {
			if contact.Name != "" || contact.Email != "" || contact.Organization != "" {
				builder.WriteString(fmt.Sprintf("  %s:\n", strings.Title(contactType)))
				if contact.Name != "" {
					builder.WriteString(fmt.Sprintf("    Name: %s\n", contact.Name))
				}
				if contact.Organization != "" {
					builder.WriteString(fmt.Sprintf("    Organization: %s\n", contact.Organization))
				}
				if contact.Email != "" {
					builder.WriteString(fmt.Sprintf("    Email: %s\n", contact.Email))
				}
				if contact.Phone != "" {
					builder.WriteString(fmt.Sprintf("    Phone: %s\n", contact.Phone))
				}
			}
		}
	}
	
	return builder.String()
}

// ValidateWHOISResult validates that a WHOIS result contains expected data
func ValidateWHOISResult(result domain.WHOISResult) error {
	if result.Domain == "" {
		return fmt.Errorf("WHOIS result missing domain name")
	}
	
	if result.RawData == "" {
		return fmt.Errorf("WHOIS result missing raw data")
	}
	
	// Check if we have at least some meaningful data
	hasData := result.Registrar != "" || 
		!result.Created.IsZero() || 
		!result.Expires.IsZero() || 
		len(result.NameServers) > 0 || 
		len(result.Contacts) > 0
	
	if !hasData {
		return fmt.Errorf("WHOIS result contains no meaningful data")
	}
	
	return nil
}