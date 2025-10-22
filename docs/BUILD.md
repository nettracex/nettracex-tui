# NetTraceX Build System

This document describes the cross-platform build system for NetTraceX, a comprehensive network diagnostic toolkit.

## Overview

The NetTraceX build system provides multiple ways to build the application for different platforms and architectures:

1. **Makefile** - Traditional make-based build system with comprehensive targets
2. **PowerShell Script** - Windows-friendly build script with full functionality
3. **Bash Script** - Unix/Linux build script for cross-platform compilation
4. **Go Build Manager** - Native Go tool for advanced build management
5. **GitHub Actions** - Automated CI/CD pipeline for continuous integration

## Supported Platforms

- **Linux**: amd64, arm64
- **Windows**: amd64
- **macOS**: amd64, arm64

## Build Methods

### 1. Makefile (Linux/macOS/WSL)

The Makefile provides comprehensive build targets with colored output and detailed configuration options.

#### Basic Usage

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Release build with compression
make release-build

# Clean build artifacts
make clean

# Show help
make help
```

#### Platform-Specific Builds

```bash
# Linux builds
make build-linux
make build-linux-amd64
make build-linux-arm64

# Windows builds
make build-windows
make build-windows-amd64

# macOS builds
make build-darwin
make build-darwin-amd64
make build-darwin-arm64
```

#### Configuration

```bash
# Custom version
VERSION=1.0.0 make build-all

# Enable compression
COMPRESS=true make build-all

# Custom output directory
OUTPUT_DIR=dist make build-all

# Custom git commit
GIT_COMMIT=abc123 make build-all
```

### 2. PowerShell Script (Windows)

The PowerShell script provides full build functionality on Windows systems.

#### Basic Usage

```powershell
# Build all platforms
.\scripts\build.ps1 all

# Build with compression
.\scripts\build.ps1 all -Compress

# Build with custom version
.\scripts\build.ps1 all -Version "1.0.0"

# Validate environment
.\scripts\build.ps1 validate

# Clean artifacts
.\scripts\build.ps1 clean

# Show help
.\scripts\build.ps1 help
```

#### Advanced Options

```powershell
# Custom configuration
.\scripts\build.ps1 all -Version "1.0.0" -OutputDir "dist" -Compress -GitCommit "abc123"
```

### 3. Bash Script (Linux/macOS)

The bash script provides cross-platform build capabilities for Unix-like systems.

#### Basic Usage

```bash
# Build all platforms
./scripts/build.sh all

# Build with compression
COMPRESS=true ./scripts/build.sh all

# Build with custom version
VERSION=1.0.0 ./scripts/build.sh all

# Validate environment
./scripts/build.sh validate

# Clean artifacts
./scripts/build.sh clean

# Show help
./scripts/build.sh help
```

#### Environment Variables

```bash
export VERSION="1.0.0"
export OUTPUT_DIR="dist"
export COMPRESS="true"
export GIT_COMMIT="abc123"
./scripts/build.sh all
```

### 4. Go Build Manager

The Go build manager provides programmatic build control with advanced features.

#### Basic Usage

```bash
# Build all platforms
go run ./cmd/build-manager/

# Build specific targets
go run ./cmd/build-manager/ -targets "linux/amd64,windows/amd64"

# Build with compression
go run ./cmd/build-manager/ -compress

# Validate environment only
go run ./cmd/build-manager/ -validate

# Show help
go run ./cmd/build-manager/ -help
```

#### Advanced Options

```bash
# Full configuration
go run ./cmd/build-manager/ \
  -version "1.0.0" \
  -commit "abc123" \
  -output "dist" \
  -compress \
  -targets "linux/amd64,windows/amd64,darwin/arm64"
```

## Build Artifacts

### Generated Files

Each build process generates the following artifacts:

- **Binaries**: Platform-specific executables
- **Checksums**: SHA256 checksums for all binaries (`checksums.txt`)
- **Metadata**: Build information in JSON format (`build-metadata.json`)
- **Compressed Archives**: Optional compressed binaries (`.tar.gz` or `.zip`)

### Directory Structure

```
bin/
├── nettracex-linux-amd64
├── nettracex-linux-arm64
├── nettracex-windows-amd64.exe
├── nettracex-darwin-amd64
├── nettracex-darwin-arm64
├── checksums.txt
└── build-metadata.json
```

### Metadata Format

The `build-metadata.json` file contains:

```json
{
  "app_name": "nettracex",
  "version": "1.0.0",
  "git_commit": "abc123",
  "build_time": "2023-01-01T00:00:00Z",
  "go_version": "go1.21.0",
  "build_host": "Linux hostname 5.4.0",
  "artifacts": [
    {
      "filename": "nettracex-linux-amd64",
      "size": 8385536,
      "checksum": "sha256:0a45b5a4..."
    }
  ]
}
```

## Development Workflow

### Prerequisites

1. **Go 1.21+**: Required for building
2. **Git**: For commit hash generation (optional)
3. **Make**: For Makefile usage (Linux/macOS)
4. **PowerShell**: For Windows build script

### Environment Validation

Before building, validate your environment:

```bash
# Using Makefile
make validate-build

