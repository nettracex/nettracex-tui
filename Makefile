# NetTraceX Makefile
# Cross-platform build system for NetTraceX network diagnostic toolkit

# Configuration
APP_NAME := nettracex
VERSION ?= dev
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
OUTPUT_DIR := bin
COMPRESS ?= false

# Build flags
LDFLAGS := -s -w
LDFLAGS += -X main.version=$(VERSION)
LDFLAGS += -X main.gitCommit=$(GIT_COMMIT)
LDFLAGS += -X main.buildTime=$(BUILD_TIME)

# Build targets
LINUX_AMD64_TARGET := $(OUTPUT_DIR)/$(APP_NAME)-linux-amd64
LINUX_ARM64_TARGET := $(OUTPUT_DIR)/$(APP_NAME)-linux-arm64
WINDOWS_AMD64_TARGET := $(OUTPUT_DIR)/$(APP_NAME)-windows-amd64.exe
DARWIN_AMD64_TARGET := $(OUTPUT_DIR)/$(APP_NAME)-darwin-amd64
DARWIN_ARM64_TARGET := $(OUTPUT_DIR)/$(APP_NAME)-darwin-arm64

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: all build test clean run fmt vet lint deps help
.PHONY: build-all build-linux build-windows build-darwin
.PHONY: build-linux-amd64 build-linux-arm64 build-windows-amd64 build-darwin-amd64 build-darwin-arm64
.PHONY: validate-build test-build compress-all generate-checksums generate-metadata
.PHONY: dev-setup init release-build

# Default target
all: fmt vet test build

# Build information
build-info:
	@echo "$(BLUE)[INFO]$(NC) NetTraceX Build Configuration"
	@echo "$(BLUE)[INFO]$(NC) ================================"
	@echo "$(BLUE)[INFO]$(NC) App Name: $(APP_NAME)"
	@echo "$(BLUE)[INFO]$(NC) Version: $(VERSION)"
	@echo "$(BLUE)[INFO]$(NC) Git Commit: $(GIT_COMMIT)"
	@echo "$(BLUE)[INFO]$(NC) Build Time: $(BUILD_TIME)"
	@echo "$(BLUE)[INFO]$(NC) Output Directory: $(OUTPUT_DIR)"
	@echo "$(BLUE)[INFO]$(NC) Compression: $(COMPRESS)"
	@echo ""

# Build the application for current platform
build:
	@echo "$(BLUE)[INFO]$(NC) Building NetTraceX for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(APP_NAME) ./
	@echo "$(GREEN)[SUCCESS]$(NC) Built $(APP_NAME) ($(shell stat -c%s $(OUTPUT_DIR)/$(APP_NAME) 2>/dev/null || stat -f%z $(OUTPUT_DIR)/$(APP_NAME) 2>/dev/null || echo unknown) bytes)"

# Cross-platform builds
build-all: build-info validate-build build-linux build-windows build-darwin generate-checksums generate-metadata
	@echo "$(GREEN)[SUCCESS]$(NC) All builds completed successfully!"
	@echo "$(BLUE)[INFO]$(NC) Generated files:"
	@ls -la $(OUTPUT_DIR)/

# Linux builds
build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	@echo "$(BLUE)[INFO]$(NC) Building for Linux AMD64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(LINUX_AMD64_TARGET) ./
	@echo "$(GREEN)[SUCCESS]$(NC) Built $(notdir $(LINUX_AMD64_TARGET)) ($(shell stat -c%s $(LINUX_AMD64_TARGET) 2>/dev/null || stat -f%z $(LINUX_AMD64_TARGET) 2>/dev/null || echo unknown) bytes)"

build-linux-arm64:
	@echo "$(BLUE)[INFO]$(NC) Building for Linux ARM64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(LINUX_ARM64_TARGET) ./
	@echo "$(GREEN)[SUCCESS]$(NC) Built $(notdir $(LINUX_ARM64_TARGET)) ($(shell stat -c%s $(LINUX_ARM64_TARGET) 2>/dev/null || stat -f%z $(LINUX_ARM64_TARGET) 2>/dev/null || echo unknown) bytes)"

# Windows builds
build-windows: build-windows-amd64

build-windows-amd64:
	@echo "$(BLUE)[INFO]$(NC) Building for Windows AMD64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(WINDOWS_AMD64_TARGET) ./
	@echo "$(GREEN)[SUCCESS]$(NC) Built $(notdir $(WINDOWS_AMD64_TARGET)) ($(shell stat -c%s $(WINDOWS_AMD64_TARGET) 2>/dev/null || stat -f%z $(WINDOWS_AMD64_TARGET) 2>/dev/null || echo unknown) bytes)"

# Darwin (macOS) builds
build-darwin: build-darwin-amd64 build-darwin-arm64

build-darwin-amd64:
	@echo "$(BLUE)[INFO]$(NC) Building for Darwin AMD64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DARWIN_AMD64_TARGET) ./
	@echo "$(GREEN)[SUCCESS]$(NC) Built $(notdir $(DARWIN_AMD64_TARGET)) ($(shell stat -c%s $(DARWIN_AMD64_TARGET) 2>/dev/null || stat -f%z $(DARWIN_AMD64_TARGET) 2>/dev/null || echo unknown) bytes)"

