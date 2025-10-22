# Homebrew Formula Creation and Publishing

This document describes the Homebrew formula creation and publishing functionality for NetTraceX.

## Overview

The Homebrew publisher automatically creates and maintains Homebrew formulas for macOS installation of NetTraceX. It supports both custom taps and submission to homebrew-core.

## Features

- **Automatic Formula Generation**: Creates Homebrew formulas from release binaries
- **Cross-Platform Support**: Supports both macOS and Linux (Homebrew on Linux)
- **Formula Validation**: Validates formulas using `brew audit`
- **Installation Testing**: Tests formula installation in CI/CD
- **Custom Tap Support**: Publishes to custom Homebrew taps
- **Homebrew-core Support**: Prepares formulas for homebrew-core submission
- **Security Scanning**: Validates URLs, checksums, and security requirements
- **Multi-architecture Support**: Handles Intel, Apple Silicon, and Linux architectures
- **Smart Platform Detection**: Automatically selects appropriate binary for user's platform

## Configuration

### Basic Configuration

```json
{
  "publishers": {
    "homebrew": {
      "enabled": true,
      "priority": 3,
      "timeout": "2m",
      "retry_count": 2,
      "config": {
        "tap_repo": "nettracex/homebrew-tap",
        "formula_name": "nettracex",
        "description": "Network diagnostic toolkit with beautiful TUI",
        "homepage": "https://github.com/nettracex/nettracex-tui",
        "license": "MIT",
        "custom_tap": true,
        "test_command": "\"--version\"",
        "dependencies": []
      }
    }
  }
}
```

### Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `tap_repo` | string | GitHub repository for custom tap | Required |
| `formula_name` | string | Name of the Homebrew formula | Required |
| `description` | string | Formula description | Required |
| `homepage` | string | Project homepage URL | Required |
| `license` | string | Software license | "MIT" |
| `custom_tap` | boolean | Use custom tap vs homebrew-core | true |
| `test_command` | string | Command to test installation | "\"--version\"" |
| `dependencies` | array | Homebrew dependencies | [] |

## Usage

### Command Line

Generate a Homebrew formula manually:

```bash
go run cmd/distribution-manager/main.go \
  --command=generate-homebrew \
  --version=v1.0.0 \
  --binary-url=https://github.com/nettracex/nettracex-tui/releases/download/v1.0.0/nettracex-darwin-amd64 \
  --output=homebrew-formula/nettracex.rb
```

### Programmatic Usage

```go
package main

import (
    "context"
    "github.com/nettracex/nettracex-tui/internal/distribution"
)

func main() {
    config := distribution.HomebrewConfig{
        TapRepo:     "myorg/homebrew-tap",
        FormulaName: "nettracex",
        Description: "Network diagnostic toolkit",
        Homepage:    "https://github.com/myorg/nettracex",
        License:     "MIT",
        CustomTap:   true,
    }

    publisher, err := distribution.NewHomebrewPublisher(config)
    if err != nil {
        panic(err)
    }

    release := distribution.Release{
        Version: "1.0.0",
        Binaries: map[string]distribution.Binary{
            "darwin-amd64": {
                Platform:    "darwin",
                Architecture: "amd64",
                DownloadURL: "https://example.com/binary",
                Checksum:    "sha256hash",
            },
        },
    }

    ctx := context.Background()
    if err := publisher.Publish(ctx, release); err != nil {
        panic(err)
    }
}
```

## Formula Structure

### Single Platform Formula

For single platform releases:

```ruby
class Nettracex < Formula
  desc "Network diagnostic toolkit with beautiful TUI"
  homepage "https://github.com/nettracex/nettracex-tui"
  url "https://github.com/nettracex/nettracex-tui/releases/download/v1.0.0/nettracex-darwin-amd64"
  sha256 "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"
  license "MIT"
  version "1.0.0"

  def install
    bin.install "nettracex" => "nettracex"
  end

  test do
    system "#{bin}/nettracex", "--version"
  end
end
```

### Multi-Platform Formula

For releases with multiple platforms (macOS and Linux):

```ruby
class Nettracex < Formula
  desc "Network diagnostic toolkit with beautiful TUI"
  homepage "https://github.com/nettracex/nettracex-tui"
  license "MIT"
  version "1.0.0"

  if OS.mac? && Hardware::CPU.intel?
    url "https://github.com/nettracex/nettracex-tui/releases/download/v1.0.0/nettracex-darwin-amd64"
    sha256 "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"
  end

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/nettracex/nettracex-tui/releases/download/v1.0.0/nettracex-darwin-arm64"
    sha256 "b2c3d4e5f6789012345678901234567890123456789012345678901234567890a1"
  end

  if OS.linux? && Hardware::CPU.intel?
    url "https://github.com/nettracex/nettracex-tui/releases/download/v1.0.0/nettracex-linux-amd64"
    sha256 "c3d4e5f6789012345678901234567890123456789012345678901234567890a1b2"
  end

  if OS.linux? && Hardware::CPU.arm?
    url "https://github.com/nettracex/nettracex-tui/releases/download/v1.0.0/nettracex-linux-arm64"
    sha256 "d4e5f6789012345678901234567890123456789012345678901234567890a1b2c3"
  end

  def install
    bin.install "nettracex" => "nettracex"
  end

  test do
    system "#{bin}/nettracex", "--version"
  end
end
```

## CI/CD Integration

### GitHub Actions Workflow

The Homebrew workflow (`.github/workflows/homebrew.yml`) provides:

1. **Formula Validation**: Validates formula syntax and structure
2. **Installation Testing**: Tests formula installation on macOS
3. **Security Scanning**: Checks for security issues
4. **Tap Submission**: Automatically submits to custom tap
5. **Documentation Updates**: Updates installation documentation

