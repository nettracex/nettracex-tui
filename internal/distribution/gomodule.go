package distribution

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// GoModulePublisher handles Go module publishing to pkg.go.dev
type GoModulePublisher struct {
	config     GoModuleConfig
	client     *http.Client
	status     PublishStatus
	validator  *GoModuleValidator
	
	// Function fields for testing
	createGitTag              func(ctx context.Context, release Release) error
	verifyModuleAvailability  func(ctx context.Context, release Release) error
}

// GoModuleConfig contains configuration for Go module publishing
type GoModuleConfig struct {
	ModulePath    string            `json:"module_path"`
	ProxyURL      string            `json:"proxy_url"`
	SumDBURL      string            `json:"sumdb_url"`
	Timeout       time.Duration     `json:"timeout"`
	Documentation DocumentationConfig `json:"documentation"`
	Examples      ExamplesConfig    `json:"examples"`
	Metadata      map[string]string `json:"metadata"`
}

// DocumentationConfig contains documentation generation settings
type DocumentationConfig struct {
	GenerateReadme   bool     `json:"generate_readme"`
	GenerateExamples bool     `json:"generate_examples"`
	IncludeBadges    bool     `json:"include_badges"`
	BadgeTypes       []string `json:"badge_types"`
	UpdateGoDoc      bool     `json:"update_godoc"`
}

// ExamplesConfig contains example generation settings
type ExamplesConfig struct {
	AutoGenerate    bool     `json:"auto_generate"`
	ExampleDirs     []string `json:"example_dirs"`
	TestExamples    bool     `json:"test_examples"`
	IncludeOutput   bool     `json:"include_output"`
}

// GoModuleValidator validates Go modules
type GoModuleValidator struct {
	config GoModuleValidatorConfig
	
	// Function field for testing
	ValidateRelease func(ctx context.Context, release Release) (*PackageValidationResult, error)
}

// GoModuleValidatorConfig contains validator configuration
type GoModuleValidatorConfig struct {
	CheckSyntax      bool `json:"check_syntax"`
	CheckDependencies bool `json:"check_dependencies"`
	CheckLicense     bool `json:"check_license"`
	CheckDocumentation bool `json:"check_documentation"`
	MinCoverage      float64 `json:"min_coverage"`
}

// NewGoModulePublisher creates a new Go module publisher
func NewGoModulePublisher(config GoModuleConfig) *GoModulePublisher {
	gmp := &GoModulePublisher{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		status: PublishStatus{
			Name:   "gomodule",
			Status: StatusIdle,
		},
		validator: NewGoModuleValidator(GoModuleValidatorConfig{
			CheckSyntax:        true,
			CheckDependencies:  true,
			CheckLicense:       true,
			CheckDocumentation: true,
			MinCoverage:        80.0,
		}),
	}
	
	// Set default implementations
	gmp.createGitTag = gmp.defaultCreateGitTag
	gmp.verifyModuleAvailability = gmp.defaultVerifyModuleAvailability
	
	return gmp
}

// NewGoModuleValidator creates a new Go module validator
func NewGoModuleValidator(config GoModuleValidatorConfig) *GoModuleValidator {
	gmv := &GoModuleValidator{
		config: config,
	}
	
	// Set default implementation
	gmv.ValidateRelease = gmv.defaultValidateRelease
	
	return gmv
}

// GetName returns the publisher name
func (gmp *GoModulePublisher) GetName() string {
	return "gomodule"
}

// GetStatus returns the current publisher status
func (gmp *GoModulePublisher) GetStatus() PublishStatus {
	return gmp.status
}

