// Package ssl provides SSL certificate diagnostic functionality
package ssl

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Tool implements the DiagnosticTool interface for SSL certificate operations
type Tool struct {
	client domain.NetworkClient
	logger domain.Logger
}

// NewTool creates a new SSL diagnostic tool
func NewTool(client domain.NetworkClient, logger domain.Logger) *Tool {
	return &Tool{
		client: client,
		logger: logger,
	}
}

// Name returns the tool name
func (t *Tool) Name() string {
	return "ssl"
}

// Description returns the tool description
func (t *Tool) Description() string {
	return "Performs SSL certificate checks with validation, chain analysis, and security warnings"
}

// Execute performs the SSL certificate check operation
func (t *Tool) Execute(ctx context.Context, params domain.Parameters) (domain.Result, error) {
	t.logger.Info("Executing SSL certificate check", "tool", t.Name())

	// Validate parameters
	if err := t.Validate(params); err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeValidation,
			Message:   "SSL parameter validation failed",
			Cause:     err,
			Context:   map[string]interface{}{"params": params.ToMap()},
			Timestamp: time.Now(),
			Code:      "SSL_VALIDATION_FAILED",
		}
	}

	host := params.Get("host").(string)
	port := params.Get("port").(int)

	// Perform SSL certificate check
	sslResult, err := t.client.SSLCheck(ctx, host, port)
	if err != nil {
		return nil, &domain.NetTraceError{
			Type:      domain.ErrorTypeNetwork,
			Message:   "SSL certificate check operation failed",
			Cause:     err,
			Context:   map[string]interface{}{"host": host, "port": port},
			Timestamp: time.Now(),
			Code:      "SSL_CHECK_FAILED",
		}
	}

	// Perform additional security analysis
	enhancedResult := t.performSecurityAnalysis(sslResult)

	// Create result with metadata
	result := domain.NewResult(enhancedResult)
	result.SetMetadata("tool", t.Name())
	result.SetMetadata("host", host)
	result.SetMetadata("port", port)
	result.SetMetadata("timestamp", time.Now())
	result.SetMetadata("certificate_valid", enhancedResult.Valid)
	result.SetMetadata("days_until_expiry", t.calculateDaysUntilExpiry(enhancedResult.Expiry))

	t.logger.Info("SSL certificate check completed successfully", "host", host, "port", port, "valid", enhancedResult.Valid)
	return result, nil
}

// Validate validates the parameters for SSL operations
func (t *Tool) Validate(params domain.Parameters) error {
	host := params.Get("host")
	if host == nil {
		return fmt.Errorf("host parameter is required")
	}

	hostStr, ok := host.(string)
	if !ok {
		return fmt.Errorf("host parameter must be a string")
	}

	if strings.TrimSpace(hostStr) == "" {
		return fmt.Errorf("host parameter cannot be empty")
	}

	// Validate host format
	if !t.isValidHost(hostStr) {
		return fmt.Errorf("host must be a valid hostname or IP address")
	}

	port := params.Get("port")
	if port == nil {
		return fmt.Errorf("port parameter is required")
	}

	var portInt int
	switch v := port.(type) {
	case int:
		portInt = v
	case string:
		var err error
		portInt, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("port parameter must be a valid integer")
		}
	default:
		return fmt.Errorf("port parameter must be an integer or string")
	}

	if portInt <= 0 || portInt > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	// Update the port parameter to ensure it's an integer
	params.Set("port", portInt)

	return nil
}

// GetModel returns the Bubble Tea model for the SSL tool
func (t *Tool) GetModel() tea.Model {
	return NewModel(t)
}

// isValidHost validates if the host is a valid hostname or IP address
func (t *Tool) isValidHost(host string) bool {
	host = strings.TrimSpace(host)
	
	if len(host) == 0 || len(host) > 253 {
		return false
	}

	// Basic hostname validation - allow letters, numbers, dots, and hyphens
	for _, char := range host {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '.' || char == '-') {
			return false
		}
	}

	return true
}

