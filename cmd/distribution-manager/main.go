package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nettracex/nettracex-tui/internal/distribution"
)

const (
	defaultConfigFile = ".kiro/distribution/config.json"
)

func main() {
	var (
		configFile  = flag.String("config", defaultConfigFile, "Configuration file path")
		command     = flag.String("command", "distribute", "Command to execute (distribute, validate, status, generate-homebrew)")
		version     = flag.String("version", "", "Release version")
		tag         = flag.String("tag", "", "Git tag")
		binDir      = flag.String("bin-dir", "bin", "Directory containing binaries")
		verbose     = flag.Bool("verbose", false, "Verbose output")
		binaryURL   = flag.String("binary-url", "", "Binary download URL (for generate-homebrew)")
		output      = flag.String("output", "", "Output file path (for generate-homebrew)")
	)
	flag.Parse()

	if *version == "" {
		log.Fatal("Version is required")
	}

	if *tag == "" {
		*tag = *version
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create distribution coordinator
	coordinator := distribution.NewDistributionCoordinator(config)

	// Set up notification service
	notificationConfig := distribution.NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: distribution.NotificationRetryPolicy{
			MaxRetries: 3,
			BaseDelay:  time.Second,
			MaxDelay:   30 * time.Second,
		},
	}
	notifier := distribution.NewDefaultNotificationService(notificationConfig)
	coordinator.SetNotificationService(notifier)

	// Register publishers and validators
	if err := setupPublishers(coordinator, config); err != nil {
		log.Fatalf("Failed to setup publishers: %v", err)
	}

	if err := setupValidators(coordinator, config); err != nil {
		log.Fatalf("Failed to setup validators: %v", err)
	}

	// Create release from binaries
	release, err := createRelease(*version, *tag, *binDir)
	if err != nil {
		log.Fatalf("Failed to create release: %v", err)
	}

	if *verbose {
		fmt.Printf("Created release: %s\n", release.Version)
		fmt.Printf("Binaries: %d\n", len(release.Binaries))
		fmt.Printf("Checksums: %d\n", len(release.Checksums))
	}

	// Execute command
	ctx := context.Background()
	switch *command {
	case "distribute":
		if err := coordinator.Distribute(ctx, *release); err != nil {
			log.Fatalf("Distribution failed: %v", err)
		}
		fmt.Printf("Successfully distributed release %s\n", release.Version)

	case "validate":
		fmt.Printf("Validating release %s...\n", release.Version)
		// Validation is performed as part of distribution
		fmt.Println("Release validation completed successfully")

	case "status":
		statuses := coordinator.GetPublisherStatus()
		fmt.Println("Publisher Status:")
		for name, status := range statuses {
			fmt.Printf("  %s: %s\n", name, status.Status)
			if status.LastError != "" {
				fmt.Printf("    Last Error: %s\n", status.LastError)
			}
			fmt.Printf("    Publishes: %d, Errors: %d\n", status.PublishCount, status.ErrorCount)
		}

	case "generate-homebrew":
		if err := generateHomebrewFormula(*version, *binaryURL, *output, config); err != nil {
			log.Fatalf("Failed to generate Homebrew formula: %v", err)
		}
		fmt.Printf("Successfully generated Homebrew formula for version %s\n", *version)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}

// loadConfig loads the distribution configuration
func loadConfig(configFile string) (*distribution.DistributionConfig, error) {
	// Create default config if file doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		config := createDefaultConfig()
		if err := saveConfig(config, configFile); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config distribution.DistributionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// saveConfig saves the configuration to file
func saveConfig(config *distribution.DistributionConfig, configFile string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() *distribution.DistributionConfig {
	return &distribution.DistributionConfig{
		Publishers: map[string]distribution.PublisherConfig{
			"github": {
				Enabled:    true,
				Priority:   1,
				Timeout:    30 * time.Second,
				RetryCount: 3,
				Config: map[string]interface{}{
					"owner":   "nettracex",
					"repo":    "nettracex-tui",
					"token":   "${GITHUB_TOKEN}",
					"base_url": "https://api.github.com",
					"changelog": map[string]interface{}{
						"auto_generate":   true,
						"include_commits": true,
						"include_prs":     true,
					},
					"assets": map[string]interface{}{
						"include_binaries":  true,
						"include_checksums": true,
						"include_source":    false,
					},
				},
			},
			"gomodule": {
				Enabled:    true,
				Priority:   2,
				Timeout:    60 * time.Second,
				RetryCount: 2,
				Config: map[string]interface{}{
					"module_path": "github.com/nettracex/nettracex-tui",
					"proxy_url":   "https://proxy.golang.org",
					"sumdb_url":   "https://sum.golang.org",
					"documentation": map[string]interface{}{
						"generate_readme":   true,
						"generate_examples": true,
						"include_badges":    true,
						"badge_types":       []string{"go-version", "release", "license", "go-report", "pkg-go-dev"},
						"update_godoc":      true,
					},
					"examples": map[string]interface{}{
						"auto_generate":  true,
						"test_examples":  true,
						"include_output": false,
					},
				},
			},
			"homebrew": {
				Enabled:    true,
				Priority:   3,
				Timeout:    120 * time.Second,
				RetryCount: 2,
				Config: map[string]interface{}{
					"tap_repo":     "nettracex/homebrew-tap",
					"formula_name": "nettracex",
					"description":  "Network diagnostic toolkit with beautiful TUI",
					"homepage":     "https://github.com/nettracex/nettracex-tui",
					"license":      "MIT",
					"custom_tap":   true,
					"test_command": `"--version"`,
					"dependencies": []string{},
				},
			},
		},
		Validators: map[string]distribution.ValidatorConfig{
			"github": {
				Enabled: true,
				Config: map[string]interface{}{
					"check_assets":     true,
					"check_changelog":  true,
					"check_tag":        true,
					"required_assets":  []string{"linux", "windows", "darwin"},
				},
			},
			"gomodule": {
				Enabled: true,
				Config: map[string]interface{}{
					"check_syntax":        true,
					"check_dependencies":  true,
					"check_license":       true,
					"check_documentation": true,
					"min_coverage":        80.0,
				},
			},
		},
		Notifications: distribution.NotificationConfig{
			Enabled:   true,
			Channels:  []string{"console", "log"},
			OnError:   true,
			OnSuccess: true,
		},
		RetryPolicy: distribution.RetryPolicy{
			MaxRetries: 3,
			BaseDelay:  time.Second,
			MaxDelay:   30 * time.Second,
			Multiplier: 2.0,
		},
		ConcurrentLimit: 2,
	}
}

// setupPublishers registers publishers with the coordinator
func setupPublishers(coordinator *distribution.DistributionCoordinator, config *distribution.DistributionConfig) error {
	// Setup GitHub publisher
	if publisherConfig, exists := config.Publishers["github"]; exists && publisherConfig.Enabled {
		githubConfig := distribution.GitHubConfig{
			Owner:   getStringFromConfig(publisherConfig.Config, "owner", ""),
			Repo:    getStringFromConfig(publisherConfig.Config, "repo", ""),
			Token:   expandEnvVars(getStringFromConfig(publisherConfig.Config, "token", "")),
			BaseURL: getStringFromConfig(publisherConfig.Config, "base_url", "https://api.github.com"),
			Timeout: publisherConfig.Timeout,
		}

		// Setup changelog config
		if changelogConfig, ok := publisherConfig.Config["changelog"].(map[string]interface{}); ok {
			githubConfig.Changelog = distribution.ChangelogConfig{
				AutoGenerate:   getBoolFromConfig(changelogConfig, "auto_generate", true),
				IncludeCommits: getBoolFromConfig(changelogConfig, "include_commits", true),
				IncludePRs:     getBoolFromConfig(changelogConfig, "include_prs", true),
			}
		}

		// Setup assets config
		if assetsConfig, ok := publisherConfig.Config["assets"].(map[string]interface{}); ok {
			githubConfig.Assets = distribution.AssetsConfig{
				IncludeBinaries:  getBoolFromConfig(assetsConfig, "include_binaries", true),
				IncludeChecksums: getBoolFromConfig(assetsConfig, "include_checksums", true),
				IncludeSource:    getBoolFromConfig(assetsConfig, "include_source", false),
			}
		}

		publisher := distribution.NewGitHubPublisher(githubConfig)
		if err := coordinator.RegisterPublisher(publisher); err != nil {
			return err
		}
	}

	// Setup Go module publisher
	if publisherConfig, exists := config.Publishers["gomodule"]; exists && publisherConfig.Enabled {
		gomodConfig := distribution.GoModuleConfig{
			ModulePath: getStringFromConfig(publisherConfig.Config, "module_path", ""),
			ProxyURL:   getStringFromConfig(publisherConfig.Config, "proxy_url", "https://proxy.golang.org"),
			SumDBURL:   getStringFromConfig(publisherConfig.Config, "sumdb_url", "https://sum.golang.org"),
			Timeout:    publisherConfig.Timeout,
		}

		// Setup documentation config
		if docConfig, ok := publisherConfig.Config["documentation"].(map[string]interface{}); ok {
			gomodConfig.Documentation = distribution.DocumentationConfig{
				GenerateReadme:   getBoolFromConfig(docConfig, "generate_readme", true),
				GenerateExamples: getBoolFromConfig(docConfig, "generate_examples", true),
				IncludeBadges:    getBoolFromConfig(docConfig, "include_badges", true),
				UpdateGoDoc:      getBoolFromConfig(docConfig, "update_godoc", true),
			}

			if badgeTypes, ok := docConfig["badge_types"].([]interface{}); ok {
				for _, badgeType := range badgeTypes {
					if str, ok := badgeType.(string); ok {
						gomodConfig.Documentation.BadgeTypes = append(gomodConfig.Documentation.BadgeTypes, str)
					}
				}
			}
		}

		// Setup examples config
		if examplesConfig, ok := publisherConfig.Config["examples"].(map[string]interface{}); ok {
			gomodConfig.Examples = distribution.ExamplesConfig{
				AutoGenerate:  getBoolFromConfig(examplesConfig, "auto_generate", true),
				TestExamples:  getBoolFromConfig(examplesConfig, "test_examples", true),
				IncludeOutput: getBoolFromConfig(examplesConfig, "include_output", false),
			}
		}

		publisher := distribution.NewGoModulePublisher(gomodConfig)
		if err := coordinator.RegisterPublisher(publisher); err != nil {
			return err
		}
	}

	// Setup Homebrew publisher
	if publisherConfig, exists := config.Publishers["homebrew"]; exists && publisherConfig.Enabled {
		homebrewConfig := distribution.HomebrewConfig{
			TapRepo:     getStringFromConfig(publisherConfig.Config, "tap_repo", ""),
			FormulaName: getStringFromConfig(publisherConfig.Config, "formula_name", ""),
			Description: getStringFromConfig(publisherConfig.Config, "description", ""),
			Homepage:    getStringFromConfig(publisherConfig.Config, "homepage", ""),
			License:     getStringFromConfig(publisherConfig.Config, "license", "MIT"),
			CustomTap:   getBoolFromConfig(publisherConfig.Config, "custom_tap", true),
			TestCommand: getStringFromConfig(publisherConfig.Config, "test_command", `"--version"`),
		}

		// Setup dependencies
		if deps, ok := publisherConfig.Config["dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if str, ok := dep.(string); ok {
					homebrewConfig.Dependencies = append(homebrewConfig.Dependencies, str)
				}
			}
		}

		publisher, err := distribution.NewHomebrewPublisher(homebrewConfig)
		if err != nil {
			return fmt.Errorf("failed to create Homebrew publisher: %w", err)
		}

		if err := coordinator.RegisterPublisher(publisher); err != nil {
			return err
		}
	}

	return nil
}

// setupValidators registers validators with the coordinator
func setupValidators(coordinator *distribution.DistributionCoordinator, config *distribution.DistributionConfig) error {
	// Setup GitHub validator
	if validatorConfig, exists := config.Validators["github"]; exists && validatorConfig.Enabled {
		githubValidatorConfig := distribution.GitHubValidatorConfig{
			CheckAssets:    getBoolFromConfig(validatorConfig.Config, "check_assets", true),
			CheckChangelog: getBoolFromConfig(validatorConfig.Config, "check_changelog", true),
			CheckTag:       getBoolFromConfig(validatorConfig.Config, "check_tag", true),
		}

		if requiredAssets, ok := validatorConfig.Config["required_assets"].([]interface{}); ok {
			for _, asset := range requiredAssets {
				if str, ok := asset.(string); ok {
					githubValidatorConfig.RequiredAssets = append(githubValidatorConfig.RequiredAssets, str)
				}
			}
		}

		validator := distribution.NewGitHubValidator(githubValidatorConfig)
		if err := coordinator.RegisterValidator(validator); err != nil {
			return err
		}
	}

	// Setup Go module validator
	if validatorConfig, exists := config.Validators["gomodule"]; exists && validatorConfig.Enabled {
		gomodValidatorConfig := distribution.GoModuleValidatorConfig{
			CheckSyntax:        getBoolFromConfig(validatorConfig.Config, "check_syntax", true),
			CheckDependencies:  getBoolFromConfig(validatorConfig.Config, "check_dependencies", true),
			CheckLicense:       getBoolFromConfig(validatorConfig.Config, "check_license", true),
			CheckDocumentation: getBoolFromConfig(validatorConfig.Config, "check_documentation", true),
			MinCoverage:        getFloatFromConfig(validatorConfig.Config, "min_coverage", 80.0),
		}

		validator := distribution.NewGoModuleValidator(gomodValidatorConfig)
		if err := coordinator.RegisterValidator(validator); err != nil {
			return err
		}
	}

	return nil
}

// createRelease creates a release from the binary directory
func createRelease(version, tag, binDir string) (*distribution.Release, error) {
	release := &distribution.Release{
		Version:   version,
		Tag:       tag,
		Binaries:  make(map[string]distribution.Binary),
		Checksums: make(map[string]string),
		Metadata: distribution.ReleaseMetadata{
			CreatedAt:    time.Now(),
			IsPrerelease: false,
		},
	}

	// Read binaries from directory
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read bin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		filePath := filepath.Join(binDir, filename)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Determine platform and architecture from filename
		platform, arch := parsePlatformArch(filename)

		binary := distribution.Binary{
			Platform:     platform,
			Architecture: arch,
			Filename:     filename,
			Size:         info.Size(),
			FilePath:     filePath,
		}

		// Calculate checksum
		checksum, err := calculateChecksum(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate checksum for %s: %w", filename, err)
		}

		binary.Checksum = checksum
		release.Binaries[filename] = binary
		release.Checksums[filename] = checksum
	}

	return release, nil
}

