# octap

[![Test](https://github.com/m-mizutani/octap/actions/workflows/test.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/test.yml)
[![Security](https://github.com/m-mizutani/octap/actions/workflows/security.yml/badge.svg)](https://github.com/m-mizutani/octap/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/m-mizutani/octap)](https://goreportcard.com/report/github.com/m-mizutani/octap)

CLI GitHub Actions notifier - Monitor and notify when GitHub Actions workflows complete.

## Features

- 🔄 **Real-time monitoring** of GitHub Actions workflows
- 🎯 **Commit-specific tracking** - Monitor workflows for specific commits
- 🔔 **Sound notifications** - Different sounds for success/failure
- 📊 **Live CUI display** - See workflow status in real-time
- ⏱️ **Configurable polling** - Adjust check intervals
- 🔐 **Secure authentication** - GitHub personal access token support

## Installation

### Using Go

```bash
go install github.com/m-mizutani/octap@latest
```

### From source

```bash
git clone https://github.com/m-mizutani/octap.git
cd octap
go build -o octap .
```

## Usage

### Basic usage

Monitor GitHub Actions for the current commit in the current directory:

```bash
octap
```

### Monitor specific commit

```bash
octap -c abc123def
```

### Adjust polling interval

```bash
# Check every 30 seconds
octap -i 30s
```

### Disable sound notifications

```bash
octap --silent
```

### Debug mode

```bash
octap --debug
```

## Authentication

octap uses GitHub OAuth Device Flow for authentication. On first run:

1. You'll receive a code to copy
2. Visit the provided GitHub URL
3. Paste the code and authorize the app
4. octap will automatically complete the authentication

The token is stored locally at `~/.config/octap/token.json`.

## Configuration

### Command-line flags

- `-c, --commit`: Specify commit SHA to monitor
- `-i, --interval`: Polling interval (default: 15s)
- `--config`: Config file path
- `--silent`: Disable sound notifications
- `--verbose`: Enable verbose logging
- `--debug`: Enable debug logging

## Development

### Prerequisites

- Go 1.23+
- golangci-lint (for linting)
- gosec (for security checks)

### Building

```bash
go build -o octap .
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run ./...
gosec -quiet ./...
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

Masayuki Mizutani ([@m-mizutani](https://github.com/m-mizutani))