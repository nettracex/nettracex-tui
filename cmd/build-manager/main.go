//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nettracex/nettracex-tui/internal/build"
)

const (
	appName     = "nettracex"
	defaultVersion = "dev"
)

func main() {
	var (
		version       = flag.String("version", getEnvOrDefault("VERSION", defaultVersion), "Version to build")
		outputDir     = flag.String("output", getEnvOrDefault("OUTPUT_DIR", "bin"), "Output directory for binaries")
		gitCommit     = flag.String("commit", getEnvOrDefault("GIT_COMMIT", "unknown"), "Git commit hash")
		compress      = flag.Bool("compress", getEnvOrDefault("COMPRESS", "false") == "true", "Enable compression")
		targets       = flag.String("targets", "", "Comma-separated list of targets (e.g., linux/amd64,windows/amd64)")
		clean         = flag.Bool("clean", false, "Clean build artifacts before building")
		validate      = flag.Bool("validate", false, "Validate build environment only")
		metadata      = flag.Bool("metadata", true, "Generate build metadata")
		checksums     = flag.Bool("checksums", true, "Generate checksums")
		wingetManifest = flag.Bool("winget", false, "Generate Winget package manifest")
		windowsInstaller = flag.Bool("windows-installer", false, "Generate Windows installer scripts")
		help          = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Handle special commands
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "winget-manifest":
			handleWingetManifest()
			return
		case "windows-installer":
			handleWindowsInstaller()
			return
		}
	}

	// Create build configuration
	config := build.BuildConfig{
		AppName:     appName,
		Version:     *version,
		GitCommit:   *gitCommit,
		BuildTime:   time.Now().UTC().Format(time.RFC3339),
		OutputDir:   *outputDir,
		Compression: getCompressionType(*compress),
	}

	// Create build manager
	bm := build.NewBuildManager(config)

	// Set custom targets if specified
	if *targets != "" {
		customTargets, err := parseTargets(*targets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing targets: %v\n", err)
			os.Exit(1)
		}
		bm.SetTargets(customTargets)
	}

	// Validate environment
	if err := bm.ValidateEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Build environment validation failed: %v\n", err)
		os.Exit(1)
	}

	if *validate {
		fmt.Println("Build environment validation successful!")
		return
	}

	// Clean if requested
	if *clean {
		fmt.Println("Cleaning build artifacts...")
		if err := bm.Clean(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to clean: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Build artifacts cleaned successfully!")
	}

	// Print build information
	printBuildInfo(config)

	// Build all targets
	fmt.Println("Starting cross-platform build...")
	if err := bm.BuildAll(); err != nil {
		fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
		os.Exit(1)
	}

	// Generate checksums if requested
	if *checksums {
		fmt.Println("Generating checksums...")
		if err := bm.GenerateChecksums(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate checksums: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate metadata if requested
	if *metadata {
		fmt.Println("Generating build metadata...")
		if err := bm.GenerateMetadata(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate metadata: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate Windows installer if requested
	if *windowsInstaller {
		fmt.Println("Generating Windows installer scripts...")
		if err := bm.GenerateWindowsInstaller(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate Windows installer: %v\n", err)
			os.Exit(1)
		}
	}

	// Generate Winget manifest if requested
	if *wingetManifest {
		fmt.Println("Generating Winget package manifest...")
		release := createReleaseFromArtifacts(bm, config)
		if err := bm.GenerateWingetManifest(release); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate Winget manifest: %v\n", err)
			os.Exit(1)
		}
	}

	// Print summary
	printBuildSummary(bm, config)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getCompressionType(compress bool) build.CompressionType {
	if compress {
		return build.CompressionGzip
	}
	return build.CompressionNone
}

func parseTargets(targets string) ([]build.BuildTarget, error) {
	var buildTargets []build.BuildTarget
	
	targetList := strings.Split(targets, ",")
	for _, target := range targetList {
		parts := strings.Split(strings.TrimSpace(target), "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid target format: %s (expected os/arch)", target)
		}
		
		os := strings.TrimSpace(parts[0])
		arch := strings.TrimSpace(parts[1])
		
		extension := ""
		if os == "windows" {
			extension = ".exe"
		}
		
		buildTarget := build.BuildTarget{
			OS:         os,
			Arch:       arch,
			CGOEnabled: false,
			OutputName: fmt.Sprintf("%s-%s-%s", appName, os, arch),
			Extension:  extension,
		}
		
		buildTargets = append(buildTargets, buildTarget)
	}
	
	return buildTargets, nil
}

func printBuildInfo(config build.BuildConfig) {
	fmt.Println("NetTraceX Build Manager")
	fmt.Println("======================")
	fmt.Printf("App Name: %s\n", config.AppName)
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Git Commit: %s\n", config.GitCommit)
	fmt.Printf("Build Time: %s\n", config.BuildTime)
	fmt.Printf("Output Directory: %s\n", config.OutputDir)
	fmt.Printf("Compression: %v\n", config.Compression != build.CompressionNone)
	fmt.Println()
}

func printBuildSummary(bm *build.BuildManager, config build.BuildConfig) {
	artifacts := bm.GetArtifacts()
	
	fmt.Println()
	fmt.Println("Build Summary")
	fmt.Println("=============")
	fmt.Printf("Total artifacts: %d\n", len(artifacts))
	fmt.Println()
	
	fmt.Println("Generated artifacts:")
	for _, artifact := range artifacts {
		fmt.Printf("  %s (%s/%s) - %d bytes - %s\n",
			artifact.Filename,
			artifact.Target.OS,
			artifact.Target.Arch,
			artifact.Size,
			artifact.Checksum[:8]+"...")
	}
	
	fmt.Println()
	fmt.Printf("All artifacts saved to: %s\n", config.OutputDir)
	
	// List all files in output directory
	fmt.Println("\nOutput directory contents:")
	if entries, err := os.ReadDir(config.OutputDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				if info, err := entry.Info(); err == nil {
					fmt.Printf("  %s (%d bytes)\n", entry.Name(), info.Size())
				} else {
					fmt.Printf("  %s\n", entry.Name())
				}
			}
		}
	}
	
	fmt.Println("\nBuild completed successfully! 🎉")
}

// handleWingetManifest handles Winget manifest generation from environment variables
func handleWingetManifest() {
	version := getEnvOrDefault("WINGET_VERSION", "")
	checksum := getEnvOrDefault("WINGET_CHECKSUM", "")
	downloadURL := getEnvOrDefault("WINGET_DOWNLOAD_URL", "")
	fileSizeStr := getEnvOrDefault("WINGET_FILE_SIZE", "0")

	if version == "" || checksum == "" || downloadURL == "" {
		fmt.Fprintf(os.Stderr, "Error: Missing required environment variables for Winget manifest generation\n")
		fmt.Fprintf(os.Stderr, "Required: WINGET_VERSION, WINGET_CHECKSUM, WINGET_DOWNLOAD_URL\n")
		os.Exit(1)
	}

	fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid WINGET_FILE_SIZE: %v\n", err)
		os.Exit(1)
	}

	// Create build configuration
	config := build.BuildConfig{
		AppName:   appName,
		Version:   version,
		GitCommit: getEnvOrDefault("GIT_COMMIT", "unknown"),
		BuildTime: time.Now().UTC().Format(time.RFC3339),
		OutputDir: getEnvOrDefault("OUTPUT_DIR", "bin"),
	}

	// Create build manager
	bm := build.NewBuildManager(config)

	// Create release from environment variables
	release := build.Release{
		Version: version,
		Tag:     "v" + version,
		Binaries: map[string]build.Binary{
			"windows-amd64": {
				Platform:     "windows",
				Architecture: "amd64",
				Filename:     "nettracex-windows-amd64.exe",
				Size:         fileSize,
				Checksum:     checksum,
				DownloadURL:  downloadURL,
			},
		},
		Checksums: map[string]string{
			"nettracex-windows-amd64.exe": checksum,
		},
		Changelog:    fmt.Sprintf("Release %s", version),
		ReleaseNotes: fmt.Sprintf("NetTraceX version %s", version),
	}

	// Generate Winget manifest
	fmt.Printf("Generating Winget manifest for version %s...\n", version)
	if err := bm.GenerateWingetManifest(release); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Winget manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Winget manifest generated successfully!")
}

