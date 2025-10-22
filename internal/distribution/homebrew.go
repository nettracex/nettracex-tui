package distribution

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// HomebrewPublisher manages Homebrew formula creation and publishing
type HomebrewPublisher struct {
	config    HomebrewConfig
	validator *HomebrewValidator
	client    *http.Client
	status    PublishStatus
}

// HomebrewConfig contains Homebrew publishing configuration
type HomebrewConfig struct {
	TapRepo      string `json:"tap_repo"`      // e.g., "username/homebrew-tap"
	FormulaName  string `json:"formula_name"`  // e.g., "nettracex"
	GitHubToken  string `json:"github_token"`
	Description  string `json:"description"`
	Homepage     string `json:"homepage"`
	License      string `json:"license"`
	TestCommand  string `json:"test_command"`
	Dependencies []string `json:"dependencies"`
	CustomTap    bool   `json:"custom_tap"`    // true for custom tap, false for homebrew-core
}

// HomebrewFormula represents a complete Homebrew formula
type HomebrewFormula struct {
	Class        string            `json:"class"`
	Description  string            `json:"description"`
	Homepage     string            `json:"homepage"`
	URL          string            `json:"url"`
	SHA256       string            `json:"sha256"`
	License      string            `json:"license"`
	Dependencies []string          `json:"dependencies"`
	TestBlock    string            `json:"test_block"`
	Version      string            `json:"version"`
	Metadata     map[string]string `json:"metadata"`
	// Multi-platform support
	PlatformURLs map[string]PlatformBinary `json:"platform_urls,omitempty"`
}

// PlatformBinary represents a platform-specific binary
type PlatformBinary struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// HomebrewValidator validates Homebrew formulas
type HomebrewValidator struct {
	brewPath string
}

// NewHomebrewPublisher creates a new Homebrew publisher
func NewHomebrewPublisher(config HomebrewConfig) (*HomebrewPublisher, error) {
	validator, err := NewHomebrewValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create Homebrew validator: %w", err)
	}

	return &HomebrewPublisher{
		config:    config,
		validator: validator,
		client:    &http.Client{Timeout: 30 * time.Second},
		status: PublishStatus{
			Name:   "homebrew",
			Status: StatusIdle,
		},
	}, nil
}

// NewHomebrewValidator creates a new Homebrew validator
func NewHomebrewValidator() (*HomebrewValidator, error) {
	brewPath, err := exec.LookPath("brew")
	if err != nil {
		// Homebrew not installed, validator will work in limited mode
		brewPath = ""
	}

	return &HomebrewValidator{
		brewPath: brewPath,
	}, nil
}

// GetName returns the publisher name
func (p *HomebrewPublisher) GetName() string {
	return "homebrew"
}

// Publish publishes a release to Homebrew
func (p *HomebrewPublisher) Publish(ctx context.Context, release Release) error {
	p.updateStatus(StatusPublishing, "")
	
	// Generate formula
	formula, err := p.GenerateFormula(release)
	if err != nil {
		p.updateStatus(StatusError, err.Error())
		return fmt.Errorf("failed to generate formula: %w", err)
	}

	// Validate formula
	if err := p.validator.ValidateFormula(formula); err != nil {
		return fmt.Errorf("formula validation failed: %w", err)
	}

	// Write formula to file for testing
	formulaContent, err := p.RenderFormula(formula)
	if err != nil {
		return fmt.Errorf("failed to render formula: %w", err)
	}

	// Test formula installation if Homebrew is available
	if p.validator.brewPath != "" {
		if err := p.testFormulaInstallation(ctx, formulaContent); err != nil {
			// Log warning but don't fail the publish
			fmt.Printf("Warning: formula installation test failed: %v\n", err)
		}
	}

	// Submit to tap repository or homebrew-core
	_, err = p.submitFormula(ctx, formula, formulaContent)
	if err != nil {
		p.updateStatus(StatusError, err.Error())
		return fmt.Errorf("failed to submit formula: %w", err)
	}

	p.updateStatus(StatusSuccess, "")
	return nil
}

// Validate validates a release for Homebrew publishing
func (p *HomebrewPublisher) Validate(ctx context.Context, release Release) error {
	// Check for supported platform binaries (macOS and Linux)
	var supportedBinary *Binary
	for _, binary := range release.Binaries {
		if binary.Platform == "darwin" || binary.Platform == "linux" {
			supportedBinary = &binary
			break
		}
	}

	if supportedBinary == nil {
		return fmt.Errorf("no supported binary found in release (macOS or Linux required)")
	}

	// Validate binary URL is accessible
	if supportedBinary.DownloadURL == "" {
		return fmt.Errorf("binary missing download URL")
	}

	// Validate checksum
	if supportedBinary.Checksum == "" {
		return fmt.Errorf("binary missing checksum")
	}

	// Validate version format
	if release.Version == "" {
		return fmt.Errorf("release version is required")
	}

	return nil
}