// performSecurityAnalysis performs additional security analysis on the SSL result
func (t *Tool) performSecurityAnalysis(result domain.SSLResult) domain.SSLResult {
	// Create a copy to avoid modifying the original
	enhanced := result
	
	// Additional security checks
	if result.Certificate != nil {
		cert := result.Certificate
		
		// Check for weak signature algorithms
		if strings.Contains(strings.ToLower(cert.SignatureAlgorithm.String()), "sha1") {
			enhanced.Errors = append(enhanced.Errors, "certificate uses weak SHA-1 signature algorithm")
			enhanced.Valid = false
		}
		
		// Check for weak key sizes
		if cert.PublicKeyAlgorithm.String() == "RSA" {
			if rsaKey, ok := cert.PublicKey.(interface{ Size() int }); ok {
				keySize := rsaKey.Size() * 8 // Convert bytes to bits
				if keySize < 2048 {
					enhanced.Errors = append(enhanced.Errors, fmt.Sprintf("certificate uses weak RSA key size: %d bits", keySize))
					enhanced.Valid = false
				}
			}
		}
		
		// Check certificate expiry warnings
		daysUntilExpiry := t.calculateDaysUntilExpiry(cert.NotAfter)
		if daysUntilExpiry <= 30 && daysUntilExpiry > 0 {
			enhanced.Errors = append(enhanced.Errors, fmt.Sprintf("certificate expires in %d days", daysUntilExpiry))
		} else if daysUntilExpiry <= 0 {
			enhanced.Errors = append(enhanced.Errors, "certificate has expired")
			enhanced.Valid = false
		}
		
		// Check for self-signed certificates
		if cert.Issuer.String() == cert.Subject.String() {
			enhanced.Errors = append(enhanced.Errors, "certificate is self-signed")
		}
		
		// Check certificate chain length
		if len(result.Chain) == 1 {
			enhanced.Errors = append(enhanced.Errors, "certificate chain contains only one certificate")
		}
	}
	
	return enhanced
}

// calculateDaysUntilExpiry calculates the number of days until certificate expiry
func (t *Tool) calculateDaysUntilExpiry(expiry time.Time) int {
	duration := time.Until(expiry)
	return int(duration.Hours() / 24)
}

// FormatSSLResult formats SSL result for display
func FormatSSLResult(result domain.SSLResult) string {
	var builder strings.Builder
	
	builder.WriteString(fmt.Sprintf("SSL Certificate Check: %s:%d\n", result.Host, result.Port))
	builder.WriteString(fmt.Sprintf("Valid: %t\n", result.Valid))
	
	if result.Certificate != nil {
		cert := result.Certificate
		
		builder.WriteString(fmt.Sprintf("Subject: %s\n", result.Subject))
		builder.WriteString(fmt.Sprintf("Issuer: %s\n", result.Issuer))
		builder.WriteString(fmt.Sprintf("Serial Number: %s\n", cert.SerialNumber.String()))
		builder.WriteString(fmt.Sprintf("Valid From: %s\n", cert.NotBefore.Format("2006-01-02 15:04:05 UTC")))
		builder.WriteString(fmt.Sprintf("Valid Until: %s\n", cert.NotAfter.Format("2006-01-02 15:04:05 UTC")))
		
		// Show expiry status
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
		if daysUntilExpiry > 0 {
			builder.WriteString(fmt.Sprintf("Days Until Expiry: %d\n", daysUntilExpiry))
		} else if daysUntilExpiry == 0 {
			builder.WriteString("⚠️  Certificate expires today!\n")
		} else {
			builder.WriteString(fmt.Sprintf("🚨 Certificate expired %d days ago!\n", -daysUntilExpiry))
		}
		
		builder.WriteString(fmt.Sprintf("Signature Algorithm: %s\n", cert.SignatureAlgorithm.String()))
		builder.WriteString(fmt.Sprintf("Public Key Algorithm: %s\n", cert.PublicKeyAlgorithm.String()))
		
		// Show key size for RSA keys
		if cert.PublicKeyAlgorithm.String() == "RSA" {
			if rsaKey, ok := cert.PublicKey.(interface{ Size() int }); ok {
				keySize := rsaKey.Size() * 8
				builder.WriteString(fmt.Sprintf("Key Size: %d bits\n", keySize))
			}
		}
		
		// Show Subject Alternative Names
		if len(result.SANs) > 0 {
			builder.WriteString("\nSubject Alternative Names:\n")
			for _, san := range result.SANs {
				builder.WriteString(fmt.Sprintf("  %s\n", san))
			}
		}
		
		// Show certificate chain information
		if len(result.Chain) > 1 {
			builder.WriteString(fmt.Sprintf("\nCertificate Chain (%d certificates):\n", len(result.Chain)))
			for i, chainCert := range result.Chain {
				if i == 0 {
					builder.WriteString(fmt.Sprintf("  1. %s (End Entity)\n", chainCert.Subject.CommonName))
				} else {
					builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, chainCert.Subject.CommonName))
				}
			}
		}
	}
	
	// Show errors and warnings
	if len(result.Errors) > 0 {
		builder.WriteString("\nSecurity Issues:\n")
		for _, err := range result.Errors {
			builder.WriteString(fmt.Sprintf("  ⚠️  %s\n", err))
		}
	}
	
	if result.Valid && len(result.Errors) == 0 {
		builder.WriteString("\n✅ Certificate is valid and secure\n")
	}
	
	return builder.String()
}