// handleWindowsInstaller handles Windows installer generation
func handleWindowsInstaller() {
	outputDir := getEnvOrDefault("OUTPUT_DIR", "bin")
	
	// Create build configuration
	config := build.BuildConfig{
		AppName:   appName,
		Version:   getEnvOrDefault("VERSION", defaultVersion),
		GitCommit: getEnvOrDefault("GIT_COMMIT", "unknown"),
		BuildTime: time.Now().UTC().Format(time.RFC3339),
		OutputDir: outputDir,
	}

	// Create build manager
	bm := build.NewBuildManager(config)

	fmt.Println("Generating Windows installer scripts...")
	if err := bm.GenerateWindowsInstaller(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Windows installer: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Windows installer scripts generated successfully!")
}

// createReleaseFromArtifacts creates a Release struct from build artifacts
func createReleaseFromArtifacts(bm *build.BuildManager, config build.BuildConfig) build.Release {
	artifacts := bm.GetArtifacts()
	binaries := make(map[string]build.Binary)
	checksums := make(map[string]string)

	for _, artifact := range artifacts {
		platform := fmt.Sprintf("%s-%s", artifact.Target.OS, artifact.Target.Arch)
		
		binary := build.Binary{
			Platform:     artifact.Target.OS,
			Architecture: artifact.Target.Arch,
			Filename:     artifact.Filename,
			Size:         artifact.Size,
			Checksum:     artifact.Checksum,
			DownloadURL:  fmt.Sprintf("https://github.com/nettracex/nettracex-tui/releases/download/v%s/%s", 
				config.Version, artifact.Filename),
		}
		
		binaries[platform] = binary
		checksums[artifact.Filename] = artifact.Checksum
	}

	return build.Release{
		Version:      config.Version,
		Tag:          "v" + config.Version,
		Binaries:     binaries,
		Checksums:    checksums,
		Changelog:    fmt.Sprintf("Release %s", config.Version),
		ReleaseNotes: fmt.Sprintf("NetTraceX version %s - Network diagnostic toolkit with beautiful TUI", config.Version),
	}
}

func showHelp() {
	fmt.Println("NetTraceX Build Manager")
	fmt.Println("=======================")
	fmt.Println()
	fmt.Println("A cross-platform build tool for NetTraceX network diagnostic toolkit.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [OPTIONS] [COMMAND]\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  winget-manifest     Generate Winget package manifest from environment variables")
	fmt.Println("  windows-installer   Generate Windows installer scripts")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -version string        Version to build (default: dev)")
	fmt.Println("  -output string         Output directory for binaries (default: bin)")
	fmt.Println("  -commit string         Git commit hash (default: unknown)")
	fmt.Println("  -compress              Enable compression of binaries")
	fmt.Println("  -targets string        Comma-separated list of targets (e.g., linux/amd64,windows/amd64)")
	fmt.Println("  -clean                 Clean build artifacts before building")
	fmt.Println("  -validate              Validate build environment only")
	fmt.Println("  -metadata              Generate build metadata (default: true)")
	fmt.Println("  -checksums             Generate checksums (default: true)")
	fmt.Println("  -winget                Generate Winget package manifest")
	fmt.Println("  -windows-installer     Generate Windows installer scripts")
	fmt.Println("  -help                  Show this help message")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  VERSION                Version to build")
	fmt.Println("  OUTPUT_DIR             Output directory for binaries")
	fmt.Println("  GIT_COMMIT             Git commit hash")
	fmt.Println("  COMPRESS               Enable compression (true/false)")
	fmt.Println()
	fmt.Println("Winget Manifest Environment Variables:")
	fmt.Println("  WINGET_VERSION         Version for Winget manifest")
	fmt.Println("  WINGET_CHECKSUM        SHA256 checksum of Windows binary")
	fmt.Println("  WINGET_DOWNLOAD_URL    Download URL for Windows binary")
	fmt.Println("  WINGET_FILE_SIZE       File size in bytes")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Build all default platforms")
	fmt.Printf("  %s\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("  # Build with specific version and compression")
	fmt.Printf("  %s -version 1.0.0 -compress\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("  # Build only for Linux and Windows")
	fmt.Printf("  %s -targets linux/amd64,windows/amd64\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("  # Generate Winget manifest and Windows installer")
	fmt.Printf("  %s -winget -windows-installer\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("  # Generate Winget manifest from environment variables")
	fmt.Printf("  %s winget-manifest\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("  # Validate build environment only")
	fmt.Printf("  %s -validate\n", filepath.Base(os.Args[0]))
	fmt.Println()
	fmt.Println("Default Supported Platforms:")
	fmt.Println("  - Linux (amd64, arm64)")
	fmt.Println("  - Windows (amd64)")
	fmt.Println("  - macOS (amd64, arm64)")
	fmt.Println()
}