// GetStatus returns the current status of the Homebrew publisher
func (p *HomebrewPublisher) GetStatus() PublishStatus {
	p.status.Metadata = map[string]string{
		"tap_repo":       p.config.TapRepo,
		"formula":        p.config.FormulaName,
		"custom_tap":     fmt.Sprintf("%t", p.config.CustomTap),
		"brew_available": fmt.Sprintf("%t", p.validator.brewPath != ""),
	}
	return p.status
}

// updateStatus updates the publisher status
func (p *HomebrewPublisher) updateStatus(status StatusType, lastError string) {
	p.status.Status = status
	p.status.LastError = lastError
	if status == StatusSuccess {
		p.status.LastPublish = time.Now()
		p.status.PublishCount++
	} else if status == StatusError {
		p.status.ErrorCount++
	}
}

// GenerateFormula creates a Homebrew formula from a release
func (p *HomebrewPublisher) GenerateFormula(release Release) (*HomebrewFormula, error) {
	// Collect all supported platform binaries
	supportedBinaries := make(map[string]*Binary)
	platformURLs := make(map[string]PlatformBinary)
	
	for _, binary := range release.Binaries {
		if binary.Platform == "darwin" || binary.Platform == "linux" {
			key := fmt.Sprintf("%s-%s", binary.Platform, binary.Architecture)
			supportedBinaries[key] = &binary
			
			// Calculate SHA256 if not provided
			sha256Hash := binary.Checksum
			if sha256Hash == "" {
				hash, err := p.calculateSHA256(binary.DownloadURL)
				if err != nil {
					return nil, fmt.Errorf("failed to calculate SHA256 for %s: %w", key, err)
				}
				sha256Hash = hash
			}
			
			platformURLs[key] = PlatformBinary{
				URL:    binary.DownloadURL,
				SHA256: sha256Hash,
			}
		}
	}

	if len(supportedBinaries) == 0 {
		return nil, fmt.Errorf("no supported binaries found (macOS or Linux required)")
	}

	// Select primary binary (prefer macOS, then Linux)
	var primaryBinary *Binary
	var primaryKey string
	
	// Try macOS first (amd64, then arm64)
	for _, arch := range []string{"amd64", "arm64"} {
		key := fmt.Sprintf("darwin-%s", arch)
		if binary, exists := supportedBinaries[key]; exists {
			primaryBinary = binary
			primaryKey = key
			break
		}
	}
	
	// Fallback to Linux if no macOS binary
	if primaryBinary == nil {
		for _, arch := range []string{"amd64", "arm64"} {
			key := fmt.Sprintf("linux-%s", arch)
			if binary, exists := supportedBinaries[key]; exists {
				primaryBinary = binary
				primaryKey = key
				break
			}
		}
	}

	if primaryBinary == nil {
		return nil, fmt.Errorf("no primary binary found")
	}

	// Generate class name (capitalize first letter)
	className := strings.Title(strings.ToLower(p.config.FormulaName))

	formula := &HomebrewFormula{
		Class:        className,
		Description:  p.config.Description,
		Homepage:     p.config.Homepage,
		URL:          primaryBinary.DownloadURL,
		SHA256:       platformURLs[primaryKey].SHA256,
		License:      p.config.License,
		Dependencies: p.config.Dependencies,
		TestBlock:    p.generateTestBlock(),
		Version:      release.Version,
		PlatformURLs: platformURLs,
		Metadata: map[string]string{
			"binary_name":      primaryBinary.Filename,
			"platform":         primaryBinary.Platform,
			"arch":             primaryBinary.Architecture,
			"primary_key":      primaryKey,
			"supported_count":  fmt.Sprintf("%d", len(supportedBinaries)),
		},
	}

	return formula, nil
}

