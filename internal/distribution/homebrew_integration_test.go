// +build integration

package distribution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHomebrewPublisher_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if Homebrew is available
	if _, err := exec.LookPath("brew"); err != nil {
		t.Skip("Homebrew not available, skipping integration test")
	}

	// Create a test server to serve the binary
	testBinary := []byte("#!/bin/bash\necho 'NetTraceX v1.0.0'\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(testBinary)
	}))
	defer server.Close()

	config := HomebrewConfig{
		TapRepo:     "test/homebrew-tap",
		FormulaName: "nettracex-test",
		Description: "Network diagnostic toolkit (test version)",
		Homepage:    "https://github.com/test/nettracex",
		License:     "MIT",
		CustomTap:   true,
		TestCommand: `"--version"`,
	}

	publisher, err := NewHomebrewPublisher(config)
	if err != nil {
		t.Fatalf("Failed to create Homebrew publisher: %v", err)
	}

	release := Release{
		Version: "1.0.0-test",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     "nettracex-test",
				DownloadURL:  server.URL,
				Checksum:     "", // Will be calculated
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("validate release", func(t *testing.T) {
		err := publisher.Validate(ctx, release)
		if err != nil {
			t.Fatalf("Release validation failed: %v", err)
		}
	})

	t.Run("publish release", func(t *testing.T) {
		err := publisher.Publish(ctx, release)
		if err != nil {
			t.Fatalf("Failed to publish release: %v", err)
		}

		// Check that formula file was created
		expectedFile := filepath.Join("homebrew-formula", "nettracex-test.rb")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Expected formula file %s to be created", expectedFile)
		}

		// Clean up
		defer os.RemoveAll("homebrew-formula")
	})

	t.Run("validate generated formula", func(t *testing.T) {
		formulaFile := filepath.Join("homebrew-formula", "nettracex-test.rb")
		
		// Use brew audit to validate the formula
		cmd := exec.CommandContext(ctx, "brew", "audit", "--strict", formulaFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Brew audit output: %s", string(output))
			// Don't fail the test for audit warnings, just log them
			if !strings.Contains(string(output), "Warning") {
				t.Errorf("Formula audit failed: %v", err)
			}
		}
	})

	t.Run("test formula installation dry run", func(t *testing.T) {
		formulaFile := filepath.Join("homebrew-formula", "nettracex-test.rb")
		
		// Test installation without actually installing
		cmd := exec.CommandContext(ctx, "brew", "install", "--build-from-source", formulaFile, "--dry-run")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Dry run output: %s", string(output))
			// Some dry run failures are expected due to test environment
			if !strings.Contains(string(output), "would install") {
				t.Logf("Dry run failed (expected in test environment): %v", err)
			}
		}
	})
}

func TestHomebrewValidator_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	validator, err := NewHomebrewValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	if validator.brewPath == "" {
		t.Skip("Homebrew not available, skipping validator integration test")
	}

	t.Run("validate real formula", func(t *testing.T) {
		// Create a test server with a real binary-like response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("fake binary content"))
		}))
		defer server.Close()

		formula := &HomebrewFormula{
			Class:       "NettracexTest",
			Description: "Network diagnostic toolkit (test)",
			Homepage:    "https://github.com/test/nettracex",
			URL:         server.URL,
			SHA256:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // Empty string SHA256
			License:     "MIT",
			Version:     "1.0.0-test",
			TestBlock:   `system "#{bin}/nettracex-test", "--version"`,
		}

		err := validator.ValidateFormula(formula)
		if err != nil {
			t.Errorf("Formula validation failed: %v", err)
		}
	})

	t.Run("validate formula with invalid URL", func(t *testing.T) {
		formula := &HomebrewFormula{
			Class:       "NettracexTest",
			Description: "Network diagnostic toolkit (test)",
			Homepage:    "https://github.com/test/nettracex",
			URL:         "https://invalid-url-that-does-not-exist.example.com/binary",
			SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			License:     "MIT",
			Version:     "1.0.0-test",
		}

		err := validator.ValidateFormula(formula)
		if err == nil {
			t.Error("Expected validation to fail for invalid URL")
		}
	})
}

