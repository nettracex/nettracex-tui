package distribution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributionIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Create test files
	setupTestEnvironment(t, tempDir)

	// Create mock GitHub API server
	githubServer := createMockGitHubServer(t)
	defer githubServer.Close()

	// Create mock Go proxy server
	proxyServer := createMockProxyServer(t)
	defer proxyServer.Close()

	// Setup distribution coordinator
	config := &DistributionConfig{
		Publishers: map[string]PublisherConfig{
			"github": {
				Enabled:    true,
				Priority:   1,
				Timeout:    10 * time.Second,
				RetryCount: 1,
			},
			"gomodule": {
				Enabled:    true,
				Priority:   2,
				Timeout:    10 * time.Second,
				RetryCount: 1,
			},
		},
		Validators: map[string]ValidatorConfig{
			"github": {Enabled: true},
			"gomodule": {Enabled: true},
		},
		RetryPolicy: RetryPolicy{
			MaxRetries: 1,
			BaseDelay:  time.Millisecond,
			MaxDelay:   time.Second,
			Multiplier: 2.0,
		},
		ConcurrentLimit: 2,
	}

	coordinator := NewDistributionCoordinator(config)

	// Setup notification service
	notificationConfig := NotificationServiceConfig{
		Enabled: true,
		RetryPolicy: NotificationRetryPolicy{
			MaxRetries: 1,
			BaseDelay:  time.Millisecond,
		},
	}
	notifier := NewDefaultNotificationService(notificationConfig)
	coordinator.SetNotificationService(notifier)

	// Register publishers
	githubPublisher := NewGitHubPublisher(GitHubConfig{
		Owner:   "test-owner",
		Repo:    "test-repo",
		Token:   "test-token",
		BaseURL: githubServer.URL,
		Timeout: 5 * time.Second,
		Assets: AssetsConfig{
			IncludeBinaries:  true,
			IncludeChecksums: true,
		},
	})

	gomodPublisher := NewGoModulePublisher(GoModuleConfig{
		ModulePath: "github.com/test/module",
		ProxyURL:   proxyServer.URL,
		Timeout:    5 * time.Second,
		Documentation: DocumentationConfig{
			GenerateReadme: false, // Skip for integration test
		},
		Examples: ExamplesConfig{
			AutoGenerate: false, // Skip for integration test
		},
	})

	// Override methods that require external dependencies
	gomodPublisher.createGitTag = func(ctx context.Context, release Release) error {
		return nil // Skip git operations in test
	}
	gomodPublisher.verifyModuleAvailability = func(ctx context.Context, release Release) error {
		return nil // Skip verification in test
	}

	require.NoError(t, coordinator.RegisterPublisher(githubPublisher))
	require.NoError(t, coordinator.RegisterPublisher(gomodPublisher))

	// Register validators
	githubValidator := NewGitHubValidator(GitHubValidatorConfig{
		CheckAssets:    true,
		CheckChangelog: false, // Skip changelog check
		CheckTag:       true,
	})

	gomodValidator := NewGoModuleValidator(GoModuleValidatorConfig{
		CheckSyntax:        false, // Skip syntax check in test
		CheckDependencies:  false, // Skip dependency check in test
		CheckLicense:       false, // Skip license check in test
		CheckDocumentation: false, // Skip doc check in test
	})
	
	// Override validator to always pass
	gomodValidator.ValidateRelease = func(ctx context.Context, release Release) (*PackageValidationResult, error) {
		return &PackageValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
			Metadata: make(map[string]string),
		}, nil
	}

	require.NoError(t, coordinator.RegisterValidator(githubValidator))
	require.NoError(t, coordinator.RegisterValidator(gomodValidator))

	// Create test release
	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
		Binaries: map[string]Binary{
			"app-linux-amd64": {
				Platform:     "linux",
				Architecture: "amd64",
				Filename:     "app-linux-amd64",
				Size:         1024,
				FilePath:     filepath.Join(tempDir, "app-linux-amd64"),
			},
			"app-windows-amd64.exe": {
				Platform:     "windows",
				Architecture: "amd64",
				Filename:     "app-windows-amd64.exe",
				Size:         2048,
				FilePath:     filepath.Join(tempDir, "app-windows-amd64.exe"),
			},
		},
		Checksums: map[string]string{
			"app-linux-amd64":       "abc123",
			"app-windows-amd64.exe": "def456",
		},
		Changelog: "Test release changelog",
		Metadata: ReleaseMetadata{
			CreatedAt:    time.Now(),
			IsPrerelease: false,
		},
	}

	// Execute distribution
	ctx := context.Background()
	err := coordinator.Distribute(ctx, release)
	assert.NoError(t, err)

	// Verify publisher statuses
	statuses := coordinator.GetPublisherStatus()
	assert.Len(t, statuses, 2)
	assert.Equal(t, StatusSuccess, statuses["github"].Status)
	assert.Equal(t, StatusSuccess, statuses["gomodule"].Status)
}

