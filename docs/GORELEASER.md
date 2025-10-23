# GoReleaser Setup and Usage

This document explains how to use GoReleaser for building and releasing NetTraceX.

## What is GoReleaser?

GoReleaser is a release automation tool for Go projects. It builds binaries for multiple platforms, creates archives, generates checksums, and publishes releases to GitHub.

## Setup

### Prerequisites

- Go 1.23 or later
- Git
- GitHub repository with push access

### Installation

#### Option 1: Using the setup script (recommended)

**Windows:**
```powershell
.\scripts\setup-goreleaser.ps1
```

**Linux/macOS:**
```bash
./scripts/setup-goreleaser.sh
```

#### Option 2: Manual installation

```bash
go install github.com/goreleaser/goreleaser@latest
```

#### Option 3: Using Makefile

```bash
make goreleaser-install
```

## Configuration

The GoReleaser configuration is in `.goreleaser.yaml`. Key features:

- **Multi-platform builds**: Linux, Windows, macOS (AMD64 and ARM64)
- **Archives**: tar.gz for Unix, zip for Windows
- **Checksums**: SHA256 checksums for all artifacts
- **GitHub releases**: Automatic release creation with changelog
- **Version injection**: Build-time version information

## Usage

### Local Development

#### Check configuration
```bash
make goreleaser-check
# or
goreleaser check
```

#### Test build (snapshot)
```bash
make goreleaser-build
# or
goreleaser build --snapshot --clean
```

#### Dry run release
```bash
make goreleaser-release-dry
# or
goreleaser release --snapshot --clean
```

### Creating Releases

#### Automated (GitHub Actions)

1. Create and push a tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Build for all platforms
   - Create GitHub release
   - Upload binaries and checksums
   - Generate changelog

#### Manual Release

```bash
# Make sure you're on the main branch and have a clean working directory
git checkout main
git pull origin main

# Create and push tag
git tag v1.0.0
git push origin v1.0.0

# Run GoReleaser (requires GITHUB_TOKEN environment variable)
export GITHUB_TOKEN="your_github_token"
goreleaser release --clean
```

## Build Artifacts

GoReleaser creates the following artifacts in the `dist/` directory:

### Binaries
- `nettracex_Linux_x86_64.tar.gz` - Linux AMD64
- `nettracex_Linux_arm64.tar.gz` - Linux ARM64
- `nettracex_Windows_x86_64.zip` - Windows AMD64
- `nettracex_Darwin_x86_64.tar.gz` - macOS AMD64
- `nettracex_Darwin_arm64.tar.gz` - macOS ARM64

### Metadata
- `checksums.txt` - SHA256 checksums
- Release notes and changelog

## GitHub Actions Workflows

### Release Workflow (`.github/workflows/release.yml`)
- Triggers on tag push (`v*`)
- Builds and releases automatically
- Requires `GITHUB_TOKEN` (automatically provided)

### Test Release Workflow (`.github/workflows/test-release.yml`)
- Triggers on PR and main branch push
- Tests GoReleaser configuration
- Creates snapshot builds for testing

## Customization

### Adding New Platforms

Edit `.goreleaser.yaml` and add to the `goos`/`goarch` lists:

```yaml
builds:
  - goos:
      - linux
      - windows
      - darwin
      - freebsd  # Add new OS
    goarch:
      - amd64
      - arm64
      - 386      # Add new architecture
```

### Package Managers

The configuration includes commented sections for:
- **Homebrew**: Automatic formula creation
- **Scoop**: Windows package manager
- **Docker**: Container images

Uncomment and configure as needed.

### Custom Archives

Modify the `archives` section to include additional files:

```yaml
archives:
  - files:
      - README.md
      - LICENSE
      - docs/**/*
      - config.example.yaml  # Add config files
```

## Troubleshooting

### Common Issues

1. **GoReleaser not found**
   - Ensure `$GOPATH/bin` is in your `$PATH`
   - Run `go env GOPATH` to check your GOPATH

2. **GitHub token issues**
   - Create a personal access token with `repo` scope
   - Set as `GITHUB_TOKEN` environment variable

3. **Build failures**
   - Check Go version compatibility
   - Ensure all dependencies are available
   - Run `go mod tidy` before building

4. **Tag already exists**
   - Delete the tag: `git tag -d v1.0.0 && git push origin :refs/tags/v1.0.0`
   - Create a new tag with incremented version

### Debug Mode

Run GoReleaser with debug output:

```bash
goreleaser release --debug --snapshot --clean
```

## Integration with Existing Build System

GoReleaser complements the existing Makefile-based build system:

- **Makefile**: Development builds, testing, local cross-compilation
- **GoReleaser**: Release builds, packaging, distribution

Both systems use the same ldflags for version injection and produce compatible binaries.

## Version Management

GoReleaser automatically:
- Uses Git tags for version numbers
- Injects version info via ldflags
- Generates changelogs from Git history
- Creates semantic version releases

Follow semantic versioning (semver) for tags:
- `v1.0.0` - Major release
- `v1.1.0` - Minor release  
- `v1.0.1` - Patch release
- `v1.0.0-beta.1` - Pre-release