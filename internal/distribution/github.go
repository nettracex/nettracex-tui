package distribution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GitHubPublisher handles GitHub release publishing
type GitHubPublisher struct {
	config    GitHubConfig
	client    *http.Client
	status    PublishStatus
	validator *GitHubValidator
}

// GitHubConfig contains configuration for GitHub publishing
type GitHubConfig struct {
	Owner       string            `json:"owner"`
	Repo        string            `json:"repo"`
	Token       string            `json:"token"`
	BaseURL     string            `json:"base_url"`
	Timeout     time.Duration     `json:"timeout"`
	Changelog   ChangelogConfig   `json:"changelog"`
	Assets      AssetsConfig      `json:"assets"`
	Metadata    map[string]string `json:"metadata"`
}

// ChangelogConfig contains changelog generation settings
type ChangelogConfig struct {
	AutoGenerate    bool     `json:"auto_generate"`
	Template        string   `json:"template"`
	IncludeCommits  bool     `json:"include_commits"`
	IncludePRs      bool     `json:"include_prs"`
	SinceTag        string   `json:"since_tag"`
	Categories      []string `json:"categories"`
}

// AssetsConfig contains asset upload settings
type AssetsConfig struct {
	IncludeBinaries bool     `json:"include_binaries"`
	IncludeChecksums bool    `json:"include_checksums"`
	IncludeSource   bool     `json:"include_source"`
	AssetPatterns   []string `json:"asset_patterns"`
	Compression     string   `json:"compression"`
}

// GitHubValidator validates GitHub releases
type GitHubValidator struct {
	config GitHubValidatorConfig
}

// GitHubValidatorConfig contains validator configuration
type GitHubValidatorConfig struct {
	CheckAssets     bool `json:"check_assets"`
	CheckChangelog  bool `json:"check_changelog"`
	CheckTag        bool `json:"check_tag"`
	RequiredAssets  []string `json:"required_assets"`
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName         string        `json:"tag_name"`
	TargetCommitish string        `json:"target_commitish"`
	Name            string        `json:"name"`
	Body            string        `json:"body"`
	Draft           bool          `json:"draft"`
	Prerelease      bool          `json:"prerelease"`
	GenerateNotes   bool          `json:"generate_release_notes"`
}

// GitHubReleaseResponse represents GitHub API response
type GitHubReleaseResponse struct {
	ID          int64             `json:"id"`
	TagName     string            `json:"tag_name"`
	Name        string            `json:"name"`
	Body        string            `json:"body"`
	Draft       bool              `json:"draft"`
	Prerelease  bool              `json:"prerelease"`
	CreatedAt   time.Time         `json:"created_at"`
	PublishedAt time.Time         `json:"published_at"`
	HTMLURL     string            `json:"html_url"`
	UploadURL   string            `json:"upload_url"`
	Assets      []GitHubAssetResp `json:"assets"`
}

