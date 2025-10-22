package distribution

import (
	"context"
	"fmt"
	"log"
)

// ExampleHomebrewPublisher demonstrates how to use the Homebrew publisher
func ExampleHomebrewPublisher() {
	// Configure Homebrew publisher
	config := HomebrewConfig{
		TapRepo:     "myorg/homebrew-tap",
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit with beautiful TUI",
		Homepage:    "https://github.com/myorg/nettracex",
		License:     "MIT",
		CustomTap:   true,
		TestCommand: `"--version"`,
		Dependencies: []string{}, // No dependencies required
	}

	// Create publisher
	publisher, err := NewHomebrewPublisher(config)
	if err != nil {
		log.Fatalf("Failed to create Homebrew publisher: %v", err)
	}

	// Create a sample release
	release := Release{
		Version: "1.0.0",
		Tag:     "v1.0.0",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     "nettracex",
				DownloadURL:  "https://httpbin.org/bytes/1024",
				Checksum:     "", // Will be calculated
				Size:         1024000,
			},
		},
		Changelog:    "Initial release with WHOIS, ping, and traceroute functionality",
		ReleaseNotes: "First stable release of NetTraceX",
	}

	ctx := context.Background()

	// Validate the release
	if err := publisher.Validate(ctx, release); err != nil {
		log.Fatalf("Release validation failed: %v", err)
	}

	// Publish to Homebrew
	if err := publisher.Publish(ctx, release); err != nil {
		log.Fatalf("Failed to publish to Homebrew: %v", err)
	}

	// Check status
	status := publisher.GetStatus()
	fmt.Printf("Publisher: %s, Status: %s\n", status.Name, status.Status)

	// Output: Publisher: homebrew, Status: success
}

// Example_homebrewDistributionCoordinator demonstrates using Homebrew with the distribution coordinator
func Example_homebrewDistributionCoordinator() {
	// Create distribution configuration
	config := &DistributionConfig{
		Publishers: map[string]PublisherConfig{
			"homebrew": {
				Enabled:    true,
				Priority:   2,
				Timeout:    300000000000, // 5 minutes
				RetryCount: 3,
				Config: map[string]interface{}{
					"tap_repo":     "myorg/homebrew-tap",
					"formula_name": "nettracex",
					"description":  "Network diagnostic toolkit",
					"homepage":     "https://github.com/myorg/nettracex",
					"license":      "MIT",
					"custom_tap":   true,
				},
			},
			"github": {
				Enabled:    true,
				Priority:   1,
				Timeout:    180000000000, // 3 minutes
				RetryCount: 2,
			},
		},
		ConcurrentLimit: 2,
		RetryPolicy: RetryPolicy{
			MaxRetries: 3,
			BaseDelay:  1000000000, // 1 second
			MaxDelay:   30000000000, // 30 seconds
			Multiplier: 2.0,
		},
	}

	// Create coordinator
	coordinator := NewDistributionCoordinator(config)

	// Create and register Homebrew publisher
	homebrewConfig := HomebrewConfig{
		TapRepo:     "myorg/homebrew-tap",
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/myorg/nettracex",
		License:     "MIT",
		CustomTap:   true,
	}

	homebrewPublisher, err := NewHomebrewPublisher(homebrewConfig)
	if err != nil {
		log.Fatalf("Failed to create Homebrew publisher: %v", err)
	}

	if err := coordinator.RegisterPublisher(homebrewPublisher); err != nil {
		log.Fatalf("Failed to register Homebrew publisher: %v", err)
	}

	// Create sample release
	release := Release{
		Version: "1.1.0",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:    "darwin",
				Architecture: "amd64",
				DownloadURL: "https://github.com/myorg/nettracex/releases/download/v1.1.0/nettracex-darwin-amd64",
				Checksum:    "b2c3d4e5f6789012345678901234567890123456789012345678901234567890a1",
			},
		},
	}

	// Distribute to all publishers (including Homebrew)
	ctx := context.Background()
	if err := coordinator.Distribute(ctx, release); err != nil {
		log.Fatalf("Distribution failed: %v", err)
	}

	// Check publisher statuses
	statuses := coordinator.GetPublisherStatus()
	for name, status := range statuses {
		fmt.Printf("Publisher %s: %s\n", name, status.Status)
	}

	// Output: Publisher homebrew: success
}

// ExampleHomebrewFormula demonstrates formula generation
func ExampleHomebrewFormula() {
	config := HomebrewConfig{
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/myorg/nettracex",
		License:     "MIT",
		TestCommand: `"--help"`,
	}

	publisher, _ := NewHomebrewPublisher(config)

	release := Release{
		Version: "2.0.0",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     "nettracex",
				DownloadURL:  "https://github.com/myorg/nettracex/releases/download/v2.0.0/nettracex-darwin-amd64.tar.gz",
				Checksum:     "c3d4e5f6789012345678901234567890123456789012345678901234567890a1b2",
			},
		},
	}

	// Generate formula
	formula, err := publisher.GenerateFormula(release)
	if err != nil {
		log.Fatalf("Failed to generate formula: %v", err)
	}

	// Render formula to Ruby code
	content, err := publisher.RenderFormula(formula)
	if err != nil {
		log.Fatalf("Failed to render formula: %v", err)
	}

	fmt.Printf("Generated formula:\n%s\n", content)

	// Output will be a complete Homebrew formula in Ruby format
}

// ExampleHomebrewValidator demonstrates formula validation
func ExampleHomebrewValidator() {
	validator, err := NewHomebrewValidator()
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}

	// Create a sample formula
	formula := &HomebrewFormula{
		Class:       "Nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/myorg/nettracex",
		URL:         "https://github.com/myorg/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64",
		SHA256:      "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890",
		License:     "MIT",
		Version:     "1.0.0",
		TestBlock:   `system "#{bin}/nettracex", "--version"`,
	}

	// Validate the formula
	if err := validator.ValidateFormula(formula); err != nil {
		log.Fatalf("Formula validation failed: %v", err)
	}

	fmt.Println("Formula validation passed")

	// Output: Formula validation passed
}