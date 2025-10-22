#!/bin/bash
# NetTraceX Cross-Platform Build Script for Unix/Linux systems
# This script builds NetTraceX for multiple platforms and architectures

set -e  # Exit on any error

# Configuration
APP_NAME="nettracex"
VERSION="${VERSION:-dev}"
OUTPUT_DIR="${OUTPUT_DIR:-bin}"
GIT_COMMIT="${GIT_COMMIT:-}"
COMPRESS="${COMPRESS:-false}"
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Colors for output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show build information
show_build_info() {
    print_info "NetTraceX Build Script"
    print_info "======================"
    print_info "App Name: $APP_NAME"
    print_info "Version: $VERSION"
    print_info "Git Commit: $GIT_COMMIT"
    print_info "Build Time: $BUILD_TIME"
    print_info "Output Directory: $OUTPUT_DIR"
    print_info "Compression: $COMPRESS"
    echo ""
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    print_info "Go version: $(go version)"
    
    # Check if git is available for commit hash
    if [ -z "$GIT_COMMIT" ]; then
        if command -v git &> /dev/null && git rev-parse --git-dir &> /dev/null; then
            GIT_COMMIT=$(git rev-parse --short HEAD)
            print_info "Git commit: $GIT_COMMIT"
        else
            print_warning "Git not available or not in a git repository, using 'unknown' for commit hash"
            GIT_COMMIT="unknown"
        fi
    fi
    
    print_success "Prerequisites check completed"
    echo ""
}

# Build for a specific platform
build_platform() {
    local os=$1
    local arch=$2
    local output_name=$3
    local extension=$4
    
    print_info "Building for $os/$arch..."
    
    # Set environment variables
    export GOOS=$os
    export GOARCH=$arch
    export CGO_ENABLED=0
    
    # Prepare output path
    local output_path="$OUTPUT_DIR/$output_name$extension"
    
    # Prepare ldflags
    local ldflags="-s -w"
    ldflags="$ldflags -X main.version=$VERSION"
    ldflags="$ldflags -X main.gitCommit=$GIT_COMMIT"
    ldflags="$ldflags -X main.buildTime=$BUILD_TIME"
    
    # Build the binary
    if go build -ldflags "$ldflags" -o "$output_path" ./; then
        # Get file size
        local file_size=$(stat -c%s "$output_path" 2>/dev/null || stat -f%z "$output_path" 2>/dev/null || echo "unknown")
        
        # Calculate checksum
        local checksum
        if command -v sha256sum &> /dev/null; then
            checksum=$(sha256sum "$output_path" | awk '{print $1}')
        elif command -v shasum &> /dev/null; then
            checksum=$(shasum -a 256 "$output_path" | awk '{print $1}')
        else
            checksum="unknown"
        fi
        
        print_success "Built $output_name$extension ($file_size bytes, checksum: ${checksum:0:8}...)"
        
        # Store checksum
        echo "$checksum  $(basename "$output_path")" >> "$OUTPUT_DIR/checksums.txt"
        
        # Compress if requested
        if [ "$COMPRESS" = "true" ]; then
            compress_binary "$output_path" "$os"
        fi
        
        return 0
    else
        print_error "Failed to build for $os/$arch"
        return 1
    fi
}

# Compress binary
compress_binary() {
    local binary_path=$1
    local os=$2
    
    print_info "Compressing $(basename "$binary_path")..."
    
    if command -v tar &> /dev/null && command -v gzip &> /dev/null; then
        local archive_name="$binary_path.tar.gz"
        tar -czf "$archive_name" -C "$OUTPUT_DIR" "$(basename "$binary_path")"
        rm "$binary_path"
        print_success "Created compressed archive: $(basename "$archive_name")"
    else
        print_warning "tar or gzip not available, skipping compression for $(basename "$binary_path")"
    fi
}