build-darwin-arm64:
	@echo "$(BLUE)[INFO]$(NC) Building for Darwin ARM64..."
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DARWIN_ARM64_TARGET) ./
	@echo "$(GREEN)[SUCCESS]$(NC) Built $(notdir $(DARWIN_ARM64_TARGET)) ($(shell stat -c%s $(DARWIN_ARM64_TARGET) 2>/dev/null || stat -f%z $(DARWIN_ARM64_TARGET) 2>/dev/null || echo unknown) bytes)"

# Validate build environment
validate-build:
	@echo "$(BLUE)[INFO]$(NC) Validating build environment..."
	@command -v go >/dev/null 2>&1 || { echo "$(RED)[ERROR]$(NC) Go is not installed or not in PATH"; exit 1; }
	@echo "$(BLUE)[INFO]$(NC) Go version: $(shell go version)"
	@echo "$(GREEN)[SUCCESS]$(NC) Build environment validation completed"

# Test build for current platform
test-build:
	@echo "$(BLUE)[INFO]$(NC) Testing build for current platform..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/test-build ./
	@echo "$(GREEN)[SUCCESS]$(NC) Test build successful"
	@rm -f $(OUTPUT_DIR)/test-build

# Generate checksums for all binaries
generate-checksums:
	@echo "$(BLUE)[INFO]$(NC) Generating checksums..."
	@mkdir -p $(OUTPUT_DIR)
	@> $(OUTPUT_DIR)/checksums.txt
	@for file in $(OUTPUT_DIR)/$(APP_NAME)-*; do \
		if [ -f "$$file" ]; then \
			if command -v sha256sum >/dev/null 2>&1; then \
				sha256sum "$$file" | sed 's|$(OUTPUT_DIR)/||' >> $(OUTPUT_DIR)/checksums.txt; \
			elif command -v shasum >/dev/null 2>&1; then \
				shasum -a 256 "$$file" | sed 's|$(OUTPUT_DIR)/||' >> $(OUTPUT_DIR)/checksums.txt; \
			fi; \
		fi; \
	done
	@echo "$(GREEN)[SUCCESS]$(NC) Checksums generated: $(OUTPUT_DIR)/checksums.txt"

# Generate build metadata
generate-metadata:
	@echo "$(BLUE)[INFO]$(NC) Generating build metadata..."
	@mkdir -p $(OUTPUT_DIR)
	@echo '{' > $(OUTPUT_DIR)/build-metadata.json
	@echo '  "app_name": "$(APP_NAME)",' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  "version": "$(VERSION)",' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  "git_commit": "$(GIT_COMMIT)",' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  "build_time": "$(BUILD_TIME)",' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  "go_version": "$(shell go version | awk '{print $$3}')",' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  "build_host": "$(shell uname -a)",' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  "artifacts": [' >> $(OUTPUT_DIR)/build-metadata.json
	@first=true; for file in $(OUTPUT_DIR)/$(APP_NAME)-*; do \
		if [ -f "$$file" ]; then \
			if [ "$$first" = "true" ]; then \
				first=false; \
			else \
				echo ',' >> $(OUTPUT_DIR)/build-metadata.json; \
			fi; \
			filename=$$(basename "$$file"); \
			size=$$(stat -c%s "$$file" 2>/dev/null || stat -f%z "$$file" 2>/dev/null || echo 0); \
			checksum=$$(grep "$$filename" $(OUTPUT_DIR)/checksums.txt 2>/dev/null | awk '{print $$1}' || echo "unknown"); \
			echo '    {' >> $(OUTPUT_DIR)/build-metadata.json; \
			echo '      "filename": "'$$filename'",' >> $(OUTPUT_DIR)/build-metadata.json; \
			echo '      "size": '$$size',' >> $(OUTPUT_DIR)/build-metadata.json; \
			echo '      "checksum": "'$$checksum'"' >> $(OUTPUT_DIR)/build-metadata.json; \
			echo -n '    }' >> $(OUTPUT_DIR)/build-metadata.json; \
		fi; \
	done
	@echo '' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '  ]' >> $(OUTPUT_DIR)/build-metadata.json
	@echo '}' >> $(OUTPUT_DIR)/build-metadata.json
	@echo "$(GREEN)[SUCCESS]$(NC) Build metadata generated: $(OUTPUT_DIR)/build-metadata.json"

# Compress all binaries (if compression is enabled)
compress-all:
ifeq ($(COMPRESS),true)
	@echo "$(BLUE)[INFO]$(NC) Compressing binaries..."
	@for file in $(OUTPUT_DIR)/$(APP_NAME)-*; do \
		if [ -f "$$file" ] && [ "$${file##*.}" != "gz" ] && [ "$${file##*.}" != "zip" ]; then \
			echo "$(BLUE)[INFO]$(NC) Compressing $$(basename $$file)..."; \
			if command -v tar >/dev/null 2>&1 && command -v gzip >/dev/null 2>&1; then \
				tar -czf "$$file.tar.gz" -C $(OUTPUT_DIR) "$$(basename $$file)"; \
				rm "$$file"; \
				echo "$(GREEN)[SUCCESS]$(NC) Created $$(basename $$file).tar.gz"; \
			else \
				echo "$(YELLOW)[WARNING]$(NC) tar or gzip not available, skipping compression for $$(basename $$file)"; \
			fi; \
		fi; \
	done
