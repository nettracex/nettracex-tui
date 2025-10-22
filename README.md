# NetTraceX TUI

A comprehensive network diagnostic toolkit built with Go, featuring a beautiful terminal user interface powered by the Bubble Tea framework.

## Project Structure

```
nettracex-tui/
├── internal/
│   ├── domain/           # Core domain interfaces and types
│   │   ├── interfaces.go # Core business logic interfaces
│   │   ├── types.go      # Domain types and value objects
│   │   ├── parameters.go # Parameter implementations
│   │   ├── result.go     # Result implementations
│   │   └── *_test.go     # Comprehensive unit tests
│   └── config/           # Configuration management
│       ├── config.go     # Configuration manager implementation
│       └── config_test.go# Configuration tests
├── go.mod               # Go module definition
├── main.go              # Application entry point
├── Makefile             # Build and development commands
└── README.md            # Project documentation
```

## Architecture

The project follows clean architecture principles with clear separation of concerns:

- **Domain Layer**: Core business logic interfaces and types
- **Application Layer**: Use cases and application services (to be implemented)
- **Infrastructure Layer**: Network clients and external integrations (to be implemented)
- **Presentation Layer**: TUI components and user interface (to be implemented)

### SOLID Principles Implementation

- **Single Responsibility**: Each interface and type has a single, well-defined purpose
- **Open/Closed**: Plugin architecture allows extension without modification
- **Liskov Substitution**: All implementations can be substituted for their interfaces
- **Interface Segregation**: Small, focused interfaces for different concerns
- **Dependency Inversion**: High-level modules depend on abstractions

## Core Interfaces

### DiagnosticTool
Defines the contract for all network diagnostic tools (ping, traceroute, DNS, WHOIS, SSL).

### NetworkClient
Abstracts network operations for testing and flexibility.

### Result
Represents diagnostic operation results with formatting and export capabilities.

### ConfigurationManager
Handles application configuration with validation.

## Configuration

The application uses a hierarchical configuration system:

1. Default values
2. Configuration file (`~/.config/nettracex/nettracex.yaml`)
3. Environment variables (prefixed with `NETTRACEX_`)

### Configuration Sections

- **Network**: Timeout, DNS servers, retry settings
- **UI**: Theme, key bindings, animation settings
- **Plugins**: Enabled/disabled plugins and settings
- **Export**: Default format and output settings
- **Logging**: Log level, format, and output settings

## Development

### Prerequisites

- Go 1.21 or later
- Make (for Unix/Linux/macOS) or PowerShell (for Windows)

### Setup

#### Unix/Linux/macOS (with Make)

```bash
# Clone the repository
git clone <repository-url>
cd nettracex-tui

# Download dependencies
make deps

# Run tests
make test

# Build the application
make build

# Run the application
make run
```

#### Windows (PowerShell)

```powershell
# Clone the repository
git clone <repository-url>
cd nettracex-tui

# Download dependencies
go mod download
go mod tidy

# Run tests
go test ./...

# Build the application
go build -o bin/nettracex-tui.exe ./cmd/nettracex-tui

# Run the application
./bin/nettracex-tui.exe

# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Clean build artifacts
Remove-Item -Recurse -Force bin/ -ErrorAction SilentlyContinue
```

### Available Commands

#### Make Targets (Unix/Linux/macOS)

- `make build` - Build the application
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make run` - Run the application
- `make fmt` - Format code
- `make vet` - Vet code for issues
- `make lint` - Lint code (requires golangci-lint)
- `make clean` - Clean build artifacts
- `make build-all` - Cross-platform builds

#### PowerShell Commands (Windows)

- `go build -o bin/nettracex-tui.exe ./cmd/nettracex-tui` - Build the application
- `go test ./...` - Run all tests
- `go test -coverprofile=coverage.out ./...` - Run tests with coverage
- `go fmt ./...` - Format code
- `go vet ./...` - Vet code for issues
- `Remove-Item -Recurse -Force bin/` - Clean build artifacts

### Testing

The project includes comprehensive unit tests for all core components:

#### Unix/Linux/macOS

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test packages
go test ./internal/domain
go test ./internal/tui
go test ./internal/tools/whois
```

#### Windows (PowerShell)

```powershell
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test packages
go test ./internal/domain
go test ./internal/tui
go test ./internal/tools/whois

# Run tests with verbose output
go test -v ./...

# Run tests multiple times to check for flaky tests
go test -count=3 ./...
```

### Windows Development Notes

- **PowerShell**: Use PowerShell instead of Command Prompt for better Go development experience
- **Path Separators**: Go handles path separators automatically, but use forward slashes in Go code
- **Build Output**: Windows executables have `.exe` extension automatically added
- **Environment Variables**: Use `$env:VARIABLE_NAME` syntax in PowerShell for environment variables
- **Make Alternative**: If you prefer Make on Windows, install it via [Chocolatey](https://chocolatey.org/): `choco install make`

### IDE Setup

#### Visual Studio Code (Recommended for Windows)

1. Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go)
2. Install Go tools when prompted, or run: `Ctrl+Shift+P` → "Go: Install/Update Tools"
3. Configure settings in `.vscode/settings.json`:

```json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "go.coverOnSave": true
}
```

#### GoLand/IntelliJ IDEA

1. Install the Go plugin
2. Configure Go SDK path in Settings → Languages & Frameworks → Go
3. Enable Go modules support
4. Configure code style and inspections

#### Command Line Tools

```powershell
# Install useful Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/go-delve/delve/cmd/dlv@latest
```

### Quick Reference for Windows Developers

#### Common Development Tasks

```powershell
# Full development cycle
go mod tidy                                    # Update dependencies
go fmt ./...                                   # Format all code
go vet ./...                                   # Check for issues
go test ./...                                  # Run all tests
go build -o bin/nettracex-tui.exe ./cmd/nettracex-tui  # Build application

# Testing specific components
go test ./internal/tui -v                     # Test TUI components
go test ./internal/tools/whois -v             # Test WHOIS tool
go test ./internal/domain -v                  # Test domain layer

# Development with file watching (requires external tool)
# Install: go install github.com/cosmtrek/air@latest
air  # Auto-rebuild on file changes

# Debugging
dlv debug ./cmd/nettracex-tui                 # Debug with Delve
```

#### Environment Setup

```powershell
# Set Go environment variables (if needed)
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

# Check Go environment
go env

# Update Go modules
go get -u ./...
go mod tidy
```

## Current Implementation Status

✅ **Task 1: Project Foundation and Core Interfaces** (COMPLETED)
- Go module setup with proper dependencies
- Core domain interfaces following SOLID principles
- Configuration system with validation
- Comprehensive unit tests for all interfaces and types

### What's Implemented

1. **Core Interfaces**: DiagnosticTool, NetworkClient, Result, Parameters, and supporting interfaces
2. **Domain Types**: NetworkHost, PingResult, TraceHop, DNSResult, WHOISResult, SSLResult, and configuration types
3. **Parameter System**: Type-safe parameter handling for all diagnostic operations
4. **Result System**: Flexible result formatting and export (JSON, CSV, Text)
5. **Configuration Management**: Hierarchical configuration with validation
6. **Comprehensive Testing**: 100% test coverage for all implemented components

### Next Steps

The foundation is now ready for implementing the remaining tasks:
- Network client infrastructure with mocking
- Individual diagnostic tool implementations
- Bubble Tea TUI framework setup
- Plugin registry and management system

## License

[License information to be added]