// Validate validates the release for Go module publishing
func (gmp *GoModulePublisher) Validate(ctx context.Context, release Release) error {
	gmp.updateStatus(StatusPublishing, "")
	
	result, err := gmp.validator.ValidateRelease(ctx, release)
	if err != nil {
		gmp.updateStatus(StatusError, err.Error())
		return err
	}
	
	if !result.Valid {
		errMsg := fmt.Sprintf("validation failed: %d errors", len(result.Errors))
		gmp.updateStatus(StatusError, errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	
	// Reset status to idle after successful validation
	gmp.updateStatus(StatusIdle, "")
	return nil
}

// Publish publishes the Go module to pkg.go.dev
func (gmp *GoModulePublisher) Publish(ctx context.Context, release Release) error {
	gmp.updateStatus(StatusPublishing, "")
	
	// Validate first
	if err := gmp.Validate(ctx, release); err != nil {
		return err
	}
	
	// Generate documentation and examples
	if err := gmp.generateDocumentation(ctx, release); err != nil {
		gmp.updateStatus(StatusError, err.Error())
		return fmt.Errorf("documentation generation failed: %w", err)
	}
	
	// Create and push git tag
	if err := gmp.createGitTag(ctx, release); err != nil {
		gmp.updateStatus(StatusError, err.Error())
		return fmt.Errorf("git tag creation failed: %w", err)
	}
	
	// Trigger module proxy update
	if err := gmp.triggerProxyUpdate(ctx, release); err != nil {
		gmp.updateStatus(StatusError, err.Error())
		return fmt.Errorf("proxy update failed: %w", err)
	}
	
	// Verify module availability
	if err := gmp.verifyModuleAvailability(ctx, release); err != nil {
		gmp.updateStatus(StatusError, err.Error())
		return fmt.Errorf("module verification failed: %w", err)
	}
	
	gmp.updateStatus(StatusSuccess, "")
	return nil
}

// defaultValidateRelease validates a release for Go module publishing
func (gmv *GoModuleValidator) defaultValidateRelease(ctx context.Context, release Release) (*PackageValidationResult, error) {
	result := &PackageValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
		Metadata: make(map[string]string),
	}
	
	// Check version format
	if err := gmv.validateVersion(release.Version); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Code:     "INVALID_VERSION",
			Message:  err.Error(),
			Field:    "version",
			Severity: "error",
		})
		result.Valid = false
	}
	
	// Check Go module syntax
	if gmv.config.CheckSyntax {
		if err := gmv.validateSyntax(ctx); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Code:     "SYNTAX_ERROR",
				Message:  err.Error(),
				Field:    "syntax",
				Severity: "error",
			})
			result.Valid = false
		}
	}
	
	// Check dependencies
	if gmv.config.CheckDependencies {
		if warnings := gmv.validateDependencies(ctx); len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	}
	
	// Check license
	if gmv.config.CheckLicense {
		if err := gmv.validateLicense(); err != nil {
			result.Warnings = append(result.Warnings, ValidationError{
				Code:     "MISSING_LICENSE",
				Message:  err.Error(),
				Field:    "license",
				Severity: "warning",
			})
		}
	}
	
	// Check documentation
	if gmv.config.CheckDocumentation {
		if warnings := gmv.validateDocumentation(); len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	}
	
	return result, nil
}

// validateVersion validates semantic version format
func (gmv *GoModuleValidator) validateVersion(version string) error {
	// Semantic version regex
	semverRegex := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	
	if !semverRegex.MatchString(version) {
		return fmt.Errorf("invalid semantic version format: %s", version)
	}
	
	return nil
}

// validateSyntax validates Go syntax
func (gmv *GoModuleValidator) validateSyntax(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "vet", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go vet failed: %s", string(output))
	}
	
	cmd = exec.CommandContext(ctx, "go", "build", "./...")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %s", string(output))
	}
	
	return nil
}

// validateDependencies validates module dependencies
func (gmv *GoModuleValidator) validateDependencies(ctx context.Context) []ValidationError {
	var warnings []ValidationError
	
	// Check for outdated dependencies
	cmd := exec.CommandContext(ctx, "go", "list", "-u", "-m", "all")
	output, err := cmd.Output()
	if err != nil {
		warnings = append(warnings, ValidationError{
			Code:     "DEPENDENCY_CHECK_FAILED",
			Message:  "Failed to check dependencies",
			Field:    "dependencies",
			Severity: "warning",
		})
		return warnings
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			warnings = append(warnings, ValidationError{
				Code:     "OUTDATED_DEPENDENCY",
				Message:  fmt.Sprintf("Outdated dependency: %s", line),
				Field:    "dependencies",
				Severity: "warning",
			})
		}
	}
	
	return warnings
}

