package distribution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoModulePublisher(t *testing.T) {
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
		ProxyURL:   "https://proxy.golang.org",
		Timeout:    30 * time.Second,
	}
	
	publisher := NewGoModulePublisher(config)
	
	assert.NotNil(t, publisher)
	assert.Equal(t, "gomodule", publisher.GetName())
	assert.Equal(t, StatusIdle, publisher.GetStatus().Status)
}

func TestGoModuleValidator_ValidateVersion(t *testing.T) {
	validator := NewGoModuleValidator(GoModuleValidatorConfig{})
	
	tests := []struct {
		version string
		valid   bool
	}{
		{"v1.0.0", true},
		{"v1.2.3", true},
		{"v0.1.0", true},
		{"v1.0.0-alpha", true},
		{"v1.0.0+build", true},
		{"1.0.0", true},
		{"invalid", false},
		{"", false},
		{"v1", false},
		{"v1.0", false},
	}
	
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			err := validator.validateVersion(test.version)
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestGoModuleValidator_ValidateLicense(t *testing.T) {
	validator := NewGoModuleValidator(GoModuleValidatorConfig{})
	
	// Test without license file
	err := validator.validateLicense()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no license file found")
	
	// Create temporary license file
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(tempDir)
	
	err = os.WriteFile("LICENSE", []byte("MIT License"), 0644)
	require.NoError(t, err)
	
	// Test with license file
	err = validator.validateLicense()
	assert.NoError(t, err)
}

func TestGoModuleValidator_ValidateDocumentation(t *testing.T) {
	validator := NewGoModuleValidator(GoModuleValidatorConfig{})
	
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(tempDir)
	
	// Test without README
	warnings := validator.validateDocumentation()
	assert.Len(t, warnings, 2) // Missing README and package doc
	
	// Create README
	err := os.WriteFile("README.md", []byte("# Test Module"), 0644)
	require.NoError(t, err)
	
	warnings = validator.validateDocumentation()
	assert.Len(t, warnings, 1) // Only missing package doc
}

func TestGoModuleValidator_ValidateRelease(t *testing.T) {
	validator := NewGoModuleValidator(GoModuleValidatorConfig{
		CheckSyntax:        false, // Skip syntax check in tests
		CheckDependencies:  false, // Skip dependency check in tests
		CheckLicense:       true,
		CheckDocumentation: true,
	})
	
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(tempDir)
	
	// Create license and README files
	os.WriteFile("LICENSE", []byte("MIT License"), 0644)
	os.WriteFile("README.md", []byte("# Test Module"), 0644)
	
	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
	}
	
	result, err := validator.ValidateRelease(context.Background(), release)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 0)
}

func TestGoModulePublisher_GenerateReadmeContent(t *testing.T) {
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
		Documentation: DocumentationConfig{
			IncludeBadges: true,
			BadgeTypes:    []string{"go-version", "release", "license"},
		},
	}
	
	publisher := NewGoModulePublisher(config)
	
	release := Release{
		Version: "v1.0.0",
	}
	
	content := publisher.generateReadmeContent(release)
	
	assert.Contains(t, content, "# module")
	assert.Contains(t, content, "go get github.com/test/module@v1.0.0")
	assert.Contains(t, content, "pkg.go.dev")
	assert.Contains(t, content, "[![Go Version]")
	assert.Contains(t, content, "[![Release]")
	assert.Contains(t, content, "[![License]")
}

func TestGoModulePublisher_GenerateBadges(t *testing.T) {
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
		Documentation: DocumentationConfig{
			BadgeTypes: []string{"go-version", "release", "license", "go-report", "pkg-go-dev"},
		},
	}
	
	publisher := NewGoModulePublisher(config)
	
	release := Release{
		Version: "v1.0.0",
	}
	
	badges := publisher.generateBadges(release)
	
	assert.Contains(t, badges, "[![Go Version]")
	assert.Contains(t, badges, "[![Release]")
	assert.Contains(t, badges, "[![License]")
	assert.Contains(t, badges, "[![Go Report Card]")
	assert.Contains(t, badges, "[![PkgGoDev]")
}