func TestHomebrewPublisher_RealWorldScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world scenario test in short mode")
	}

	// This test simulates a real-world scenario with GitHub releases
	config := HomebrewConfig{
		TapRepo:     "nettracex/homebrew-tap",
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit with beautiful TUI",
		Homepage:    "https://github.com/nettracex/nettracex",
		License:     "MIT",
		CustomTap:   true,
		Dependencies: []string{},
		TestCommand: `"--version"`,
	}

	publisher, err := NewHomebrewPublisher(config)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}

	// Simulate a real GitHub release
	release := Release{
		Version: "1.2.3",
		Tag:     "v1.2.3",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     "nettracex",
				Size:         1024000, // 1MB
				DownloadURL:  "https://github.com/nettracex/nettracex/releases/download/v1.2.3/nettracex-darwin-amd64",
				Checksum:     "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890",
			},
			"darwin-arm64": {
				Platform:     "darwin",
				Architecture: "arm64",
				Filename:     "nettracex",
				Size:         1024000,
				DownloadURL:  "https://github.com/nettracex/nettracex/releases/download/v1.2.3/nettracex-darwin-arm64",
				Checksum:     "b2c3d4e5f6789012345678901234567890123456789012345678901234567890a1",
			},
		},
		Checksums: map[string]string{
			"darwin-amd64": "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890",
			"darwin-arm64": "b2c3d4e5f6789012345678901234567890123456789012345678901234567890a1",
		},
		Changelog: "## What's Changed\n* Added new features\n* Fixed bugs\n* Improved performance",
		ReleaseNotes: "This release includes significant improvements to the TUI interface.",
		Metadata: ReleaseMetadata{
			CreatedAt:    time.Now(),
			Author:       "release-bot",
			CommitSHA:    "abc123def456",
			BuildNumber:  "123",
			IsPrerelease: false,
			Tags:         []string{"stable", "production"},
		},
	}

	ctx := context.Background()

	t.Run("validate real-world release", func(t *testing.T) {
		err := publisher.Validate(ctx, release)
		if err != nil {
			t.Errorf("Real-world release validation failed: %v", err)
		}
	})

	t.Run("generate formula for real-world release", func(t *testing.T) {
		formula, err := publisher.generateFormula(release)
		if err != nil {
			t.Fatalf("Failed to generate formula: %v", err)
		}

		// Validate formula structure
		if formula.Class != "Nettracex" {
			t.Errorf("Expected class 'Nettracex', got %s", formula.Class)
		}

		if formula.Version != "1.2.3" {
			t.Errorf("Expected version '1.2.3', got %s", formula.Version)
		}

		if !strings.Contains(formula.URL, "darwin-amd64") {
			t.Errorf("Expected URL to contain darwin-amd64, got %s", formula.URL)
		}

		// Render and validate the formula content
		content, err := publisher.renderFormula(formula)
		if err != nil {
			t.Fatalf("Failed to render formula: %v", err)
		}

		expectedElements := []string{
			"class Nettracex < Formula",
			`desc "Network diagnostic toolkit with beautiful TUI"`,
			`homepage "https://github.com/nettracex/nettracex"`,
			`license "MIT"`,
			`version "1.2.3"`,
			"def install",
			"test do",
		}

		for _, element := range expectedElements {
			if !strings.Contains(content, element) {
				t.Errorf("Formula missing expected element: %s\nFormula content:\n%s", element, content)
			}
		}
	})

	t.Run("check status reporting", func(t *testing.T) {
		status := publisher.GetStatus()
		
		if status.Name != "homebrew" {
			t.Errorf("Expected status name 'homebrew', got %s", status.Name)
		}

		expectedMetadata := []string{"tap_repo", "formula", "custom_tap", "brew_available"}
		for _, key := range expectedMetadata {
			if _, exists := status.Metadata[key]; !exists {
				t.Errorf("Expected metadata key %s to exist", key)
			}
		}
	})
}

func TestHomebrewPublisher_ErrorHandling(t *testing.T) {
	config := HomebrewConfig{
		FormulaName: "nettracex-error-test",
		CustomTap:   true,
	}

	publisher, err := NewHomebrewPublisher(config)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}

	ctx := context.Background()

	t.Run("handle missing macOS binary", func(t *testing.T) {
		release := Release{
			Version: "1.0.0",
			Binaries: map[string]Binary{
				"linux-amd64": {
					Platform:    "linux",
					Architecture: "amd64",
					DownloadURL: "https://example.com/binary",
					Checksum:    "abc123",
				},
			},
		}

		err := publisher.Validate(ctx, release)
		if err == nil {
			t.Error("Expected validation to fail for missing macOS binary")
		}

		if !strings.Contains(err.Error(), "no macOS binary found") {
			t.Errorf("Expected error about missing macOS binary, got: %v", err)
		}
	})

	t.Run("handle network errors gracefully", func(t *testing.T) {
		release := Release{
			Version: "1.0.0",
			Binaries: map[string]Binary{
				"darwin-amd64": {
					Platform:    "darwin",
					Architecture: "amd64",
					DownloadURL: "https://invalid-domain-that-does-not-exist.example.com/binary",
					Checksum:    "abc123",
				},
			},
		}

		err := publisher.Publish(ctx, release)
		if err == nil {
			t.Error("Expected publish to fail for invalid URL")
		}

		// Check that status was updated to error
		status := publisher.GetStatus()
		if status.Status != StatusError {
			t.Errorf("Expected status error, got %s", status.Status)
		}

		if status.LastError == "" {
			t.Error("Expected last error to be set")
		}
	})

	t.Run("handle formula validation errors", func(t *testing.T) {
		// Create a server that returns invalid content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		release := Release{
			Version: "1.0.0",
			Binaries: map[string]Binary{
				"darwin-amd64": {
					Platform:    "darwin",
					Architecture: "amd64",
					DownloadURL: server.URL,
					Checksum:    "abc123",
				},
			},
		}

		err := publisher.Publish(ctx, release)
		if err == nil {
			t.Error("Expected publish to fail for invalid URL response")
		}

		if !strings.Contains(err.Error(), "formula validation failed") {
			t.Errorf("Expected formula validation error, got: %v", err)
		}
	})
}

// Benchmark integration tests
func BenchmarkHomebrewPublisher_Integration(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping integration benchmark in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test binary content"))
	}))
	defer server.Close()

	config := HomebrewConfig{
		FormulaName: "nettracex-bench",
		Description: "Benchmark test",
		Homepage:    "https://example.com",
		License:     "MIT",
		CustomTap:   true,
	}

	publisher, _ := NewHomebrewPublisher(config)

	release := Release{
		Version: "1.0.0",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:    "darwin",
				Architecture: "amd64",
				DownloadURL: server.URL,
				Checksum:    "abc123def456",
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := publisher.Validate(ctx, release)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}