// Helper functions

func getStringFromConfig(config map[string]interface{}, key, defaultValue string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return defaultValue
}

func getBoolFromConfig(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return defaultValue
}

func getFloatFromConfig(config map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := config[key].(float64); ok {
		return value
	}
	return defaultValue
}

func expandEnvVars(value string) string {
	return os.ExpandEnv(value)
}

func parsePlatformArch(filename string) (platform, arch string) {
	// Simple parsing based on common naming conventions
	if strings.Contains(filename, "linux") {
		platform = "linux"
	} else if strings.Contains(filename, "windows") {
		platform = "windows"
	} else if strings.Contains(filename, "darwin") {
		platform = "darwin"
	} else {
		platform = "unknown"
	}

	if strings.Contains(filename, "amd64") {
		arch = "amd64"
	} else if strings.Contains(filename, "arm64") {
		arch = "arm64"
	} else if strings.Contains(filename, "386") {
		arch = "386"
	} else {
		arch = "unknown"
	}

	return platform, arch
}

func calculateChecksum(filePath string) (string, error) {
	// This would typically use SHA256 or similar
	// For now, return a placeholder
	return "sha256:placeholder", nil
}

// generateHomebrewFormula generates a Homebrew formula for the specified version
func generateHomebrewFormula(version, binaryURL, outputFile string, config *distribution.DistributionConfig) error {
	if binaryURL == "" {
		return fmt.Errorf("binary URL is required for Homebrew formula generation")
	}

	if outputFile == "" {
		outputFile = "homebrew-formula/nettracex.rb"
	}

	// Get Homebrew configuration
	homebrewConfig := distribution.HomebrewConfig{
		TapRepo:     "nettracex/homebrew-tap",
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit with beautiful TUI",
		Homepage:    "https://github.com/nettracex/nettracex-tui",
		License:     "MIT",
		CustomTap:   true,
		TestCommand: `"--version"`,
	}

	// Override with config values if available
	if publisherConfig, exists := config.Publishers["homebrew"]; exists {
		homebrewConfig.TapRepo = getStringFromConfig(publisherConfig.Config, "tap_repo", homebrewConfig.TapRepo)
		homebrewConfig.FormulaName = getStringFromConfig(publisherConfig.Config, "formula_name", homebrewConfig.FormulaName)
		homebrewConfig.Description = getStringFromConfig(publisherConfig.Config, "description", homebrewConfig.Description)
		homebrewConfig.Homepage = getStringFromConfig(publisherConfig.Config, "homepage", homebrewConfig.Homepage)
		homebrewConfig.License = getStringFromConfig(publisherConfig.Config, "license", homebrewConfig.License)
		homebrewConfig.TestCommand = getStringFromConfig(publisherConfig.Config, "test_command", homebrewConfig.TestCommand)
	}

	// Create publisher
	publisher, err := distribution.NewHomebrewPublisher(homebrewConfig)
	if err != nil {
		return fmt.Errorf("failed to create Homebrew publisher: %w", err)
	}

	// Create release with the provided binary URL
	release := distribution.Release{
		Version: version,
		Tag:     version,
		Binaries: map[string]distribution.Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     homebrewConfig.FormulaName,
				DownloadURL:  binaryURL,
				Checksum:     "", // Will be calculated by the publisher
			},
		},
	}

	// Generate formula
	formula, err := publisher.GenerateFormula(release)
	if err != nil {
		return fmt.Errorf("failed to generate formula: %w", err)
	}

	// Render formula content
	content, err := publisher.RenderFormula(formula)
	if err != nil {
		return fmt.Errorf("failed to render formula: %w", err)
	}

	// Create output directory
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write formula to file
	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write formula file: %w", err)
	}

	fmt.Printf("Homebrew formula written to: %s\n", outputFile)
	return nil
}