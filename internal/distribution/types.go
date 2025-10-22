package distribution

import (
	"time"
)

// Release represents a software release across all platforms
type Release struct {
	Version      string            `json:"version"`
	Tag          string            `json:"tag"`
	Binaries     map[string]Binary `json:"binaries"`
	Checksums    map[string]string `json:"checksums"`
	Changelog    string            `json:"changelog"`
	ReleaseNotes string            `json:"release_notes"`
	Metadata     ReleaseMetadata   `json:"metadata"`
}

// Binary represents a platform-specific executable
type Binary struct {
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	Checksum     string `json:"checksum"`
	DownloadURL  string `json:"download_url"`
	FilePath     string `json:"file_path"`
}

// ReleaseMetadata contains additional release information
type ReleaseMetadata struct {
	CreatedAt    time.Time         `json:"created_at"`
	Author       string            `json:"author"`
	CommitSHA    string            `json:"commit_sha"`
	BuildNumber  string            `json:"build_number"`
	IsPrerelease bool              `json:"is_prerelease"`
	Tags         []string          `json:"tags"`
	Assets       map[string]string `json:"assets"`
}

// PublishStatus represents the status of a publisher
type PublishStatus struct {
	Name         string            `json:"name"`
	Status       StatusType        `json:"status"`
	LastPublish  time.Time         `json:"last_publish"`
	LastError    string            `json:"last_error"`
	PublishCount int               `json:"publish_count"`
	ErrorCount   int               `json:"error_count"`
	Metadata     map[string]string `json:"metadata"`
}

// StatusType represents the current status of a publisher
type StatusType string

const (
	StatusIdle       StatusType = "idle"
	StatusPublishing StatusType = "publishing"
	StatusSuccess    StatusType = "success"
	StatusError      StatusType = "error"
	StatusDisabled   StatusType = "disabled"
)

// GoModuleInfo contains Go module specific information
type GoModuleInfo struct {
	ModulePath   string            `json:"module_path"`
	Version      string            `json:"version"`
	GoVersion    string            `json:"go_version"`
	Dependencies []Dependency      `json:"dependencies"`
	Documentation Documentation    `json:"documentation"`
	Examples     []Example         `json:"examples"`
	Metadata     map[string]string `json:"metadata"`
}

// Dependency represents a Go module dependency
type Dependency struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Type    string `json:"type"` // direct, indirect, test
}

// Documentation contains module documentation information
type Documentation struct {
	README       string            `json:"readme"`
	GoDoc        string            `json:"godoc"`
	Examples     []Example         `json:"examples"`
	Badges       []Badge           `json:"badges"`
	Links        map[string]string `json:"links"`
}

// Example represents a code example
type Example struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Output      string `json:"output"`
	Runnable    bool   `json:"runnable"`
}

// Badge represents a documentation badge
type Badge struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	LinkURL  string `json:"link_url"`
	Alt      string `json:"alt"`
}

// GitHubReleaseInfo contains GitHub release specific information
type GitHubReleaseInfo struct {
	Owner        string            `json:"owner"`
	Repo         string            `json:"repo"`
	TagName      string            `json:"tag_name"`
	Name         string            `json:"name"`
	Body         string            `json:"body"`
	Draft        bool              `json:"draft"`
	Prerelease   bool              `json:"prerelease"`
	Assets       []GitHubAsset     `json:"assets"`
	TargetCommit string            `json:"target_commit"`
	Metadata     map[string]string `json:"metadata"`
}

// GitHubAsset represents a GitHub release asset
type GitHubAsset struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
	FilePath    string `json:"file_path"`
}

// PackageValidationResult contains validation results
type PackageValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
	Metadata map[string]string `json:"metadata"`
}

// ValidationError represents a validation error or warning
type ValidationError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Field       string `json:"field"`
	Severity    string `json:"severity"`
	Suggestion  string `json:"suggestion"`
}

// PublishResult contains the result of a publish operation
type PublishResult struct {
	Success     bool              `json:"success"`
	Publisher   string            `json:"publisher"`
	Release     Release           `json:"release"`
	PublishedAt time.Time         `json:"published_at"`
	URL         string            `json:"url"`
	Errors      []string          `json:"errors"`
	Warnings    []string          `json:"warnings"`
	Metadata    map[string]string `json:"metadata"`
}

// DistributionReport contains a summary of distribution results
type DistributionReport struct {
	Release        Release                    `json:"release"`
	StartTime      time.Time                  `json:"start_time"`
	EndTime        time.Time                  `json:"end_time"`
	Duration       time.Duration              `json:"duration"`
	TotalPublishers int                       `json:"total_publishers"`
	SuccessCount   int                        `json:"success_count"`
	ErrorCount     int                        `json:"error_count"`
	Results        map[string]PublishResult   `json:"results"`
	Summary        string                     `json:"summary"`
}