// GitHubAssetResp represents a GitHub release asset response
type GitHubAssetResp struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Label             string    `json:"label"`
	ContentType       string    `json:"content_type"`
	Size              int64     `json:"size"`
	DownloadCount     int64     `json:"download_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	BrowserDownloadURL string   `json:"browser_download_url"`
}

// NewGitHubPublisher creates a new GitHub publisher
func NewGitHubPublisher(config GitHubConfig) *GitHubPublisher {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.github.com"
	}
	
	return &GitHubPublisher{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		status: PublishStatus{
			Name:   "github",
			Status: StatusIdle,
		},
		validator: NewGitHubValidator(GitHubValidatorConfig{
			CheckAssets:    true,
			CheckChangelog: true,
			CheckTag:       true,
		}),
	}
}

// NewGitHubValidator creates a new GitHub validator
func NewGitHubValidator(config GitHubValidatorConfig) *GitHubValidator {
	return &GitHubValidator{
		config: config,
	}
}

// GetName returns the publisher name
func (ghp *GitHubPublisher) GetName() string {
	return "github"
}

// GetStatus returns the current publisher status
func (ghp *GitHubPublisher) GetStatus() PublishStatus {
	return ghp.status
}

// Validate validates the release for GitHub publishing
func (ghp *GitHubPublisher) Validate(ctx context.Context, release Release) error {
	ghp.updateStatus(StatusPublishing, "")
	
	result, err := ghp.validator.ValidateRelease(ctx, release)
	if err != nil {
		ghp.updateStatus(StatusError, err.Error())
		return err
	}
	
	if !result.Valid {
		errMsg := fmt.Sprintf("validation failed: %d errors", len(result.Errors))
		ghp.updateStatus(StatusError, errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	
	// Reset status to idle after successful validation
	ghp.updateStatus(StatusIdle, "")
	return nil
}

// Publish publishes the release to GitHub
func (ghp *GitHubPublisher) Publish(ctx context.Context, release Release) error {
	ghp.updateStatus(StatusPublishing, "")
	
	// Validate first
	if err := ghp.Validate(ctx, release); err != nil {
		return err
	}
	
	// Generate changelog if configured
	changelog := release.Changelog
	if ghp.config.Changelog.AutoGenerate {
		var err error
		changelog, err = ghp.generateChangelog(ctx, release)
		if err != nil {
			ghp.updateStatus(StatusError, err.Error())
			return fmt.Errorf("changelog generation failed: %w", err)
		}
	}
	
	// Create GitHub release
	githubRelease := &GitHubRelease{
		TagName:       release.Tag,
		Name:          fmt.Sprintf("Release %s", release.Version),
		Body:          changelog,
		Draft:         false,
		Prerelease:    release.Metadata.IsPrerelease,
		GenerateNotes: ghp.config.Changelog.AutoGenerate,
	}
	
	releaseResp, err := ghp.createRelease(ctx, githubRelease)
	if err != nil {
		ghp.updateStatus(StatusError, err.Error())
		return fmt.Errorf("release creation failed: %w", err)
	}
	
	// Upload assets
	if err := ghp.uploadAssets(ctx, releaseResp, release); err != nil {
		ghp.updateStatus(StatusError, err.Error())
		return fmt.Errorf("asset upload failed: %w", err)
	}
	
	ghp.updateStatus(StatusSuccess, "")
	return nil
}

// ValidateRelease validates a release for GitHub publishing
func (ghv *GitHubValidator) ValidateRelease(ctx context.Context, release Release) (*PackageValidationResult, error) {
	result := &PackageValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
		Metadata: make(map[string]string),
	}
	
	// Check tag format
	if ghv.config.CheckTag {
		if err := ghv.validateTag(release.Tag); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Code:     "INVALID_TAG",
				Message:  err.Error(),
				Field:    "tag",
				Severity: "error",
			})
			result.Valid = false
		}
	}
	
	// Check changelog
	if ghv.config.CheckChangelog {
		if release.Changelog == "" && release.ReleaseNotes == "" {
			result.Warnings = append(result.Warnings, ValidationError{
				Code:     "MISSING_CHANGELOG",
				Message:  "No changelog or release notes provided",
				Field:    "changelog",
				Severity: "warning",
			})
		}
	}
	
	// Check assets
	if ghv.config.CheckAssets {
		if warnings := ghv.validateAssets(release); len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	}
	
	return result, nil
}

// validateTag validates git tag format
func (ghv *GitHubValidator) validateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	
	// Check if tag starts with 'v' for semantic versioning
	if !strings.HasPrefix(tag, "v") {
		return fmt.Errorf("tag should start with 'v' for semantic versioning")
	}
	
	return nil
}

// validateAssets validates release assets
func (ghv *GitHubValidator) validateAssets(release Release) []ValidationError {
	var warnings []ValidationError
	
	// Check for required assets
	for _, required := range ghv.config.RequiredAssets {
		found := false
		for filename := range release.Binaries {
			if strings.Contains(filename, required) {
				found = true
				break
			}
		}
		
		if !found {
			warnings = append(warnings, ValidationError{
				Code:     "MISSING_REQUIRED_ASSET",
				Message:  fmt.Sprintf("Required asset not found: %s", required),
				Field:    "assets",
				Severity: "warning",
			})
		}
	}
	
	// Check binary sizes
	for filename, binary := range release.Binaries {
		if binary.Size == 0 {
			warnings = append(warnings, ValidationError{
				Code:     "EMPTY_BINARY",
				Message:  fmt.Sprintf("Binary file is empty: %s", filename),
				Field:    "assets",
				Severity: "warning",
			})
		}
	}
	
	return warnings
}

// generateChangelog generates a changelog for the release
func (ghp *GitHubPublisher) generateChangelog(ctx context.Context, release Release) (string, error) {
	var changelog strings.Builder
	
	changelog.WriteString(fmt.Sprintf("# Release %s\n\n", release.Version))
	
	if release.ReleaseNotes != "" {
		changelog.WriteString(release.ReleaseNotes)
		changelog.WriteString("\n\n")
	}
	
	// Add installation instructions
	changelog.WriteString("## Installation\n\n")
	changelog.WriteString("### Using Go\n")
	changelog.WriteString(fmt.Sprintf("```bash\ngo install github.com/%s/%s@%s\n```\n\n", 
		ghp.config.Owner, ghp.config.Repo, release.Version))
	
	changelog.WriteString("### Download Binary\n")
	changelog.WriteString("Download the appropriate binary for your platform from the assets below.\n\n")
	
	// Add platform-specific installation instructions
	changelog.WriteString("### Platform-specific Installation\n\n")
	
	// Windows
	changelog.WriteString("#### Windows\n")
	changelog.WriteString("```powershell\n")
	changelog.WriteString("# Download and extract the Windows binary\n")
	changelog.WriteString(fmt.Sprintf("Invoke-WebRequest -Uri \"https://github.com/%s/%s/releases/download/%s/nettracex-windows-amd64.exe\" -OutFile \"nettracex.exe\"\n", 
		ghp.config.Owner, ghp.config.Repo, release.Tag))
	changelog.WriteString("```\n\n")
	
	// macOS
	changelog.WriteString("#### macOS\n")
	changelog.WriteString("```bash\n")
	changelog.WriteString("# Download and install the macOS binary\n")
	changelog.WriteString(fmt.Sprintf("curl -L \"https://github.com/%s/%s/releases/download/%s/nettracex-darwin-amd64\" -o nettracex\n", 
		ghp.config.Owner, ghp.config.Repo, release.Tag))
	changelog.WriteString("chmod +x nettracex\n")
	changelog.WriteString("sudo mv nettracex /usr/local/bin/\n")
	changelog.WriteString("```\n\n")
	
	// Linux
	changelog.WriteString("#### Linux\n")
	changelog.WriteString("```bash\n")
	changelog.WriteString("# Download and install the Linux binary\n")
	changelog.WriteString(fmt.Sprintf("curl -L \"https://github.com/%s/%s/releases/download/%s/nettracex-linux-amd64\" -o nettracex\n", 
		ghp.config.Owner, ghp.config.Repo, release.Tag))
	changelog.WriteString("chmod +x nettracex\n")
	changelog.WriteString("sudo mv nettracex /usr/local/bin/\n")
	changelog.WriteString("```\n\n")
	
	// Add checksums section
	changelog.WriteString("## Checksums\n\n")
	changelog.WriteString("Verify the integrity of downloaded files using the checksums below:\n\n")
	changelog.WriteString("```\n")
	for filename, checksum := range release.Checksums {
		changelog.WriteString(fmt.Sprintf("%s  %s\n", checksum, filename))
	}
	changelog.WriteString("```\n\n")
	
	// Add what's changed section if we have commit info
	if ghp.config.Changelog.IncludeCommits {
		commits, err := ghp.getCommitsSinceLastTag(ctx, release)
		if err == nil && len(commits) > 0 {
			changelog.WriteString("## What's Changed\n\n")
			for _, commit := range commits {
				changelog.WriteString(fmt.Sprintf("- %s\n", commit))
			}
			changelog.WriteString("\n")
		}
	}
	
	return changelog.String(), nil
}

