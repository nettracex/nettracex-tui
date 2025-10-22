// Package ssl provides focused SSL certificate diagnostic TUI tests
package ssl

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/nettracex/nettracex-tui/internal/tui"
	"github.com/stretchr/testify/assert"
)

// SimpleMockLogger implements a simple logger for testing
type SimpleMockLogger struct{}

func (l *SimpleMockLogger) Debug(msg string, fields ...interface{}) {}
func (l *SimpleMockLogger) Info(msg string, fields ...interface{})  {}
func (l *SimpleMockLogger) Warn(msg string, fields ...interface{})  {}
func (l *SimpleMockLogger) Error(msg string, fields ...interface{}) {}
func (l *SimpleMockLogger) Fatal(msg string, fields ...interface{}) {}

// Test helper functions

// createTestValidCertificate creates a valid test certificate
func createTestValidCertificate() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject: pkix.Name{
			CommonName:   "example.com",
			Organization: []string{"Example Corp"},
		},
		Issuer: pkix.Name{
			CommonName:   "Example CA",
			Organization: []string{"Example CA Corp"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(90 * 24 * time.Hour), // 90 days from now
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		PublicKeyAlgorithm:    x509.RSA,
	}
}

// createTestExpiredCertificate creates an expired test certificate
func createTestExpiredCertificate() *x509.Certificate {
	cert := createTestValidCertificate()
	cert.NotBefore = time.Now().Add(-365 * 24 * time.Hour) // 1 year ago
	cert.NotAfter = time.Now().Add(-30 * 24 * time.Hour)   // 30 days ago (expired)
	return cert
}

// createTestValidSSLResult creates a valid SSL result for testing
func createTestValidSSLResult() domain.SSLResult {
	cert := createTestValidCertificate()
	return domain.SSLResult{
		Host:        "example.com",
		Port:        443,
		Certificate: cert,
		Chain:       []*x509.Certificate{cert},
		Valid:       true,
		Errors:      []string{},
		Expiry:      cert.NotAfter,
		Issuer:      cert.Issuer.String(),
		Subject:     cert.Subject.String(),
		SANs:        []string{"example.com", "www.example.com"},
	}
}

// createTestExpiredSSLResult creates an expired SSL result for testing
func createTestExpiredSSLResult() domain.SSLResult {
	cert := createTestExpiredCertificate()
	return domain.SSLResult{
		Host:        "expired.example.com",
		Port:        443,
		Certificate: cert,
		Chain:       []*x509.Certificate{cert},
		Valid:       false,
		Errors:      []string{"certificate has expired"},
		Expiry:      cert.NotAfter,
		Issuer:      cert.Issuer.String(),
		Subject:     cert.Subject.String(),
		SANs:        []string{"expired.example.com"},
	}
}

// TUI Tests

func TestSSLTUIModel_Creation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	assert.NotNil(t, model)
	assert.Equal(t, tui.ViewStateInput, model.state)
	assert.Equal(t, 0, model.focusedInput)
	assert.NotNil(t, model.hostInput)
	assert.NotNil(t, model.portInput)
}

func TestSSLTUIModel_InitialView(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test initial view rendering
	view := model.View()
	assert.Contains(t, view, "SSL Certificate Check")
	assert.Contains(t, view, "Host:")
	assert.Contains(t, view, "Port:")
	assert.Contains(t, view, "Tab: Switch fields")
	assert.Contains(t, view, "Enter: Check certificate")
}

func TestSSLTUIModel_InputNavigation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test tab navigation
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	sslModel := updatedModel.(*Model)
	assert.Equal(t, 1, sslModel.focusedInput) // Should move to port input
	
	// Test tab navigation wrapping
	updatedModel, _ = sslModel.Update(tea.KeyMsg{Type: tea.KeyTab})
	sslModel = updatedModel.(*Model)
	assert.Equal(t, 0, sslModel.focusedInput) // Should wrap back to host input
}

