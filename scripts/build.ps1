# NetTraceX Cross-Platform Build Script for Windows PowerShell
# This script builds NetTraceX for multiple platforms and architectures

param(
    [Parameter(Position=0)]
    [string]$Command = "all",
    
    [string]$Version = "dev",
    [string]$OutputDir = "bin",
    [string]$GitCommit = "",
    [switch]$Compress = $false,
    [switch]$Help = $false
)

# Configuration
$AppName = "nettracex"
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

# Colors for output (Windows PowerShell compatible)
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Print build information
function Show-BuildInfo {
    Write-Info "NetTraceX Build Script"
    Write-Info "======================"
    Write-Info "App Name: $AppName"
    Write-Info "Version: $Version"
    Write-Info "Git Commit: $GitCommit"
    Write-Info "Build Time: $BuildTime"
    Write-Info "Output Directory: $OutputDir"
    Write-Info "Compression: $Compress"
    Write-Host ""
}

# Check prerequisites
function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check if Go is installed
    try {
        $goVersion = & go version 2>$null
        if ($LASTEXITCODE -ne 0) {
            throw "Go command failed"
        }
        Write-Info "Go version: $goVersion"
    }
    catch {
        Write-Error "Go is not installed or not in PATH"
        exit 1
    }
    
    # Check if git is available for commit hash
    if ([string]::IsNullOrEmpty($GitCommit)) {
        try {
            $GitCommit = & git rev-parse --short HEAD 2>$null
            if ($LASTEXITCODE -ne 0) {
                throw "Git command failed"
            }
            Write-Info "Git commit: $GitCommit"
        }
        catch {
            Write-Warning "Git not available or not in a git repository, using 'unknown' for commit hash"
            $script:GitCommit = "unknown"
        }
    }
    
    Write-Success "Prerequisites check completed"
    Write-Host ""
}

# Build for a specific platform
function Build-Platform {
    param(
        [string]$OS,
        [string]$Arch,
        [string]$OutputName,
        [string]$Extension
    )
    
    Write-Info "Building for $OS/$Arch..."
    
    # Set environment variables
    $env:GOOS = $OS
    $env:GOARCH = $Arch
    $env:CGO_ENABLED = "0"
    
    # Prepare output path
    $outputPath = Join-Path $OutputDir "$OutputName$Extension"
    
    # Prepare ldflags
    $ldflags = "-s -w"
    $ldflags += " -X main.version=$Version"
    $ldflags += " -X main.gitCommit=$GitCommit"
    $ldflags += " -X main.buildTime=$BuildTime"
    
    # Build the binary
    try {
        & go build -ldflags $ldflags -o $outputPath ./
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed"
        }
        
        # Get file size
        $fileInfo = Get-Item $outputPath
        $fileSize = $fileInfo.Length
        
        # Calculate checksum
        $checksum = (Get-FileHash -Path $outputPath -Algorithm SHA256).Hash.ToLower()
        
        Write-Success "Built $OutputName$Extension ($fileSize bytes, checksum: $($checksum.Substring(0,8))...)"
        
        # Compress if requested
        if ($Compress) {
            Compress-Binary -BinaryPath $outputPath -OS $OS
        }
        
        # Store build info
        "$checksum  $(Split-Path $outputPath -Leaf)" | Add-Content -Path (Join-Path $OutputDir "checksums.txt")
        
        return $true
    }
    catch {
        Write-Error "Failed to build for $OS/$Arch`: $_"
        return $false
    }
}

# Compress binary
function Compress-Binary {
    param(
        [string]$BinaryPath,
        [string]$OS
    )
    
    Write-Info "Compressing $(Split-Path $BinaryPath -Leaf)..."
    
    try {
        # Create zip archive
        $archiveName = "$BinaryPath.zip"
        Compress-Archive -Path $BinaryPath -DestinationPath $archiveName -Force
        
        # Remove original binary
        Remove-Item $BinaryPath -Force
        
        Write-Success "Created compressed archive: $(Split-Path $archiveName -Leaf)"
    }
    catch {
        Write-Warning "Failed to compress binary: $_"
    }
}