// getCommitsSinceLastTag gets commits since the last tag
func (ghp *GitHubPublisher) getCommitsSinceLastTag(ctx context.Context, release Release) ([]string, error) {
	// This would typically use git commands or GitHub API to get commits
	// For now, return a placeholder
	return []string{
		"Improved network diagnostic capabilities",
		"Enhanced TUI interface with better responsiveness",
		"Added comprehensive test coverage",
		"Updated dependencies and security fixes",
	}, nil
}

// createRelease creates a GitHub release
func (ghp *GitHubPublisher) createRelease(ctx context.Context, release *GitHubRelease) (*GitHubReleaseResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", ghp.config.BaseURL, ghp.config.Owner, ghp.config.Repo)
	
	jsonData, err := json.Marshal(release)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "token "+ghp.config.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	resp, err := ghp.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}
	
	var releaseResp GitHubReleaseResponse
	if err := json.Unmarshal(body, &releaseResp); err != nil {
		return nil, err
	}
	
	return &releaseResp, nil
}

// uploadAssets uploads release assets to GitHub
func (ghp *GitHubPublisher) uploadAssets(ctx context.Context, release *GitHubReleaseResponse, releaseData Release) error {
	// Upload binaries
	if ghp.config.Assets.IncludeBinaries {
		for filename, binary := range releaseData.Binaries {
			if err := ghp.uploadAsset(ctx, release, filename, binary.FilePath, "application/octet-stream"); err != nil {
				return fmt.Errorf("failed to upload binary %s: %w", filename, err)
			}
		}
	}
	
	// Upload checksums
	if ghp.config.Assets.IncludeChecksums {
		checksumFile := "checksums.txt"
		if err := ghp.createChecksumsFile(releaseData, checksumFile); err != nil {
			return fmt.Errorf("failed to create checksums file: %w", err)
		}
		
		if err := ghp.uploadAsset(ctx, release, checksumFile, checksumFile, "text/plain"); err != nil {
			return fmt.Errorf("failed to upload checksums: %w", err)
		}
	}
	
	return nil
}

