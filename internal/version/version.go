package version

import (
	"fmt"
	"runtime/debug"
)

// Build-time variables (set by ldflags in GoReleaser)
var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

// Info contains version information
type Info struct {
	Version   string
	GitCommit string
	BuildTime string
	GoVersion string
}

// Get returns version information, attempting to detect from build info if not set by ldflags
func Get() Info {
	info := Info{
		Version:   version,
		GitCommit: gitCommit,
		BuildTime: buildTime,
		GoVersion: "unknown",
	}

	// Get build info
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		// Set Go version
		info.GoVersion = buildInfo.GoVersion

		// If version is still "dev", try to get it from build info (go install case)
		if info.Version == "dev" {
			// Try to get version from module info first
			if buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
				info.Version = buildInfo.Main.Version
			}

			// Look for VCS info in build settings
			for _, setting := range buildInfo.Settings {
				switch setting.Key {
				case "vcs.revision":
					if info.GitCommit == "unknown" {
						info.GitCommit = setting.Value
						if len(info.GitCommit) > 7 {
							info.GitCommit = info.GitCommit[:7]
						}
					}
				case "vcs.time":
					if info.BuildTime == "unknown" {
						info.BuildTime = setting.Value
					}
				}
			}
		}
	}

	return info
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("NetTraceX %s", i.Version)
}

// Detailed returns a detailed version string with all information
func (i Info) Detailed() string {
	return fmt.Sprintf(`NetTraceX %s
Git Commit: %s
Build Time: %s
Go Version: %s`, i.Version, i.GitCommit, i.BuildTime, i.GoVersion)
}