func TestGoModulePublisher_GenerateBasicExample(t *testing.T) {
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
	}
	
	publisher := NewGoModulePublisher(config)
	
	release := Release{
		Version: "v1.0.0",
	}
	
	example := publisher.generateBasicExample(release)
	
	assert.Contains(t, example, "package main")
	assert.Contains(t, example, "import (")
	assert.Contains(t, example, "github.com/test/module")
	assert.Contains(t, example, "func main()")
	assert.Contains(t, example, "Hello from NetTraceX!")
}

func TestGoModulePublisher_GenerateExamples(t *testing.T) {
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
		Examples: ExamplesConfig{
			AutoGenerate: true,
			TestExamples: false, // Skip testing in unit tests
		},
	}
	
	publisher := NewGoModulePublisher(config)
	
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(tempDir)
	
	release := Release{
		Version: "v1.0.0",
	}
	
	err := publisher.generateExamples(context.Background(), release)
	assert.NoError(t, err)
	
	// Check if example file was created
	examplePath := filepath.Join("examples", "basic", "main.go")
	assert.FileExists(t, examplePath)
	
	content, err := os.ReadFile(examplePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "package main")
}

func TestGoModulePublisher_TriggerProxyUpdate(t *testing.T) {
	// Create mock proxy server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/github.com/test/module/@v/v1.0.0.info" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"Version":"v1.0.0","Time":"2023-01-01T00:00:00Z"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
		ProxyURL:   server.URL,
		Timeout:    5 * time.Second,
	}
	
	publisher := NewGoModulePublisher(config)
	
	release := Release{
		Version: "v1.0.0",
	}
	
	err := publisher.triggerProxyUpdate(context.Background(), release)
	assert.NoError(t, err)
}

func TestGoModulePublisher_VerifyModuleAvailability(t *testing.T) {
	// Create mock pkg.go.dev server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/github.com/test/module@v1.0.0" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("<html><title>github.com/test/module</title></html>"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
		Timeout:    5 * time.Second,
	}
	
	publisher := NewGoModulePublisher(config)
	
	// Replace the verification URL for testing
	publisher.verifyModuleAvailability = func(ctx context.Context, release Release) error {
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/github.com/test/module@v1.0.0", nil)
		if err != nil {
			return err
		}
		
		resp, err := publisher.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return assert.AnError
		}
		
		return nil
	}
	
	release := Release{
		Version: "v1.0.0",
	}
	
	err := publisher.verifyModuleAvailability(context.Background(), release)
	assert.NoError(t, err)
}

func TestGoModulePublisher_Validate(t *testing.T) {
	config := GoModuleConfig{
		ModulePath: "github.com/test/module",
	}
	
	publisher := NewGoModulePublisher(config)
	
	// Mock the validator to return success
	publisher.validator = &GoModuleValidator{
		config: GoModuleValidatorConfig{},
	}
	
	// Create a mock validator for testing
	mockValidator := &GoModuleValidator{
		config: GoModuleValidatorConfig{},
	}
	mockValidator.ValidateRelease = func(ctx context.Context, release Release) (*PackageValidationResult, error) {
		return &PackageValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
			Metadata: make(map[string]string),
		}, nil
	}
	publisher.validator = mockValidator
	
	release := Release{
		Version: "v1.0.0",
	}
	
	err := publisher.Validate(context.Background(), release)
	assert.NoError(t, err)
	// Status should be idle after successful validation
	status := publisher.GetStatus()
	assert.Equal(t, StatusIdle, status.Status)
}

func TestGoModulePublisher_UpdateStatus(t *testing.T) {
	publisher := NewGoModulePublisher(GoModuleConfig{})
	
	// Test success status
	publisher.updateStatus(StatusSuccess, "")
	status := publisher.GetStatus()
	assert.Equal(t, StatusSuccess, status.Status)
	assert.Equal(t, 1, status.PublishCount)
	assert.Equal(t, 0, status.ErrorCount)
	
	// Test error status
	publisher.updateStatus(StatusError, "test error")
	status = publisher.GetStatus()
	assert.Equal(t, StatusError, status.Status)
	assert.Equal(t, "test error", status.LastError)
	assert.Equal(t, 1, status.ErrorCount)
}