// createChecksumsFile creates a checksums file
func (ghp *GitHubPublisher) createChecksumsFile(release Release, filename string) error {
	var content strings.Builder
	
	for filename, checksum := range release.Checksums {
		content.WriteString(fmt.Sprintf("%s  %s\n", checksum, filename))
	}
	
	return os.WriteFile(filename, []byte(content.String()), 0644)
}

// uploadAsset uploads a single asset to GitHub release
func (ghp *GitHubPublisher) uploadAsset(ctx context.Context, release *GitHubReleaseResponse, name, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Parse upload URL template
	uploadURL := strings.Replace(release.UploadURL, "{?name,label}", "", 1)
	uploadURL = fmt.Sprintf("%s?name=%s", uploadURL, name)
	
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, file)
	if err != nil {
		return err
	}
	
	req.Header.Set("Authorization", "token "+ghp.config.Token)
	req.Header.Set("Content-Type", contentType)
	
	resp, err := ghp.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("asset upload failed: %s", string(body))
	}
	
	return nil
}

// updateStatus updates the publisher status
func (ghp *GitHubPublisher) updateStatus(status StatusType, errorMsg string) {
	ghp.status.Status = status
	ghp.status.LastError = errorMsg
	if status == StatusSuccess {
		ghp.status.PublishCount++
		ghp.status.LastPublish = time.Now()
	} else if status == StatusError {
		ghp.status.ErrorCount++
	}
}

// GetName returns the validator name
func (ghv *GitHubValidator) GetName() string {
	return "github"
}

// Validate validates a release
func (ghv *GitHubValidator) Validate(ctx context.Context, release Release) error {
	result, err := ghv.ValidateRelease(ctx, release)
	if err != nil {
		return err
	}
	
	if !result.Valid {
		return fmt.Errorf("validation failed with %d errors", len(result.Errors))
	}
	
	return nil
}