func TestSSLTUIModel_SecurityIndicators_ValidCertificate(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Set result state with valid certificate
	validResult := createTestValidSSLResult()
	model.state = tui.ViewStateResult
	model.result = &validResult
	
	// Test result view rendering with security indicators
	view := model.View()
	
	// Check for positive security indicators
	assert.Contains(t, view, "✅ Certificate Valid")
	assert.Contains(t, view, "Subject:")
	assert.Contains(t, view, "Issuer:")
	assert.Contains(t, view, "Valid From:")
	assert.Contains(t, view, "Valid Until:")
	assert.Contains(t, view, "Days Until Expiry:")
	assert.Contains(t, view, "Signature Algorithm:")
	assert.Contains(t, view, "Subject Alternative Names:")
	// Certificate chain should be shown for single certificate too
	assert.Contains(t, view, "Recommendations:")
	
	// Should not contain error indicators for valid certificate
	assert.NotContains(t, view, "❌")
	assert.NotContains(t, view, "Security Issues:")
}

func TestSSLTUIModel_SecurityIndicators_ExpiredCertificate(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Set result state with expired certificate
	expiredResult := createTestExpiredSSLResult()
	model.state = tui.ViewStateResult
	model.result = &expiredResult
	
	// Test result view rendering with security warnings
	view := model.View()
	
	// Check for negative security indicators
	assert.Contains(t, view, "❌ Certificate Invalid")
	assert.Contains(t, view, "Security Issues:")
	assert.Contains(t, view, "certificate has expired")
	// Check that the certificate is marked as expired in recommendations
	assert.Contains(t, view, "Certificate has expired")
	
	// Should contain error styling
	assert.Contains(t, view, "⚠️")
}

func TestSSLTUIModel_CertificateChainVisualization(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Create certificate chain
	endEntity := createTestValidCertificate()
	endEntity.Subject.CommonName = "example.com"
	
	intermediate := createTestValidCertificate()
	intermediate.Subject.CommonName = "Intermediate CA"
	intermediate.Issuer.CommonName = "Root CA"
	
	root := createTestValidCertificate()
	root.Subject.CommonName = "Root CA"
	root.Issuer = root.Subject // Self-signed root
	
	result := createTestValidSSLResult()
	result.Chain = []*x509.Certificate{endEntity, intermediate, root}
	
	model.state = tui.ViewStateResult
	model.result = &result
	
	// Test certificate chain visualization
	view := model.View()
	
	assert.Contains(t, view, "Certificate Chain (3 certificates):")
	assert.Contains(t, view, "1. example.com (End Entity)")
	assert.Contains(t, view, "2. Intermediate CA")
	assert.Contains(t, view, "3. Root CA")
}

func TestSSLTUIModel_SecurityLevelDisplay(t *testing.T) {
	tests := []struct {
		name           string
		result         domain.SSLResult
		expectedLevel  string
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:          "Secure Certificate",
			result:        createTestValidSSLResult(),
			expectedLevel: "SECURE",
			shouldContain: []string{"✅", "Certificate Valid"},
			shouldNotContain: []string{"❌", "Security Issues"},
		},
		{
			name:          "Expired Certificate",
			result:        createTestExpiredSSLResult(),
			expectedLevel: "INSECURE",
			shouldContain: []string{"❌", "Certificate Invalid", "Security Issues"},
			shouldNotContain: []string{"✅"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test security level function
			level := GetSecurityLevel(tt.result)
			assert.Equal(t, tt.expectedLevel, level)
			
			// Test TUI display
			mockClient := network.NewMockClient()
			mockLogger := &SimpleMockLogger{}
			
			tool := NewTool(mockClient, mockLogger)
			model := NewModel(tool)
			
			model.state = tui.ViewStateResult
			model.result = &tt.result
			
			view := model.View()
			
			for _, expected := range tt.shouldContain {
				assert.Contains(t, view, expected, "View should contain: %s", expected)
			}
			
			for _, notExpected := range tt.shouldNotContain {
				assert.NotContains(t, view, notExpected, "View should not contain: %s", notExpected)
			}
		})
	}
}