// calculateSHA256 downloads a file and calculates its SHA256 hash
func (p *HomebrewPublisher) calculateSHA256(url string) (string, error) {
	resp, err := p.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: %s", resp.Status)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, resp.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// generateTestBlock creates the test block for the formula
func (p *HomebrewPublisher) generateTestBlock() string {
	if p.config.TestCommand != "" {
		return fmt.Sprintf(`system "#{bin}/%s", %s`, p.config.FormulaName, p.config.TestCommand)
	}
	return fmt.Sprintf(`system "#{bin}/%s", "--version"`, p.config.FormulaName)
}

// RenderFormula renders the formula to Ruby code
func (p *HomebrewPublisher) RenderFormula(formula *HomebrewFormula) (string, error) {
	// Check if we have multiple platforms
	hasMultiplePlatforms := len(formula.PlatformURLs) > 1
	
	var tmpl string
	if hasMultiplePlatforms {
		// Multi-platform template
		tmpl = `class {{ .Class }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  license "{{ .License }}"
  version "{{ .Version }}"

{{- if .Dependencies }}
{{- range .Dependencies }}
  depends_on "{{ . }}"
{{- end }}
{{- end }}

{{- range $key, $binary := .PlatformURLs }}
{{- if eq $key "darwin-amd64" }}
  if OS.mac? && Hardware::CPU.intel?
    url "{{ $binary.URL }}"
    sha256 "{{ $binary.SHA256 }}"
  end
{{- else if eq $key "darwin-arm64" }}
  if OS.mac? && Hardware::CPU.arm?
    url "{{ $binary.URL }}"
    sha256 "{{ $binary.SHA256 }}"
  end
{{- else if eq $key "linux-amd64" }}
  if OS.linux? && Hardware::CPU.intel?
    url "{{ $binary.URL }}"
    sha256 "{{ $binary.SHA256 }}"
  end
{{- else if eq $key "linux-arm64" }}
  if OS.linux? && Hardware::CPU.arm?
    url "{{ $binary.URL }}"
    sha256 "{{ $binary.SHA256 }}"
  end
{{- end }}
{{- end }}

  def install
    bin.install "{{ .Metadata.binary_name }}" => "{{ .Class | lower }}"
  end

  test do
    {{ .TestBlock }}
  end
end`
	} else {
		// Single platform template (backward compatibility)
		tmpl = `class {{ .Class }} < Formula
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  url "{{ .URL }}"
  sha256 "{{ .SHA256 }}"
  license "{{ .License }}"
  version "{{ .Version }}"

{{- if .Dependencies }}
{{- range .Dependencies }}
  depends_on "{{ . }}"
{{- end }}
{{- end }}

  def install
    bin.install "{{ .Metadata.binary_name }}" => "{{ .Class | lower }}"
  end

  test do
    {{ .TestBlock }}
  end
end`
	}

	t, err := template.New("formula").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, formula); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ValidateFormula validates a Homebrew formula
func (v *HomebrewValidator) ValidateFormula(formula *HomebrewFormula) error {
	// Basic validation
	if formula.Class == "" {
		return fmt.Errorf("formula class name is required")
	}

	if formula.Description == "" {
		return fmt.Errorf("formula description is required")
	}

	if formula.Homepage == "" {
		return fmt.Errorf("formula homepage is required")
	}

	if formula.URL == "" {
		return fmt.Errorf("formula URL is required")
	}

	if formula.SHA256 == "" {
		return fmt.Errorf("formula SHA256 is required")
	}

	if len(formula.SHA256) != 64 {
		return fmt.Errorf("invalid SHA256 hash length")
	}

	if formula.License == "" {
		return fmt.Errorf("formula license is required")
	}

	// Validate URL is accessible
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(formula.URL)
	if err != nil {
		return fmt.Errorf("formula URL not accessible: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("formula URL returned status: %s", resp.Status)
	}

	return nil
}

// testFormulaInstallation tests the formula installation process
func (p *HomebrewPublisher) testFormulaInstallation(ctx context.Context, formulaContent string) error {
	if p.validator.brewPath == "" {
		return fmt.Errorf("Homebrew not available for testing")
	}

	// Create temporary formula file
	tmpDir, err := os.MkdirTemp("", "homebrew-test-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	formulaFile := filepath.Join(tmpDir, fmt.Sprintf("%s.rb", strings.ToLower(p.config.FormulaName)))
	if err := os.WriteFile(formulaFile, []byte(formulaContent), 0644); err != nil {
		return fmt.Errorf("failed to write formula file: %w", err)
	}

	// Test formula syntax
	cmd := exec.CommandContext(ctx, p.validator.brewPath, "audit", "--strict", formulaFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("formula audit failed: %s", string(output))
	}

	return nil
}

// submitFormula submits the formula to the appropriate repository
func (p *HomebrewPublisher) submitFormula(ctx context.Context, formula *HomebrewFormula, content string) (string, error) {
	if p.config.CustomTap {
		return p.submitToCustomTap(ctx, formula, content)
	}
	return p.submitToHomebrewCore(ctx, formula, content)
}

// submitToCustomTap submits the formula to a custom tap repository
func (p *HomebrewPublisher) submitToCustomTap(ctx context.Context, formula *HomebrewFormula, content string) (string, error) {
	// For now, just save the formula to a local file
	// In a real implementation, this would create a PR to the tap repository
	
	formulaDir := "homebrew-formula"
	if err := os.MkdirAll(formulaDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create formula directory: %w", err)
	}

	formulaFile := filepath.Join(formulaDir, fmt.Sprintf("%s.rb", strings.ToLower(p.config.FormulaName)))
	if err := os.WriteFile(formulaFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write formula file: %w", err)
	}

	return fmt.Sprintf("file://%s", formulaFile), nil
}

// submitToHomebrewCore submits the formula to homebrew-core
func (p *HomebrewPublisher) submitToHomebrewCore(ctx context.Context, formula *HomebrewFormula, content string) (string, error) {
	// This would require creating a PR to homebrew-core
	// For now, we'll just validate and return a placeholder URL
	return "https://github.com/Homebrew/homebrew-core/pulls", nil
}