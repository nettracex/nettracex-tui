package distribution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitHubPublisher(t *testing.T) {
	config := GitHubConfig{
		Owner:   "test-owner",
		Repo:    "test-repo",
		Token:   "test-token",
		Timeout: 30 * time.Second,
	}
	
	publisher := NewGitHubPublisher(config)
	
	assert.NotNil(t, publisher)
	assert.Equal(t, "github", publisher.GetName())
	assert.Equal(t, StatusIdle, publisher.GetStatus().Status)
	assert.Equal(t, "https://api.github.com", publisher.config.BaseURL)
}

func TestGitHubValidator_ValidateTag(t *testing.T) {
	validator := NewGitHubValidator(GitHubValidatorConfig{})
	
	tests := []struct {
		tag   string
		valid bool
	}{
		{"v1.0.0", true},
		{"v1.2.3-alpha", true},
		{"v0.1.0", true},
		{"1.0.0", false}, // Should start with 'v'
		{"", false},      // Empty tag
		{"invalid", false}, // No 'v' prefix
	}
	
	for _, test := range tests {
		t.Run(test.tag, func(t *testing.T) {
			err := validator.validateTag(test.tag)
			if test.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestGitHubValidator_ValidateAssets(t *testing.T) {
	validator := NewGitHubValidator(GitHubValidatorConfig{
		RequiredAssets: []string{"linux", "windows", "darwin"},
	})
	
	release := Release{
		Binaries: map[string]Binary{
			"app-linux-amd64":   {Size: 1024},
			"app-windows-amd64": {Size: 2048},
			"app-darwin-amd64":  {Size: 1536},
		},
	}
	
	warnings := validator.validateAssets(release)
	assert.Len(t, warnings, 0)
	
	// Test with missing required asset
	validator.config.RequiredAssets = []string{"linux", "windows", "darwin", "freebsd"}
	warnings = validator.validateAssets(release)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Message, "freebsd")
	
	// Test with empty binary
	release.Binaries["app-empty"] = Binary{Size: 0}
	warnings = validator.validateAssets(release)
	assert.Len(t, warnings, 2) // Missing freebsd + empty binary
}

func TestGitHubValidator_ValidateRelease(t *testing.T) {
	validator := NewGitHubValidator(GitHubValidatorConfig{
		CheckAssets:    true,
		CheckChangelog: true,
		CheckTag:       true,
	})
	
	release := Release{
		Version:   "v1.0.0",
		Tag:       "v1.0.0",
		Changelog: "Initial release",
		Binaries: map[string]Binary{
			"app-linux": {Size: 1024},
		},
	}
	
	result, err := validator.ValidateRelease(context.Background(), release)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Len(t, result.Errors, 0)
	
	// Test with invalid tag
	release.Tag = "invalid"
	result, err = validator.ValidateRelease(context.Background(), release)
	assert.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
}

func TestGitHubPublisher_GenerateChangelog(t *testing.T) {
	config := GitHubConfig{
		Owner: "test-owner",
		Repo:  "test-repo",
		Changelog: ChangelogConfig{
			IncludeCommits: true,
		},
	}
	
	publisher := NewGitHubPublisher(config)
	
	release := Release{
		Version:      "v1.0.0",
		Tag:          "v1.0.0",
		ReleaseNotes: "This is a test release",
		Checksums: map[string]string{
			"app-linux":   "abc123",
			"app-windows": "def456",
		},
	}
	
	changelog, err := publisher.generateChangelog(context.Background(), release)
	assert.NoError(t, err)
	
	assert.Contains(t, changelog, "# Release v1.0.0")
	assert.Contains(t, changelog, "This is a test release")
	assert.Contains(t, changelog, "## Installation")
	assert.Contains(t, changelog, "go install github.com/test-owner/test-repo@v1.0.0")
	assert.Contains(t, changelog, "## Checksums")
	assert.Contains(t, changelog, "abc123  app-linux")
	assert.Contains(t, changelog, "def456  app-windows")
	assert.Contains(t, changelog, "## What's Changed")
}

func TestGitHubPublisher_CreateRelease(t *testing.T) {
	// Mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/test-owner/test-repo/releases", r.URL.Path)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		
		var release GitHubRelease
		err := json.NewDecoder(r.Body).Decode(&release)
		assert.NoError(t, err)
		assert.Equal(t, "v1.0.0", release.TagName)
		
		response := GitHubReleaseResponse{
			ID:        123,
			TagName:   release.TagName,
			Name:      release.Name,
			Body:      release.Body,
			HTMLURL:   "https://github.com/test-owner/test-repo/releases/tag/v1.0.0",
			UploadURL: "https://uploads.github.com/repos/test-owner/test-repo/releases/123/assets{?name,label}",
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := GitHubConfig{
		Owner:   "test-owner",
		Repo:    "test-repo",
		Token:   "test-token",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
	}
	
	publisher := NewGitHubPublisher(config)
	
	githubRelease := &GitHubRelease{
		TagName: "v1.0.0",
		Name:    "Release v1.0.0",
		Body:    "Test release",
	}
	
	response, err := publisher.createRelease(context.Background(), githubRelease)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), response.ID)
	assert.Equal(t, "v1.0.0", response.TagName)
}

func TestGitHubPublisher_CreateChecksumsFile(t *testing.T) {
	publisher := NewGitHubPublisher(GitHubConfig{})
	
	release := Release{
		Checksums: map[string]string{
			"app-linux":   "abc123def456",
			"app-windows": "789xyz012",
			"app-darwin":  "345uvw678",
		},
	}
	
	tempDir := t.TempDir()
	checksumFile := tempDir + "/checksums.txt"
	
	err := publisher.createChecksumsFile(release, checksumFile)
	assert.NoError(t, err)
	
	content, err := os.ReadFile(checksumFile)
	assert.NoError(t, err)
	
	contentStr := string(content)
	assert.Contains(t, contentStr, "abc123def456  app-linux")
	assert.Contains(t, contentStr, "789xyz012  app-windows")
	assert.Contains(t, contentStr, "345uvw678  app-darwin")
}

func TestGitHubPublisher_UploadAsset(t *testing.T) {
	// Create a temporary file to upload
	tempDir := t.TempDir()
	testFile := tempDir + "/test-binary"
	testContent := "test binary content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)
	
	// Mock GitHub upload server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.RawQuery, "name=test-binary")
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		
		// Read and verify uploaded content
		body := make([]byte, len(testContent))
		n, err := r.Body.Read(body)
		assert.NoError(t, err)
		assert.Equal(t, len(testContent), n)
		assert.Equal(t, testContent, string(body))
		
		response := GitHubAssetResp{
			ID:   456,
			Name: "test-binary",
			Size: int64(len(testContent)),
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	config := GitHubConfig{
		Token:   "test-token",
		Timeout: 5 * time.Second,
	}
	
	publisher := NewGitHubPublisher(config)
	
	release := &GitHubReleaseResponse{
		UploadURL: server.URL + "{?name,label}",
	}
	
	err = publisher.uploadAsset(context.Background(), release, "test-binary", testFile, "application/octet-stream")
	assert.NoError(t, err)
}

func TestGitHubPublisher_UploadAssets(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test binary files
	binaryFiles := map[string]string{
		"app-linux":   "linux binary content",
		"app-windows": "windows binary content",
	}
	
	binaries := make(map[string]Binary)
	for name, content := range binaryFiles {
		filePath := tempDir + "/" + name
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
		
		binaries[name] = Binary{
			Filename: name,
			FilePath: filePath,
			Size:     int64(len(content)),
		}
	}
	
	uploadedAssets := make(map[string]bool)
	
	// Mock GitHub upload server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		uploadedAssets[name] = true
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(GitHubAssetResp{Name: name})
	}))
	defer server.Close()
	
	config := GitHubConfig{
		Token: "test-token",
		Assets: AssetsConfig{
			IncludeBinaries:  true,
			IncludeChecksums: true,
		},
		Timeout: 5 * time.Second,
	}
	
	publisher := NewGitHubPublisher(config)
	
	release := &GitHubReleaseResponse{
		UploadURL: server.URL + "{?name,label}",
	}
	
	releaseData := Release{
		Binaries: binaries,
		Checksums: map[string]string{
			"app-linux":   "abc123",
			"app-windows": "def456",
		},
	}
	
	err := publisher.uploadAssets(context.Background(), release, releaseData)
	assert.NoError(t, err)
	
	// Verify all assets were uploaded
	assert.True(t, uploadedAssets["app-linux"])
	assert.True(t, uploadedAssets["app-windows"])
	assert.True(t, uploadedAssets["checksums.txt"])
}

func TestGitHubPublisher_Validate(t *testing.T) {
	config := GitHubConfig{
		Owner: "test-owner",
		Repo:  "test-repo",
	}
	
	publisher := NewGitHubPublisher(config)
	
	release := Release{
		Version: "v1.0.0",
		Tag:     "v1.0.0",
		Binaries: map[string]Binary{
			"app-linux": {Size: 1024},
		},
	}
	
	err := publisher.Validate(context.Background(), release)
	assert.NoError(t, err)
}

func TestGitHubPublisher_UpdateStatus(t *testing.T) {
	publisher := NewGitHubPublisher(GitHubConfig{})
	
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

func TestGitHubPublisher_GetCommitsSinceLastTag(t *testing.T) {
	publisher := NewGitHubPublisher(GitHubConfig{})
	
	release := Release{
		Version: "v1.0.0",
	}
	
	commits, err := publisher.getCommitsSinceLastTag(context.Background(), release)
	assert.NoError(t, err)
	assert.NotEmpty(t, commits)
	
	// Verify placeholder commits are returned
	assert.Contains(t, strings.Join(commits, " "), "network diagnostic")
}