### Workflow Triggers

- **Push to main**: Validates formulas
- **Pull requests**: Tests formula changes
- **Releases**: Publishes formulas and updates taps

### Required Secrets

| Secret | Description |
|--------|-------------|
| `HOMEBREW_TAP_TOKEN` | GitHub token for tap repository access |

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HOMEBREW_TAP_REPO` | Custom tap repository | "nettracex/homebrew-tap" |
| `SUBMIT_TO_HOMEBREW_CORE` | Submit to homebrew-core | false |

## Testing

### Unit Tests

```bash
go test ./internal/distribution -run TestHomebrew
```

### Integration Tests

```bash
go test ./internal/distribution -tags=integration -run TestHomebrewIntegration
```

### Manual Testing

1. Generate a formula:
   ```bash
   make generate-homebrew VERSION=v1.0.0
   ```

2. Validate with Homebrew:
   ```bash
   brew audit --strict homebrew-formula/nettracex.rb
   ```

3. Test installation:
   ```bash
   brew install homebrew-formula/nettracex.rb
   nettracex --version
   brew uninstall nettracex
   ```

## Custom Tap Setup

### Creating a Custom Tap

1. Create a new GitHub repository: `username/homebrew-tapname`
2. Add a `Formula/` directory
3. Configure the tap repository in the publisher config

### Tap Repository Structure

```
homebrew-tap/
├── Formula/
│   └── nettracex.rb
├── README.md
└── .github/
    └── workflows/
        └── tests.yml
```

### Installing from Custom Tap

**On macOS:**
```bash
brew tap nettracex/tap
brew install nettracex
```

**On Linux:**
```bash
# Install Homebrew on Linux first (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Add Homebrew to PATH (add to ~/.bashrc or ~/.zshrc)
eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

# Install from tap
brew tap nettracex/tap
brew install nettracex
```

## Homebrew-core Submission

### Requirements

- Formula must pass all Homebrew requirements
- Software must be notable and widely used
- Must have stable releases
- Must follow Homebrew guidelines

### Submission Process

1. Generate formula with `custom_tap: false`
2. Test formula thoroughly
3. Submit PR to homebrew-core manually
4. Address reviewer feedback

### Automated Preparation

The workflow can prepare formulas for homebrew-core submission by setting `SUBMIT_TO_HOMEBREW_CORE=true`.

## Troubleshooting

### Common Issues

#### Formula Validation Fails

```bash
Error: Formula validation failed: invalid SHA256 hash length
```

**Solution**: Ensure the binary URL is accessible and checksum is correct.

#### Installation Test Fails

```bash
Error: formula installation test failed: command not found
```

**Solution**: Check that the binary name matches the formula name and test command.

#### Tap Submission Fails

```bash
Error: failed to submit formula: authentication failed
```

**Solution**: Verify `HOMEBREW_TAP_TOKEN` has write access to the tap repository.

### Debug Mode

Enable verbose logging:

```bash
go run cmd/distribution-manager/main.go --verbose --command=generate-homebrew ...
```

### Manual Formula Testing

Test formula without installing:

```bash
brew install --build-from-source homebrew-formula/nettracex.rb --dry-run
```

## Security Considerations

### URL Validation

- All URLs must use HTTPS
- Binary URLs are validated for accessibility
- SHA256 checksums are required and validated

### Dependency Management

- Dependencies are validated against Homebrew formulae
- No hardcoded secrets in formulas
- License information is required

### Access Control

- Tap repositories should use appropriate access controls
- GitHub tokens should have minimal required permissions
- Formula submissions are reviewed before merging

## Performance

### Formula Generation

- Formula generation is optimized for speed
- SHA256 calculation is done efficiently
- Network requests are cached when possible

### CI/CD Performance

- Parallel validation and testing
- Artifact caching for faster builds
- Conditional execution based on changes

## Best Practices

### Formula Design

1. Use descriptive formula names
2. Include comprehensive test blocks
3. Specify all required dependencies
4. Use semantic versioning

### Tap Management

1. Keep tap repositories organized
2. Use consistent naming conventions
3. Maintain proper documentation
4. Regular cleanup of old formulas

### Release Process

1. Test formulas before releasing
2. Update documentation with releases
3. Monitor installation success rates
4. Respond to user feedback promptly

## Examples

### Basic Formula

See `internal/distribution/homebrew_example_test.go` for complete examples.

### Advanced Configuration

```json
{
  "homebrew": {
    "config": {
      "tap_repo": "myorg/homebrew-tools",
      "formula_name": "nettracex",
      "description": "Advanced network diagnostic toolkit",
      "homepage": "https://nettracex.dev",
      "license": "Apache-2.0",
      "custom_tap": true,
      "test_command": "\"--help\"",
      "dependencies": ["openssl@3"]
    }
  }
}
```

### Multi-Architecture Support

The publisher automatically handles multiple architectures when available:

```go
release := distribution.Release{
    Binaries: map[string]distribution.Binary{
        "darwin-amd64": {...},  // Intel Macs
        "darwin-arm64": {...},  // Apple Silicon
    },
}
```

The formula will use the appropriate binary based on the target architecture.

### Hardware Detection Notes

**Important:** Homebrew's `Hardware::CPU.intel?` detects **all x86_64 processors**, including both Intel and AMD CPUs. The naming is historical but the detection works correctly:

- `Hardware::CPU.intel?` → **x86_64 architecture** (Intel AND AMD processors)
- `Hardware::CPU.arm?` → **ARM64 architecture** (Apple Silicon, ARM64 Linux)

So our `linux-amd64` binaries will work on both Intel and AMD Linux systems.