# Build all platforms
build_all() {
    print_info "Building for all platforms..."
    echo ""
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    # Clear previous checksums
    > "$OUTPUT_DIR/checksums.txt"
    
    # Define build targets
    local targets=(
        "linux amd64 nettracex-linux-amd64 ''"
        "linux arm64 nettracex-linux-arm64 ''"
        "windows amd64 nettracex-windows-amd64 .exe"
        "darwin amd64 nettracex-darwin-amd64 ''"
        "darwin arm64 nettracex-darwin-arm64 ''"
    )
    
    local success_count=0
    local total_count=${#targets[@]}
    
    # Build each target
    for target in "${targets[@]}"; do
        # Parse target string
        read -r os arch output_name extension <<< "$target"
        
        if build_platform "$os" "$arch" "$output_name" "$extension"; then
            ((success_count++))
        fi
        echo ""
    done
    
    # Generate build metadata
    generate_build_metadata
    
    # Print summary
    print_info "Build Summary"
    print_info "============="
    print_info "Successful builds: $success_count/$total_count"
    
    if [ $success_count -eq $total_count ]; then
        print_success "All builds completed successfully!"
    else
        print_warning "Some builds failed. Check the output above for details."
    fi
    
    # List generated files
    print_info "Generated files:"
    ls -la "$OUTPUT_DIR/"
}

# Generate build metadata
generate_build_metadata() {
    local metadata_file="$OUTPUT_DIR/build-metadata.json"
    
    print_info "Generating build metadata..."
    
    local go_version=$(go version)
    local build_host=$(uname -a)
    
    cat > "$metadata_file" << EOF
{
  "app_name": "$APP_NAME",
  "version": "$VERSION",
  "git_commit": "$GIT_COMMIT",
  "build_time": "$BUILD_TIME",
  "go_version": "$go_version",
  "build_host": "$build_host",
  "artifacts": [
EOF

    # Add artifact information
    local first=true
    for file in "$OUTPUT_DIR"/$APP_NAME-*; do
        if [ -f "$file" ] && [[ ! "$file" =~ \.(txt|json)$ ]]; then
            if [ "$first" = "true" ]; then
                first=false
            else
                echo "," >> "$metadata_file"
            fi
            
            local filename=$(basename "$file")
            local size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
            local checksum="unknown"
            
            if [ -f "$OUTPUT_DIR/checksums.txt" ]; then
                checksum=$(grep "$filename" "$OUTPUT_DIR/checksums.txt" | awk '{print $1}' || echo "unknown")
            fi
            
            cat >> "$metadata_file" << EOF
    {
      "filename": "$filename",
      "size": $size,
      "checksum": "$checksum"
    }
EOF
        fi
    done
    
    cat >> "$metadata_file" << EOF

  ]
}
EOF
    
    print_success "Build metadata generated: $metadata_file"
}

# Clean build artifacts
clean_artifacts() {
    print_info "Cleaning build artifacts..."
    
    if [ -d "$OUTPUT_DIR" ]; then
        rm -rf "$OUTPUT_DIR"
        print_success "Cleaned $OUTPUT_DIR directory"
    else
        print_info "No build artifacts to clean"
    fi
}

# Validate build environment
validate_environment() {
    print_info "Validating build environment..."
    
    check_prerequisites
    
    # Test build for current platform
    print_info "Testing build for current platform..."
    local test_output="$OUTPUT_DIR/test-build"
    
    mkdir -p "$OUTPUT_DIR"
    
    if go build -o "$test_output" ./; then
        print_success "Test build successful"
        rm -f "$test_output"
    else
        print_error "Test build failed"
        exit 1
    fi
    
    print_success "Build environment validation completed"
}

# Show help
show_help() {
    cat << EOF
NetTraceX Build Script for Unix/Linux Systems

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    all         Build for all supported platforms (default)
    clean       Clean build artifacts
    validate    Validate build environment
    help        Show this help message

Environment Variables:
    VERSION=x.x.x      Set version (default: dev)
    OUTPUT_DIR=path    Set output directory (default: bin)
    COMPRESS=true      Enable compression (default: false)
    GIT_COMMIT=hash    Set git commit hash (default: auto-detect)

Examples:
    $0 all                                 # Build all platforms
    VERSION=1.0.0 $0 all                  # Build with specific version
    COMPRESS=true $0 all                  # Build with compression
    OUTPUT_DIR=dist $0 all                # Build to custom directory
    $0 clean                              # Clean build artifacts
    $0 validate                           # Validate environment

Supported Platforms:
    - Linux (amd64, arm64)
    - Windows (amd64)
    - macOS (amd64, arm64)

EOF
}

# Main script logic
main() {
    local command=${1:-all}
    
    case $command in
        "all")
            show_build_info
            check_prerequisites
            build_all
            ;;
        "clean")
            clean_artifacts
            ;;
        "validate")
            validate_environment
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"