// validateLicense validates license file presence
func (gmv *GoModuleValidator) validateLicense() error {
	licenseFiles := []string{"LICENSE", "LICENSE.txt", "LICENSE.md", "COPYING"}
	
	for _, filename := range licenseFiles {
		if _, err := os.Stat(filename); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("no license file found")
}

// validateDocumentation validates documentation completeness
func (gmv *GoModuleValidator) validateDocumentation() []ValidationError {
	var warnings []ValidationError
	
	// Check for README
	readmeFiles := []string{"README.md", "README.txt", "README"}
	hasReadme := false
	for _, filename := range readmeFiles {
		if _, err := os.Stat(filename); err == nil {
			hasReadme = true
			break
		}
	}
	
	if !hasReadme {
		warnings = append(warnings, ValidationError{
			Code:     "MISSING_README",
			Message:  "No README file found",
			Field:    "documentation",
			Severity: "warning",
		})
	}
	
	// Check for package documentation
	cmd := exec.Command("go", "doc", ".")
	if err := cmd.Run(); err != nil {
		warnings = append(warnings, ValidationError{
			Code:     "MISSING_PACKAGE_DOC",
			Message:  "Package documentation missing or incomplete",
			Field:    "documentation",
			Severity: "warning",
		})
	}
	
	return warnings
}

// generateDocumentation generates documentation and examples
func (gmp *GoModulePublisher) generateDocumentation(ctx context.Context, release Release) error {
	if gmp.config.Documentation.GenerateReadme {
		if err := gmp.generateReadme(release); err != nil {
			return fmt.Errorf("README generation failed: %w", err)
		}
	}
	
	if gmp.config.Documentation.GenerateExamples {
		if err := gmp.generateExamples(ctx, release); err != nil {
			return fmt.Errorf("examples generation failed: %w", err)
		}
	}
	
	if gmp.config.Documentation.UpdateGoDoc {
		if err := gmp.updateGoDoc(ctx, release); err != nil {
			return fmt.Errorf("GoDoc update failed: %w", err)
		}
	}
	
	return nil
}

// generateReadme generates or updates README.md
func (gmp *GoModulePublisher) generateReadme(release Release) error {
	readmePath := "README.md"
	
	// Check if README already exists
	if _, err := os.Stat(readmePath); err == nil {
		// Update existing README with badges and version info
		return gmp.updateReadmeBadges(readmePath, release)
	}
	
	// Generate new README
	readme := gmp.generateReadmeContent(release)
	return os.WriteFile(readmePath, []byte(readme), 0644)
}

// generateReadmeContent generates README content
func (gmp *GoModulePublisher) generateReadmeContent(release Release) string {
	var content strings.Builder
	
	content.WriteString(fmt.Sprintf("# %s\n\n", filepath.Base(gmp.config.ModulePath)))
	
	// Add badges if configured
	if gmp.config.Documentation.IncludeBadges {
		content.WriteString(gmp.generateBadges(release))
		content.WriteString("\n\n")
	}
	
	content.WriteString("## Installation\n\n")
	content.WriteString(fmt.Sprintf("```bash\ngo get %s@%s\n```\n\n", gmp.config.ModulePath, release.Version))
	
	content.WriteString("## Usage\n\n")
	content.WriteString("```go\n")
	content.WriteString(fmt.Sprintf("import \"%s\"\n", gmp.config.ModulePath))
	content.WriteString("```\n\n")
	
	content.WriteString("## Documentation\n\n")
	content.WriteString(fmt.Sprintf("Full documentation is available at [pkg.go.dev](%s).\n\n", 
		fmt.Sprintf("https://pkg.go.dev/%s", gmp.config.ModulePath)))
	
	return content.String()
}

// generateBadges generates documentation badges
func (gmp *GoModulePublisher) generateBadges(release Release) string {
	var badges strings.Builder
	
	for _, badgeType := range gmp.config.Documentation.BadgeTypes {
		switch badgeType {
		case "go-version":
			badges.WriteString(fmt.Sprintf("[![Go Version](https://img.shields.io/github/go-mod/go-version/%s)](https://golang.org/)\n", 
				strings.TrimPrefix(gmp.config.ModulePath, "github.com/")))
		case "release":
			badges.WriteString(fmt.Sprintf("[![Release](https://img.shields.io/github/v/release/%s)](https://github.com/%s/releases)\n", 
				strings.TrimPrefix(gmp.config.ModulePath, "github.com/"),
				strings.TrimPrefix(gmp.config.ModulePath, "github.com/")))
		case "license":
			badges.WriteString(fmt.Sprintf("[![License](https://img.shields.io/github/license/%s)](LICENSE)\n", 
				strings.TrimPrefix(gmp.config.ModulePath, "github.com/")))
		case "go-report":
			badges.WriteString(fmt.Sprintf("[![Go Report Card](https://goreportcard.com/badge/%s)](https://goreportcard.com/report/%s)\n", 
				gmp.config.ModulePath, gmp.config.ModulePath))
		case "pkg-go-dev":
			badges.WriteString(fmt.Sprintf("[![PkgGoDev](https://pkg.go.dev/badge/%s)](https://pkg.go.dev/%s)\n", 
				gmp.config.ModulePath, gmp.config.ModulePath))
		}
	}
	
	return badges.String()
}

// updateReadmeBadges updates badges in existing README
func (gmp *GoModulePublisher) updateReadmeBadges(readmePath string, release Release) error {
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	
	// Simple badge update - in a real implementation, this would be more sophisticated
	updatedContent := string(content)
	
	// Update version references
	versionRegex := regexp.MustCompile(`@v\d+\.\d+\.\d+`)
	updatedContent = versionRegex.ReplaceAllString(updatedContent, "@"+release.Version)
	
	return os.WriteFile(readmePath, []byte(updatedContent), 0644)
}

// generateExamples generates code examples
func (gmp *GoModulePublisher) generateExamples(ctx context.Context, release Release) error {
	if !gmp.config.Examples.AutoGenerate {
		return nil
	}
	
	// Create examples directory if it doesn't exist
	examplesDir := "examples"
	if err := os.MkdirAll(examplesDir, 0755); err != nil {
		return err
	}
	
	// Generate basic example
	basicExample := gmp.generateBasicExample(release)
	examplePath := filepath.Join(examplesDir, "basic", "main.go")
	if err := os.MkdirAll(filepath.Dir(examplePath), 0755); err != nil {
		return err
	}
	
	if err := os.WriteFile(examplePath, []byte(basicExample), 0644); err != nil {
		return err
	}
	
	// Test examples if configured
	if gmp.config.Examples.TestExamples {
		return gmp.testExamples(ctx, examplesDir)
	}
	
	return nil
}

// generateBasicExample generates a basic usage example
func (gmp *GoModulePublisher) generateBasicExample(release Release) string {
	var example strings.Builder
	
	example.WriteString("package main\n\n")
	example.WriteString("import (\n")
	example.WriteString("\t\"fmt\"\n")
	example.WriteString(fmt.Sprintf("\t\"%s\"\n", gmp.config.ModulePath))
	example.WriteString(")\n\n")
	example.WriteString("func main() {\n")
	example.WriteString("\t// Basic usage example\n")
	example.WriteString("\tfmt.Println(\"Hello from NetTraceX!\")\n")
	example.WriteString("}\n")
	
	return example.String()
}

// testExamples tests generated examples
func (gmp *GoModulePublisher) testExamples(ctx context.Context, examplesDir string) error {
	return filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, "main.go") {
			dir := filepath.Dir(path)
			cmd := exec.CommandContext(ctx, "go", "run", "main.go")
			cmd.Dir = dir
			
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("example test failed in %s: %s", dir, string(output))
			}
		}
		
		return nil
	})
}