// ValidateSSLResult validates that an SSL result contains expected data
func ValidateSSLResult(result domain.SSLResult) error {
	if result.Host == "" {
		return fmt.Errorf("SSL result missing host")
	}
	
	if result.Port <= 0 || result.Port > 65535 {
		return fmt.Errorf("SSL result has invalid port: %d", result.Port)
	}
	
	if result.Certificate == nil {
		return fmt.Errorf("SSL result missing certificate")
	}
	
	if result.Issuer == "" {
		return fmt.Errorf("SSL result missing issuer information")
	}
	
	if result.Subject == "" {
		return fmt.Errorf("SSL result missing subject information")
	}
	
	if result.Expiry.IsZero() {
		return fmt.Errorf("SSL result missing expiry date")
	}
	
	return nil
}

// GetSecurityLevel returns a security level assessment for the certificate
func GetSecurityLevel(result domain.SSLResult) string {
	if !result.Valid {
		return "INSECURE"
	}
	
	if len(result.Errors) > 0 {
		// Check for critical errors
		for _, err := range result.Errors {
			if strings.Contains(strings.ToLower(err), "expired") ||
			   strings.Contains(strings.ToLower(err), "weak") ||
			   strings.Contains(strings.ToLower(err), "sha-1") {
				return "WEAK"
			}
		}
		return "WARNING"
	}
	
	return "SECURE"
}

// GetSecurityRecommendations returns security recommendations based on the SSL result
func GetSecurityRecommendations(result domain.SSLResult) []string {
	var recommendations []string
	
	if result.Certificate != nil {
		cert := result.Certificate
		
		// Check expiry
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
		if daysUntilExpiry <= 30 && daysUntilExpiry > 0 {
			recommendations = append(recommendations, "Renew certificate before expiry")
		} else if daysUntilExpiry <= 0 {
			recommendations = append(recommendations, "Certificate has expired - renew immediately")
		}
		
		// Check signature algorithm
		if strings.Contains(strings.ToLower(cert.SignatureAlgorithm.String()), "sha1") {
			recommendations = append(recommendations, "Upgrade to SHA-256 or higher signature algorithm")
		}
		
		// Check key size
		if cert.PublicKeyAlgorithm.String() == "RSA" {
			if rsaKey, ok := cert.PublicKey.(interface{ Size() int }); ok {
				keySize := rsaKey.Size() * 8
				if keySize < 2048 {
					recommendations = append(recommendations, "Use RSA key size of 2048 bits or higher")
				}
			}
		}
		
		// Check for self-signed
		if cert.Issuer.String() == cert.Subject.String() {
			recommendations = append(recommendations, "Use a certificate from a trusted Certificate Authority")
		}
		
		// Check chain
		if len(result.Chain) == 1 {
			recommendations = append(recommendations, "Ensure complete certificate chain is configured")
		}
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Certificate configuration appears secure")
	}
	
	return recommendations
}