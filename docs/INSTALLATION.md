# Installation Guide

NetTraceX can be installed in several ways:

## Option 1: Go Install (Recommended for Go users)

If you have Go installed:

```bash
go install github.com/nettracex/nettracex-tui@latest
```

Or install a specific version:

```bash
go install github.com/nettracex/nettracex-tui@v0.1.3
```

The binary will be installed as `nettracex-tui` in your `$GOPATH/bin` directory.

**Note:** Make sure `$GOPATH/bin` is in your PATH:

### Windows (PowerShell)
```powershell
$env:PATH += ";$(go env GOPATH)\bin"
```

### Linux/macOS (Bash/Zsh)
```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Option 2: Download Pre-built Binaries

Download the latest release from GitHub:

1. Go to [Releases](https://github.com/nettracex/nettracex-tui/releases)
2. Download the appropriate binary for your platform:
   - **Windows**: `nettracex_Windows_x86_64.zip`
   - **Linux**: `nettracex_Linux_x86_64.tar.gz` or `nettracex_Linux_arm64.tar.gz`
   - **macOS**: `nettracex_Darwin_x86_64.tar.gz` or `nettracex_Darwin_arm64.tar.gz`
3. Extract and run the binary

## Option 3: Build from Source

```bash
git clone https://github.com/nettracex/nettracex-tui.git
cd nettracex-tui
go build -o nettracex .
```

## Verification

After installation, verify it works:

```bash
# If installed via go install
nettracex-tui --version

# If using downloaded binary
./nettracex --version
```

## Usage

Run the interactive TUI:

```bash
nettracex-tui
```

Or show help:

```bash
nettracex-tui --help
```

## Updating

### Go Install
```bash
go install github.com/nettracex/nettracex-tui@latest
```

### Pre-built Binaries
Download the latest release and replace your existing binary.

## Troubleshooting

### Command not found
- Ensure `$GOPATH/bin` is in your PATH
- On Windows, you may need to restart your terminal after adding to PATH

### Permission denied (Linux/macOS)
```bash
chmod +x nettracex
```

### Antivirus false positive (Windows)
Some antivirus software may flag Go binaries. Add an exception for the binary if needed.