// updateGoDoc updates Go documentation
func (gmp *GoModulePublisher) updateGoDoc(ctx context.Context, release Release) error {
	// Ensure all packages have proper documentation
	cmd := exec.CommandContext(ctx, "go", "list", "./...")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	packages := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, pkg := range packages {
		if err := gmp.validatePackageDoc(ctx, pkg); err != nil {
			return fmt.Errorf("package %s documentation validation failed: %w", pkg, err)
		}
	}
	
	return nil
}

// validatePackageDoc validates package documentation
func (gmp *GoModulePublisher) validatePackageDoc(ctx context.Context, pkg string) error {
	cmd := exec.CommandContext(ctx, "go", "doc", pkg)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	// Check if package has documentation
	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("package %s has no documentation", pkg)
	}
	
	return nil
}

// defaultCreateGitTag creates and pushes a git tag for the release
func (gmp *GoModulePublisher) defaultCreateGitTag(ctx context.Context, release Release) error {
	// Create tag
	cmd := exec.CommandContext(ctx, "git", "tag", "-a", release.Tag, "-m", fmt.Sprintf("Release %s", release.Version))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create git tag: %w", err)
	}
	
	// Push tag
	cmd = exec.CommandContext(ctx, "git", "push", "origin", release.Tag)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push git tag: %w", err)
	}
	
	return nil
}

// triggerProxyUpdate triggers Go module proxy to fetch the new version
func (gmp *GoModulePublisher) triggerProxyUpdate(ctx context.Context, release Release) error {
	// Request module info from proxy to trigger update
	url := fmt.Sprintf("%s/%s/@v/%s.info", gmp.config.ProxyURL, gmp.config.ModulePath, release.Version)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := gmp.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("proxy update failed: %s", string(body))
	}
	
	return nil
}

// defaultVerifyModuleAvailability verifies the module is available on pkg.go.dev
func (gmp *GoModulePublisher) defaultVerifyModuleAvailability(ctx context.Context, release Release) error {
	// Wait a bit for propagation
	time.Sleep(30 * time.Second)
	
	// Check if module is available
	url := fmt.Sprintf("https://pkg.go.dev/%s@%s", gmp.config.ModulePath, release.Version)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	
	resp, err := gmp.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("module not available on pkg.go.dev: status %d", resp.StatusCode)
	}
	
	return nil
}

// updateStatus updates the publisher status
func (gmp *GoModulePublisher) updateStatus(status StatusType, errorMsg string) {
	gmp.status.Status = status
	gmp.status.LastError = errorMsg
	if status == StatusSuccess {
		gmp.status.PublishCount++
		gmp.status.LastPublish = time.Now()
	} else if status == StatusError {
		gmp.status.ErrorCount++
	}
}

// GetName returns the validator name
func (gmv *GoModuleValidator) GetName() string {
	return "gomodule"
}

// Validate validates a release
func (gmv *GoModuleValidator) Validate(ctx context.Context, release Release) error {
	result, err := gmv.ValidateRelease(ctx, release)
	if err != nil {
		return err
	}
	
	if !result.Valid {
		return fmt.Errorf("validation failed with %d errors", len(result.Errors))
	}
	
	return nil
}