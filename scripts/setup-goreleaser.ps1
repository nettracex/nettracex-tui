# Setup script for GoReleaser on Windows
param(
    [switch]$Force
)

Write-Host "Setting up GoReleaser for NetTraceX..." -ForegroundColor Green

# Check if Go is installed
Write-Host "Checking Go installation..." -ForegroundColor Yellow
try {
    $goVersion = go version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Go is installed: $goVersion" -ForegroundColor Green
    } else {
        throw "Go command failed"
    }
} catch {
    Write-Host "Go is not installed. Please install Go first." -ForegroundColor Red
    exit 1
}

# Install GoReleaser
Write-Host "Installing GoReleaser..." -ForegroundColor Yellow
try {
    $output = go install github.com/goreleaser/goreleaser/v2@latest 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "GoReleaser installation command executed" -ForegroundColor Green
    } else {
        throw "Installation failed: $output"
    }
} catch {
    Write-Host "GoReleaser installation failed: $_" -ForegroundColor Red
    exit 1
}

# Verify installation
Write-Host "Verifying GoReleaser installation..." -ForegroundColor Yellow
try {
    $goreleaserVersion = goreleaser --version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "GoReleaser installed: $goreleaserVersion" -ForegroundColor Green
    } else {
        throw "GoReleaser not found"
    }
} catch {
    Write-Host "GoReleaser not found in PATH. Make sure GOPATH/bin is in your PATH." -ForegroundColor Red
    $goPath = go env GOPATH 2>$null
    if ($goPath) {
        Write-Host "Try adding $goPath\bin to your PATH environment variable" -ForegroundColor Yellow
    }
    exit 1
}

# Check configuration
Write-Host "Checking GoReleaser configuration..." -ForegroundColor Yellow
try {
    $checkOutput = goreleaser check 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "GoReleaser configuration is valid" -ForegroundColor Green
    } else {
        throw "Configuration check failed: $checkOutput"
    }
} catch {
    Write-Host "GoReleaser configuration has issues: $_" -ForegroundColor Red
    exit 1
}

# Test build
Write-Host "Testing GoReleaser build..." -ForegroundColor Yellow
try {
    $buildOutput = goreleaser build --snapshot --clean 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "GoReleaser test build successful" -ForegroundColor Green
        Write-Host "Build artifacts are in the dist directory" -ForegroundColor Cyan
    } else {
        throw "Build failed: $buildOutput"
    }
} catch {
    Write-Host "GoReleaser test build failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "GoReleaser setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Features:" -ForegroundColor Cyan
Write-Host "- Multi-platform builds (Linux, Windows, macOS - AMD64/ARM64)"
Write-Host "- Proper version detection from Git tags"
Write-Host "- No 'next' suffix in snapshot builds"
Write-Host "- Automatic GitHub releases on tag push"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "1. Create a GitHub release tag:"
Write-Host "   git tag v1.0.0"
Write-Host "   git push origin v1.0.0"
Write-Host "2. The GitHub Action will automatically build and release"
Write-Host "3. Or run locally: goreleaser release --snapshot --clean"
Write-Host ""
Write-Host "Installation options:" -ForegroundColor Cyan
Write-Host "- go install github.com/nettracex/nettracex-tui@latest"
Write-Host "- Download binaries from GitHub releases"
Write-Host "- Build from source with proper version detection"
Write-Host ""
Write-Host "Useful commands:" -ForegroundColor Cyan
Write-Host "- make goreleaser-check     # Check configuration"
Write-Host "- make goreleaser-build     # Test build"
Write-Host "- make goreleaser-release-dry # Dry run release"