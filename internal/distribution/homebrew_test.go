package distribution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewHomebrewPublisher(t *testing.T) {
	config := HomebrewConfig{
		TapRepo:     "test/homebrew-tap",
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/test/nettracex",
		License:     "MIT",
		CustomTap:   true,
	}

	publisher, err := NewHomebrewPublisher(config)
	if err != nil {
		t.Fatalf("Failed to create Homebrew publisher: %v", err)
	}

	if publisher.config.TapRepo != config.TapRepo {
		t.Errorf("Expected tap repo %s, got %s", config.TapRepo, publisher.config.TapRepo)
	}

	if publisher.validator == nil {
		t.Error("Expected validator to be initialized")
	}
}

func TestHomebrewPublisher_Validate(t *testing.T) {
	publisher, _ := NewHomebrewPublisher(HomebrewConfig{})
	ctx := context.Background()

	tests := []struct {
		name    string
		release Release
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid release",
			release: Release{
				Version: "1.0.0",
				Binaries: map[string]Binary{
					"darwin-amd64": {
						Platform:    "darwin",
						Architecture: "amd64",
						DownloadURL: "https://example.com/binary",
						Checksum:    "abc123",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "linux binary only (should work)",
			release: Release{
				Version: "1.0.0",
				Binaries: map[string]Binary{
					"linux-amd64": {
						Platform:    "linux",
						Architecture: "amd64",
						DownloadURL: "https://example.com/binary",
						Checksum:    "abc123",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no supported binary",
			release: Release{
				Version: "1.0.0",
				Binaries: map[string]Binary{
					"windows-amd64": {
						Platform:    "windows",
						Architecture: "amd64",
						DownloadURL: "https://example.com/binary",
						Checksum:    "abc123",
					},
				},
			},
			wantErr: true,
			errMsg:  "no supported binary found",
		},
		{
			name: "missing download URL",
			release: Release{
				Version: "1.0.0",
				Binaries: map[string]Binary{
					"darwin-amd64": {
						Platform:     "darwin",
						Architecture: "amd64",
						Checksum:     "abc123",
					},
				},
			},
			wantErr: true,
			errMsg:  "missing download URL",
		},
		{
			name: "missing checksum",
			release: Release{
				Version: "1.0.0",
				Binaries: map[string]Binary{
					"darwin-amd64": {
						Platform:    "darwin",
						Architecture: "amd64",
						DownloadURL: "https://example.com/binary",
					},
				},
			},
			wantErr: true,
			errMsg:  "missing checksum",
		},
		{
			name: "missing version",
			release: Release{
				Binaries: map[string]Binary{
					"darwin-amd64": {
						Platform:    "darwin",
						Architecture: "amd64",
						DownloadURL: "https://example.com/binary",
						Checksum:    "abc123",
					},
				},
			},
			wantErr: true,
			errMsg:  "version is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publisher.Validate(ctx, tt.release)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHomebrewPublisher_GenerateFormula(t *testing.T) {
	config := HomebrewConfig{
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/test/nettracex",
		License:     "MIT",
		TestCommand: "--version",
	}

	publisher, _ := NewHomebrewPublisher(config)

	t.Run("single platform (macOS)", func(t *testing.T) {
		release := Release{
			Version: "1.0.0",
			Binaries: map[string]Binary{
				"darwin-amd64": {
					Platform:     "darwin",
					Architecture: "amd64",
					Filename:     "nettracex",
					DownloadURL:  "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64",
					Checksum:     "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				},
			},
		}

		formula, err := publisher.GenerateFormula(release)
		if err != nil {
			t.Fatalf("Failed to generate formula: %v", err)
		}

		if formula.Class != "Nettracex" {
			t.Errorf("Expected class name 'Nettracex', got %s", formula.Class)
		}

		if formula.Description != config.Description {
			t.Errorf("Expected description %s, got %s", config.Description, formula.Description)
		}

		expectedURL := release.Binaries["darwin-amd64"].DownloadURL
		if formula.URL != expectedURL {
			t.Errorf("Expected URL %s, got %s", expectedURL, formula.URL)
		}

		expectedSHA := release.Binaries["darwin-amd64"].Checksum
		if formula.SHA256 != expectedSHA {
			t.Errorf("Expected SHA256 %s, got %s", expectedSHA, formula.SHA256)
		}
	})

	t.Run("multi-platform (macOS and Linux)", func(t *testing.T) {
		release := Release{
			Version: "1.0.0",
			Binaries: map[string]Binary{
				"darwin-amd64": {
					Platform:     "darwin",
					Architecture: "amd64",
					Filename:     "nettracex",
					DownloadURL:  "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64",
					Checksum:     "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				},
				"linux-amd64": {
					Platform:     "linux",
					Architecture: "amd64",
					Filename:     "nettracex",
					DownloadURL:  "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-linux-amd64",
					Checksum:     "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				},
			},
		}

		formula, err := publisher.GenerateFormula(release)
		if err != nil {
			t.Fatalf("Failed to generate formula: %v", err)
		}

		// Should prefer macOS as primary
		expectedURL := release.Binaries["darwin-amd64"].DownloadURL
		if formula.URL != expectedURL {
			t.Errorf("Expected primary URL %s, got %s", expectedURL, formula.URL)
		}

		// Should have platform URLs for both platforms
		if len(formula.PlatformURLs) != 2 {
			t.Errorf("Expected 2 platform URLs, got %d", len(formula.PlatformURLs))
		}

		if _, exists := formula.PlatformURLs["darwin-amd64"]; !exists {
			t.Error("Expected darwin-amd64 platform URL")
		}

		if _, exists := formula.PlatformURLs["linux-amd64"]; !exists {
			t.Error("Expected linux-amd64 platform URL")
		}
	})

	t.Run("Linux only", func(t *testing.T) {
		release := Release{
			Version: "1.0.0",
			Binaries: map[string]Binary{
				"linux-amd64": {
					Platform:     "linux",
					Architecture: "amd64",
					Filename:     "nettracex",
					DownloadURL:  "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-linux-amd64",
					Checksum:     "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				},
			},
		}

		formula, err := publisher.GenerateFormula(release)
		if err != nil {
			t.Fatalf("Failed to generate formula: %v", err)
		}

		// Should use Linux as primary since no macOS available
		expectedURL := release.Binaries["linux-amd64"].DownloadURL
		if formula.URL != expectedURL {
			t.Errorf("Expected primary URL %s, got %s", expectedURL, formula.URL)
		}

		if formula.Metadata["platform"] != "linux" {
			t.Errorf("Expected platform 'linux', got %s", formula.Metadata["platform"])
		}
	})
}

func TestHomebrewPublisher_RenderFormula(t *testing.T) {
	config := HomebrewConfig{
		FormulaName: "nettracex",
	}

	publisher, _ := NewHomebrewPublisher(config)

	formula := &HomebrewFormula{
		Class:       "Nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/test/nettracex",
		URL:         "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64",
		SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		License:     "MIT",
		Version:     "1.0.0",
		TestBlock:   `system "#{bin}/nettracex", "--version"`,
		Metadata: map[string]string{
			"binary_name": "nettracex",
		},
	}

	content, err := publisher.RenderFormula(formula)
	if err != nil {
		t.Fatalf("Failed to render formula: %v", err)
	}

	// Check that the rendered content contains expected elements
	expectedElements := []string{
		"class Nettracex < Formula",
		`desc "Network diagnostic toolkit"`,
		`homepage "https://github.com/test/nettracex"`,
		`url "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64"`,
		`sha256 "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"`,
		`license "MIT"`,
		`version "1.0.0"`,
		`bin.install "nettracex" => "nettracex"`,
		`system "#{bin}/nettracex", "--version"`,
	}

	for _, element := range expectedElements {
		if !strings.Contains(content, element) {
			t.Errorf("Expected formula to contain %q, but it didn't.\nFormula content:\n%s", element, content)
		}
	}
}

func TestHomebrewValidator_ValidateFormula(t *testing.T) {
	validator, _ := NewHomebrewValidator()

	// Create a test server to simulate URL accessibility
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/valid" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		formula HomebrewFormula
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid formula",
			formula: HomebrewFormula{
				Class:       "Nettracex",
				Description: "Network diagnostic toolkit",
				Homepage:    "https://github.com/test/nettracex",
				URL:         server.URL + "/valid",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				License:     "MIT",
			},
			wantErr: false,
		},
		{
			name: "missing class",
			formula: HomebrewFormula{
				Description: "Network diagnostic toolkit",
				Homepage:    "https://github.com/test/nettracex",
				URL:         server.URL + "/valid",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				License:     "MIT",
			},
			wantErr: true,
			errMsg:  "class name is required",
		},
		{
			name: "missing description",
			formula: HomebrewFormula{
				Class:    "Nettracex",
				Homepage: "https://github.com/test/nettracex",
				URL:      server.URL + "/valid",
				SHA256:   "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				License:  "MIT",
			},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name: "invalid URL",
			formula: HomebrewFormula{
				Class:       "Nettracex",
				Description: "Network diagnostic toolkit",
				Homepage:    "https://github.com/test/nettracex",
				URL:         server.URL + "/invalid",
				SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				License:     "MIT",
			},
			wantErr: true,
			errMsg:  "URL returned status",
		},
		{
			name: "invalid SHA256 length",
			formula: HomebrewFormula{
				Class:       "Nettracex",
				Description: "Network diagnostic toolkit",
				Homepage:    "https://github.com/test/nettracex",
				URL:         server.URL + "/valid",
				SHA256:      "short",
				License:     "MIT",
			},
			wantErr: true,
			errMsg:  "invalid SHA256 hash length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFormula(&tt.formula)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHomebrewPublisher_CalculateSHA256(t *testing.T) {
	// Create a test server with known content
	testContent := "test content for SHA256 calculation"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testContent))
	}))
	defer server.Close()

	publisher, _ := NewHomebrewPublisher(HomebrewConfig{})

	hash, err := publisher.calculateSHA256(server.URL)
	if err != nil {
		t.Fatalf("Failed to calculate SHA256: %v", err)
	}

	// Validate SHA256 hash format
	if len(hash) != 64 {
		t.Errorf("Expected SHA256 hash length 64, got %d", len(hash))
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}
}

func TestHomebrewPublisher_Publish(t *testing.T) {
	// Create a test server to simulate binary download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test binary content"))
	}))
	defer server.Close()

	config := HomebrewConfig{
		TapRepo:     "test/homebrew-tap",
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/test/nettracex",
		License:     "MIT",
		CustomTap:   true,
	}

	publisher, _ := NewHomebrewPublisher(config)

	release := Release{
		Version: "1.0.0",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     "nettracex",
				DownloadURL:  server.URL,
				Checksum:     "", // Will be calculated
			},
		},
	}

	ctx := context.Background()
	err := publisher.Publish(ctx, release)
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Check status was updated to success
	status := publisher.GetStatus()
	if status.Status != StatusSuccess {
		t.Errorf("Expected status success, got %s", status.Status)
	}

	// Check that formula file was created
	expectedFile := filepath.Join("homebrew-formula", "nettracex.rb")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected formula file %s to be created", expectedFile)
	} else {
		// Clean up
		os.RemoveAll("homebrew-formula")
	}
}