func TestSSLTUIModel_ErrorHandling(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test error state display
	testError := fmt.Errorf("network timeout")
	model.state = tui.ViewStateError
	model.error = testError
	
	view := model.View()
	
	assert.Contains(t, view, "SSL Check Error")
	assert.Contains(t, view, "network timeout")
	assert.Contains(t, view, "Esc: Back")
}

func TestSSLTUIModel_LoadingState(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test loading state display
	model.state = tui.ViewStateLoading
	
	view := model.View()
	
	assert.Contains(t, view, "Checking SSL certificate...")
}

func TestSSLTUIModel_KeyboardNavigation(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test escape key from result state
	model.state = tui.ViewStateResult
	model.result = &domain.SSLResult{}
	
	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	sslModel := updatedModel.(*Model)
	
	assert.Equal(t, tui.ViewStateInput, sslModel.state)
	assert.Nil(t, sslModel.result)
	assert.Nil(t, sslModel.error)
	
	// Test quit key
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, cmd)
	// We can't directly compare tea.Cmd functions, so just check it's not nil
}

func TestSSLTUIModel_WindowResize(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test window resize
	updatedModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	sslModel := updatedModel.(*Model)
	
	assert.Equal(t, 100, sslModel.width)
	assert.Equal(t, 50, sslModel.height)
	
	// Host input width should be adjusted
	expectedWidth := 50 // min(50, 100-10)
	assert.Equal(t, expectedWidth, sslModel.hostInput.Width)
}

func TestSSLTUIModel_ComponentInterface(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Test TUIComponent interface methods
	model.SetSize(80, 40)
	assert.Equal(t, 80, model.width)
	assert.Equal(t, 40, model.height)
	
	// Test theme setting
	theme := tui.NewDefaultTheme()
	model.SetTheme(theme)
	assert.Equal(t, theme, model.theme)
	
	// Test focus/blur
	model.Focus()
	model.Blur()
	// These methods should not panic
}

// Integration Tests

func TestSSLTUIModel_FullWorkflow(t *testing.T) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	// Setup mock SSL result
	validResult := createTestValidSSLResult()
	mockClient.SetSSLResponse("example.com", 443, validResult)
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// 1. Initial state - input form
	view := model.View()
	assert.Contains(t, view, "SSL Certificate Check")
	assert.Contains(t, view, "Host:")
	
	// 2. Enter host and port
	model.hostInput.SetValue("example.com")
	model.portInput.SetValue("443")
	
	// 3. Submit form
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	sslModel := updatedModel.(*Model)
	
	// Should be in loading state
	assert.Equal(t, tui.ViewStateLoading, sslModel.state)
	loadingView := sslModel.View()
	assert.Contains(t, loadingView, "Checking SSL certificate...")
	
	// 4. Process result
	msg := cmd()
	resultMsg := msg.(tui.SSLCheckCompleteMsg)
	
	updatedModel, _ = sslModel.Update(resultMsg)
	sslModel = updatedModel.(*Model)
	
	// Should be in result state
	assert.Equal(t, tui.ViewStateResult, sslModel.state)
	assert.NotNil(t, sslModel.result)
	
	// 5. Check result display
	resultView := sslModel.View()
	assert.Contains(t, resultView, "✅ Certificate Valid")
	assert.Contains(t, resultView, "example.com:443")
	
	// 6. Navigate back
	updatedModel, _ = sslModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	sslModel = updatedModel.(*Model)
	
	// Should be back to input state
	assert.Equal(t, tui.ViewStateInput, sslModel.state)
	assert.Nil(t, sslModel.result)
}

// Benchmark Tests

func BenchmarkSSLTUIModel_ViewRendering(b *testing.B) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	// Set up result state
	validResult := createTestValidSSLResult()
	model.state = tui.ViewStateResult
	model.result = &validResult
	model.SetSize(80, 40)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

func BenchmarkSSLTUIModel_Update(b *testing.B) {
	mockClient := network.NewMockClient()
	mockLogger := &SimpleMockLogger{}
	
	tool := NewTool(mockClient, mockLogger)
	model := NewModel(tool)
	
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.Update(keyMsg)
	}
}