# Build all platforms
function Build-All {
    Write-Info "Building for all platforms..."
    Write-Host ""
    
    # Create output directory
    if (!(Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    }
    
    # Clear previous checksums
    $checksumsPath = Join-Path $OutputDir "checksums.txt"
    if (Test-Path $checksumsPath) {
        Remove-Item $checksumsPath -Force
    }
    
    # Define build targets
    $targets = @(
        @{ OS = "linux"; Arch = "amd64"; OutputName = "nettracex-linux-amd64"; Extension = "" },
        @{ OS = "linux"; Arch = "arm64"; OutputName = "nettracex-linux-arm64"; Extension = "" },
        @{ OS = "windows"; Arch = "amd64"; OutputName = "nettracex-windows-amd64"; Extension = ".exe" },
        @{ OS = "darwin"; Arch = "amd64"; OutputName = "nettracex-darwin-amd64"; Extension = "" },
        @{ OS = "darwin"; Arch = "arm64"; OutputName = "nettracex-darwin-arm64"; Extension = "" }
    )
    
    $successCount = 0
    $totalCount = $targets.Count
    
    # Build each target
    foreach ($target in $targets) {
        if (Build-Platform -OS $target.OS -Arch $target.Arch -OutputName $target.OutputName -Extension $target.Extension) {
            $successCount++
        }
        Write-Host ""
    }
    
    # Generate build metadata
    New-BuildMetadata
    
    # Print summary
    Write-Info "Build Summary"
    Write-Info "============="
    Write-Info "Successful builds: $successCount/$totalCount"
    
    if ($successCount -eq $totalCount) {
        Write-Success "All builds completed successfully!"
    } else {
        Write-Warning "Some builds failed. Check the output above for details."
    }
    
    # List generated files
    Write-Info "Generated files:"
    Get-ChildItem $OutputDir | Format-Table Name, Length, LastWriteTime -AutoSize
}

# Generate build metadata
function New-BuildMetadata {
    $metadataFile = Join-Path $OutputDir "build-metadata.json"
    
    Write-Info "Generating build metadata..."
    
    $goVersion = & go version
    $buildHost = "$env:COMPUTERNAME ($env:OS)"
    
    $metadata = @{
        app_name = $AppName
        version = $Version
        git_commit = $GitCommit
        build_time = $BuildTime
        go_version = $goVersion
        build_host = $buildHost
        artifacts = @()
    }
    
    # Add artifact information
    $files = Get-ChildItem $OutputDir -File | Where-Object { 
        $_.Name -ne "checksums.txt" -and $_.Name -ne "build-metadata.json" 
    }
    
    foreach ($file in $files) {
        $checksum = "unknown"
        $checksumContent = Get-Content (Join-Path $OutputDir "checksums.txt") -ErrorAction SilentlyContinue
        if ($checksumContent) {
            $checksumLine = $checksumContent | Where-Object { $_ -match [regex]::Escape($file.Name) }
            if ($checksumLine) {
                $checksum = ($checksumLine -split '\s+')[0]
            }
        }
        
        $artifact = @{
            filename = $file.Name
            size = $file.Length
            checksum = $checksum
        }
        
        $metadata.artifacts += $artifact
    }
    
    # Convert to JSON and save
    $metadata | ConvertTo-Json -Depth 3 | Set-Content $metadataFile -Encoding UTF8
    
    Write-Success "Build metadata generated: $metadataFile"
}

# Clean build artifacts
function Remove-BuildArtifacts {
    Write-Info "Cleaning build artifacts..."
    
    if (Test-Path $OutputDir) {
        Remove-Item $OutputDir -Recurse -Force
        Write-Success "Cleaned $OutputDir directory"
    } else {
        Write-Info "No build artifacts to clean"
    }
}

# Validate build environment
function Test-BuildEnvironment {
    Write-Info "Validating build environment..."
    
    Test-Prerequisites
    
    # Test build for current platform
    Write-Info "Testing build for current platform..."
    $testOutput = Join-Path $OutputDir "test-build.exe"
    
    if (!(Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    }
    
    try {
        & go build -o $testOutput ./
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed"
        }
        Write-Success "Test build successful"
        Remove-Item $testOutput -Force -ErrorAction SilentlyContinue
    }
    catch {
        Write-Error "Test build failed: $_"
        exit 1
    }
    
    Write-Success "Build environment validation completed"
}

# Show help
function Show-Help {
    @"
NetTraceX Build Script for Windows PowerShell

Usage: .\build.ps1 [COMMAND] [OPTIONS]

Commands:
    all         Build for all supported platforms (default)
    clean       Clean build artifacts
    validate    Validate build environment
    help        Show this help message

Options:
    -Version x.x.x      Set version (default: dev)
    -OutputDir path     Set output directory (default: bin)
    -Compress           Enable compression
    -GitCommit hash     Set git commit hash (default: auto-detect)

Examples:
    .\build.ps1 all                                 # Build all platforms
    .\build.ps1 all -Version "1.0.0"              # Build with specific version
    .\build.ps1 all -Compress                      # Build with compression
    .\build.ps1 all -OutputDir "dist"             # Build to custom directory
    .\build.ps1 clean                              # Clean build artifacts
    .\build.ps1 validate                           # Validate environment

Supported Platforms:
    - Linux (amd64, arm64)
    - Windows (amd64)
    - macOS (amd64, arm64)

"@
}

# Main script logic
function Main {
    if ($Help) {
        Show-Help
        return
    }
    
    switch ($Command.ToLower()) {
        "all" {
            Show-BuildInfo
            Test-Prerequisites
            Build-All
        }
        "clean" {
            Remove-BuildArtifacts
        }
        "validate" {
            Test-BuildEnvironment
        }
        "help" {
            Show-Help
        }
        default {
            Write-Error "Unknown command: $Command"
            Write-Host ""
            Show-Help
            exit 1
        }
    }
}

# Run main function
Main