func TestHomebrewPublisher_GetStatus(t *testing.T) {
	config := HomebrewConfig{
		TapRepo:     "test/homebrew-tap",
		FormulaName: "nettracex",
		CustomTap:   true,
	}

	publisher, _ := NewHomebrewPublisher(config)

	status := publisher.GetStatus()

	if status.Name != "homebrew" {
		t.Errorf("Expected status name 'homebrew', got %s", status.Name)
	}

	if status.Status != StatusIdle {
		t.Errorf("Expected status 'idle', got %s", status.Status)
	}

	expectedMetadata := map[string]string{
		"tap_repo":   "test/homebrew-tap",
		"formula":    "nettracex",
		"custom_tap": "true",
	}

	for key, expectedValue := range expectedMetadata {
		if value, exists := status.Metadata[key]; !exists {
			t.Errorf("Expected metadata key %s to exist", key)
		} else if value != expectedValue {
			t.Errorf("Expected metadata %s=%s, got %s", key, expectedValue, value)
		}
	}
}

func TestHomebrewPublisher_GenerateTestBlock(t *testing.T) {
	tests := []struct {
		name        string
		config      HomebrewConfig
		expected    string
	}{
		{
			name: "custom test command",
			config: HomebrewConfig{
				FormulaName: "nettracex",
				TestCommand: `"--help"`,
			},
			expected: `system "#{bin}/nettracex", "--help"`,
		},
		{
			name: "default test command",
			config: HomebrewConfig{
				FormulaName: "nettracex",
			},
			expected: `system "#{bin}/nettracex", "--version"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher, _ := NewHomebrewPublisher(tt.config)
			result := publisher.generateTestBlock()
			if result != tt.expected {
				t.Errorf("Expected test block %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHomebrewPublisher_SubmitToCustomTap(t *testing.T) {
	config := HomebrewConfig{
		FormulaName: "nettracex",
		CustomTap:   true,
	}

	publisher, _ := NewHomebrewPublisher(config)

	formula := &HomebrewFormula{
		Class: "Nettracex",
	}

	content := "test formula content"

	ctx := context.Background()
	url, err := publisher.submitToCustomTap(ctx, formula, content)
	if err != nil {
		t.Fatalf("Failed to submit to custom tap: %v", err)
	}

	if !strings.Contains(url, "nettracex.rb") {
		t.Errorf("Expected URL to contain formula file path, got %s", url)
	}

	// Check that file was created
	expectedFile := filepath.Join("homebrew-formula", "nettracex.rb")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected formula file %s to be created", expectedFile)
	} else {
		// Verify content
		fileContent, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Errorf("Failed to read formula file: %v", err)
		} else if string(fileContent) != content {
			t.Errorf("Expected file content %q, got %q", content, string(fileContent))
		}
		// Clean up
		os.RemoveAll("homebrew-formula")
	}
}

// Benchmark tests
func BenchmarkHomebrewPublisher_GenerateFormula(b *testing.B) {
	config := HomebrewConfig{
		FormulaName: "nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/test/nettracex",
		License:     "MIT",
	}

	publisher, _ := NewHomebrewPublisher(config)

	release := Release{
		Version: "1.0.0",
		Binaries: map[string]Binary{
			"darwin-amd64": {
				Platform:     "darwin",
				Architecture: "amd64",
				Filename:     "nettracex",
				DownloadURL:  "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64",
				Checksum:     "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := publisher.GenerateFormula(release)
		if err != nil {
			b.Fatalf("Failed to generate formula: %v", err)
		}
	}
}

func BenchmarkHomebrewPublisher_RenderFormula(b *testing.B) {
	publisher, _ := NewHomebrewPublisher(HomebrewConfig{})

	formula := &HomebrewFormula{
		Class:       "Nettracex",
		Description: "Network diagnostic toolkit",
		Homepage:    "https://github.com/test/nettracex",
		URL:         "https://github.com/test/nettracex/releases/download/v1.0.0/nettracex-darwin-amd64",
		SHA256:      "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		License:     "MIT",
		Version:     "1.0.0",
		TestBlock:   `system "#{bin}/nettracex", "--version"`,
		Metadata: map[string]string{
			"binary_name": "nettracex",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := publisher.RenderFormula(formula)
		if err != nil {
			b.Fatalf("Failed to render formula: %v", err)
		}
	}
}