func TestDistributionIntegration_PartialFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	setupTestEnvironment(t, tempDir)

	// Create failing GitHub server
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	}))
	defer failingServer.Close()

	// Create working proxy server
	proxyServer := createMockProxyServer(t)
	defer proxyServer.Close()

	config := &DistributionConfig{
		Publishers: map[string]PublisherConfig{
			"github": {
				Enabled:    true,
				Timeout:    5 * time.Second,
				RetryCount: 1,
			},
			"gomodule": {
				Enabled:    true,
				Timeout:    5 * time.Second,
				RetryCount: 1,
			},
		},
		RetryPolicy: RetryPolicy{
			MaxRetries: 1,
			BaseDelay:  time.Millisecond,
			MaxDelay:   time.Second,
		},
		ConcurrentLimit: 2,
	}

	coordinator := NewDistributionCoordinator(config)

	// Register publishers - GitHub will fail, Go module should succeed
	githubPublisher := NewGitHubPublisher(GitHubConfig{
		Owner:   "test-owner",
		Repo:    "test-repo",
		Token:   "test-token",
		BaseURL: failingServer.URL,
		Timeout: 2 * time.Second,
	})

	gomodPublisher := NewGoModulePublisher(GoModuleConfig{
		ModulePath: "github.com/test/module",
		ProxyURL:   proxyServer.URL,
		Timeout:    5 * time.Second,
		Documentation: DocumentationConfig{
			GenerateReadme: false,
		},
		Examples: ExamplesConfig{
			AutoGenerate: false,
		},
	})

	// Override methods for test
	gomodPublisher.createGitTag = func(ctx context.Context, release Release) error {
		return nil
	}
	gomodPublisher.verifyModuleAvailability = func(ctx context.Context, release Release) error {
		return nil
	}
	
	// Override validator to always pass
	gomodValidator2 := NewGoModuleValidator(GoModuleValidatorConfig{})
	gomodValidator2.ValidateRelease = func(ctx context.Context, release Release) (*PackageValidationResult, error) {
		return &PackageValidationResult{
			Valid:    true,
			Errors:   []ValidationError{},
			Warnings: []ValidationError{},
			Metadata: make(map[string]string),
		}, nil
	}
	gomodPublisher.validator = gomodValidator2

	require.NoError(t, coordinator.RegisterPublisher(githubPublisher))
	require.NoError(t, coordinator.RegisterPublisher(gomodPublisher))

	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
		Binaries: map[string]Binary{
			"app-linux": {Size: 1024, FilePath: filepath.Join(tempDir, "app-linux")},
		},
		Checksums: map[string]string{
			"app-linux": "abc123",
		},
	}

	// Execute distribution - should fail due to GitHub failure
	ctx := context.Background()
	err := coordinator.Distribute(ctx, release)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publishing failed")

	// Verify statuses
	statuses := coordinator.GetPublisherStatus()
	assert.Equal(t, StatusError, statuses["github"].Status)
	assert.Equal(t, StatusSuccess, statuses["gomodule"].Status)
}

func TestDistributionIntegration_ValidationFailure(t *testing.T) {
	config := &DistributionConfig{
		Validators: map[string]ValidatorConfig{
			"github": {Enabled: true},
		},
	}

	coordinator := NewDistributionCoordinator(config)

	validator := NewGitHubValidator(GitHubValidatorConfig{
		CheckTag: true,
	})
	require.NoError(t, coordinator.RegisterValidator(validator))

	// Create release with invalid tag
	release := Release{
		Version: "v1.0.0",
		Tag:     "invalid-tag", // Should start with 'v'
	}

	ctx := context.Background()
	err := coordinator.Distribute(ctx, release)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// Helper functions for integration tests

func setupTestEnvironment(t *testing.T, tempDir string) {
	// Create LICENSE file
	err := os.WriteFile("LICENSE", []byte("MIT License"), 0644)
	require.NoError(t, err)

	// Create README file
	err = os.WriteFile("README.md", []byte("# Test Project"), 0644)
	require.NoError(t, err)

	// Create test binary files
	binaries := []string{"app-linux-amd64", "app-windows-amd64.exe"}
	for _, binary := range binaries {
		content := []byte("fake binary content for " + binary)
		err := os.WriteFile(binary, content, 0644)
		require.NoError(t, err)
	}
}

func createMockGitHubServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test-owner/test-repo/releases":
			if r.Method == "POST" {
				response := GitHubReleaseResponse{
					ID:        123,
					TagName:   "v1.0.0",
					Name:      "Release v1.0.0",
					HTMLURL:   "https://github.com/test-owner/test-repo/releases/tag/v1.0.0",
					UploadURL: "https://uploads.github.com/repos/test-owner/test-repo/releases/123/assets{?name,label}",
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			}
		default:
			if r.Method == "POST" && r.URL.Query().Get("name") != "" {
				// Asset upload
				response := GitHubAssetResp{
					ID:   456,
					Name: r.URL.Query().Get("name"),
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
}

func createMockProxyServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/github.com/test/module/@v/v1.0.0.info" {
			response := map[string]interface{}{
				"Version": "v1.0.0",
				"Time":    "2023-01-01T00:00:00Z",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}