else
	@echo "$(BLUE)[INFO]$(NC) Compression disabled (COMPRESS=$(COMPRESS))"
endif

# Release build with compression and metadata
release-build: COMPRESS=true
release-build: build-all compress-all
	@echo "$(GREEN)[SUCCESS]$(NC) Release build completed!"

# Run tests
test:
	@echo "$(BLUE)[INFO]$(NC) Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "$(BLUE)[INFO]$(NC) Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage report generated: coverage.html"

# Run build validation tests
test-build-validation:
	@echo "$(BLUE)[INFO]$(NC) Running build validation tests..."
	go test -v ./internal/build/...

# Run the application
run:
	@echo "$(BLUE)[INFO]$(NC) Running NetTraceX..."
	go run ./

# Format code
fmt:
	@echo "$(BLUE)[INFO]$(NC) Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "$(BLUE)[INFO]$(NC) Vetting code..."
	go vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "$(BLUE)[INFO]$(NC) Linting code..."
	golangci-lint run

# Download dependencies
deps:
	@echo "$(BLUE)[INFO]$(NC) Downloading dependencies..."
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	@echo "$(BLUE)[INFO]$(NC) Cleaning build artifacts..."
	rm -rf $(OUTPUT_DIR)/
	rm -f coverage.out coverage.html coverage
	@echo "$(GREEN)[SUCCESS]$(NC) Build artifacts cleaned"

# Initialize Go module (for fresh setup)
init:
	@echo "$(BLUE)[INFO]$(NC) Initializing Go module..."
	go mod init github.com/nettracex/nettracex-tui
	go mod tidy

# Development setup
dev-setup: deps
	@echo "$(BLUE)[INFO]$(NC) Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)[SUCCESS]$(NC) Development environment setup completed"

# Build using build manager (Go-based build system)
build-manager:
	@echo "$(BLUE)[INFO]$(NC) Building using Go build manager..."
	go run ./cmd/build-manager/

# Help target
help:
	@echo "NetTraceX Build System"
	@echo "====================="
	@echo ""
	@echo "Build Targets:"
	@echo "  build              - Build for current platform"
	@echo "  build-all          - Build for all supported platforms"
	@echo "  build-linux        - Build for Linux (amd64, arm64)"
	@echo "  build-windows      - Build for Windows (amd64)"
	@echo "  build-darwin       - Build for macOS (amd64, arm64)"
	@echo "  release-build      - Build release with compression and metadata"
	@echo ""
	@echo "Platform-Specific Builds:"
	@echo "  build-linux-amd64  - Build for Linux AMD64"
	@echo "  build-linux-arm64  - Build for Linux ARM64"
	@echo "  build-windows-amd64- Build for Windows AMD64"
	@echo "  build-darwin-amd64 - Build for macOS AMD64"
	@echo "  build-darwin-arm64 - Build for macOS ARM64"
	@echo ""
	@echo "Testing:"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-build         - Test build for current platform"
	@echo "  test-build-validation - Run build validation tests"
	@echo ""
	@echo "Development:"
	@echo "  run                - Run the application"
	@echo "  fmt                - Format code"
	@echo "  vet                - Vet code"
	@echo "  lint               - Lint code (requires golangci-lint)"
	@echo "  deps               - Download dependencies"
	@echo "  dev-setup          - Set up development environment"
	@echo ""
	@echo "Utilities:"
	@echo "  validate-build     - Validate build environment"
	@echo "  generate-checksums - Generate checksums for binaries"
	@echo "  generate-metadata  - Generate build metadata"
	@echo "  compress-all       - Compress all binaries"
	@echo "  clean              - Clean build artifacts"
	@echo "  build-info         - Show build configuration"
	@echo "  help               - Show this help message"
	@echo ""
	@echo "Configuration (Environment Variables):"
	@echo "  VERSION=x.x.x      - Set version (default: dev)"
	@echo "  OUTPUT_DIR=path    - Set output directory (default: bin)"
	@echo "  COMPRESS=true      - Enable compression (default: false)"
	@echo "  GIT_COMMIT=hash    - Set git commit hash (default: auto-detect)"
	@echo ""
	@echo "Examples:"
	@echo "  make build-all                    # Build all platforms"
	@echo "  VERSION=1.0.0 make release-build # Release build with version"
	@echo "  COMPRESS=true make build-all     # Build with compression"
	@echo "  OUTPUT_DIR=dist make build-all   # Build to custom directory"
	@echo ""
	@echo "Supported Platforms:"
	@echo "  - Linux (amd64, arm64)"
	@echo "  - Windows (amd64)"
	@echo "  - macOS (amd64, arm64)"