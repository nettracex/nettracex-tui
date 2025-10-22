// Package ssl provides basic SSL certificate diagnostic functionality tests
package ssl

import (
	"testing"
	"time"

	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSSLTool_Basic(t *testing.T) {
	// Test tool creation without network client to avoid mock issues
	tool := &Tool{}
	
	assert.Equal(t, "ssl", tool.Name())
	assert.Contains(t, tool.Description(), "SSL certificate checks")
}

func TestSSLTool_Validation(t *testing.T) {
	tool := &Tool{}
	
	// Test host validation
	assert.True(t, tool.isValidHost("example.com"))
	assert.True(t, tool.isValidHost("api.example.com"))
	assert.True(t, tool.isValidHost("192.168.1.1"))
	assert.False(t, tool.isValidHost(""))
	assert.False(t, tool.isValidHost("invalid@host"))
}

func TestSSLTool_SecurityAnalysis(t *testing.T) {
	tool := &Tool{}
	
	// Test days until expiry calculation
	// This is a simple test that doesn't require certificates
	days := tool.calculateDaysUntilExpiry(time.Now().Add(30 * 24 * time.Hour))
	assert.Equal(t, 30, days)
	
	days = tool.calculateDaysUntilExpiry(time.Now().Add(-10 * 24 * time.Hour))
	assert.Equal(t, -10, days)
}

func TestSSLFormatting(t *testing.T) {
	// Test security level function
	validResult := domain.SSLResult{Valid: true, Errors: []string{}}
	level := GetSecurityLevel(validResult)
	assert.Equal(t, "SECURE", level)
	
	invalidResult := domain.SSLResult{Valid: false, Errors: []string{"expired"}}
	level = GetSecurityLevel(invalidResult)
	assert.Equal(t, "INSECURE", level)
}

func TestSSLRecommendations(t *testing.T) {
	// Test with no certificate (should return default recommendation)
	result := domain.SSLResult{}
	recommendations := GetSecurityRecommendations(result)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations, "Certificate configuration appears secure")
}