# Using PowerShell
.\scripts\build.ps1 validate

# Using Bash
./scripts/build.sh validate

# Using Go build manager
go run ./cmd/build-manager/ -validate
```

### Testing

Run build validation tests:

```bash
# Run all build tests
go test -v ./internal/build/...

# Run specific test
go test -v ./internal/build/... -run TestBuildValidation

# Run benchmarks
go test -v ./internal/build/... -bench=.
```

### Development Build

For development, use the simple build command:

```bash
# Quick build for current platform
go build -o bin/nettracex ./

# Or using make
make build
```

## CI/CD Integration

### GitHub Actions

The repository includes a comprehensive GitHub Actions workflow (`.github/workflows/build.yml`) that:

1. **Tests**: Runs tests and validation on multiple platforms
2. **Builds**: Creates binaries for all supported platforms
3. **Releases**: Automatically creates releases with artifacts
4. **Security**: Performs security scanning and vulnerability checks

### Workflow Triggers

- **Push**: To main/develop branches
- **Pull Request**: To main branch
- **Release**: When creating a new release tag

### Artifacts

The CI/CD pipeline generates:

- Individual platform binaries
- Compressed archives
- Checksums and metadata
- Coverage reports
- Security scan results

## Troubleshooting

### Common Issues

1. **Go Not Found**
   ```
   Error: Go is not installed or not in PATH
   Solution: Install Go 1.21+ and ensure it's in your PATH
   ```

2. **Build Fails**
   ```
   Error: Build failed for target
   Solution: Check Go version, dependencies, and target validity
   ```

3. **Permission Denied**
   ```
   Error: Permission denied creating output directory
   Solution: Ensure write permissions to output directory
   ```

### Debug Mode

Enable verbose output for debugging:

```bash
# Makefile with verbose output
make build-all V=1

# Go build manager with debug
go run ./cmd/build-manager/ -targets "linux/amd64" 2>&1 | tee build.log
```

### Environment Information

Check your build environment:

```bash
# Go environment
go env

# Build tool versions
go version
git --version
make --version  # Linux/macOS
```

## Advanced Configuration

### Custom Build Targets

Add custom build targets by modifying the build manager:

```go
customTarget := build.BuildTarget{
    OS:         "freebsd",
    Arch:       "amd64",
    CGOEnabled: false,
    OutputName: "nettracex-freebsd-amd64",
    LDFlags:    []string{"-custom-flag"},
    Tags:       []string{"custom-tag"},
}
```

### Build Flags

Customize build flags through environment variables or command-line options:

- **LDFLAGS**: Custom linker flags
- **TAGS**: Build tags
- **CGO_ENABLED**: Enable/disable CGO

### Compression Options

The build system supports multiple compression formats:

- **None**: No compression (fastest)
- **Gzip**: tar.gz format (good compression)
- **Zip**: zip format (Windows-friendly)

## Performance

### Build Times

Typical build times on modern hardware:

- **Single platform**: 10-30 seconds
- **All platforms**: 1-3 minutes
- **With compression**: +20-50% time

### Optimization

For faster builds:

1. Use specific targets instead of building all platforms
2. Disable compression for development builds
3. Use Go build cache (`GOCACHE`)
4. Use parallel builds where possible

## Security

### Code Signing

The build system includes placeholder support for code signing:

```go
signingConfig := &build.SigningConfig{
    Enabled:     true,
    Certificate: "path/to/cert.crt",
    KeyFile:     "path/to/key.key",
    Password:    "signing-password",
}
```

### Checksums

All binaries include SHA256 checksums for integrity verification:

```bash
# Verify checksum
sha256sum -c checksums.txt
```

### Supply Chain Security

- Dependencies are verified with `go mod verify`
- Vulnerability scanning with `govulncheck`
- Security scanning with `gosec`
- License compliance checking

## Contributing

When contributing to the build system:

1. Test all build methods on your platform
2. Update documentation for new features
3. Add tests for new functionality
4. Ensure backward compatibility
5. Update CI/CD workflows if needed

## Support

For build system issues:

1. Check this documentation
2. Run environment validation
3. Check GitHub Issues
4. Review CI/CD logs for similar problems
5. Create a new issue with build logs and environment info