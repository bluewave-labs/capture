![github-license](https://img.shields.io/github/license/bluewave-labs/capture)
![github-repo-size](https://img.shields.io/github/repo-size/bluewave-labs/capture)
![github-commit-activity](https://img.shields.io/github/commit-activity/w/bluewave-labs/capture)
![github-last-commit-data](https://img.shields.io/github/last-commit/bluewave-labs/capture)
![github-languages](https://img.shields.io/github/languages/top/bluewave-labs/capture)
![github-issues-and-prs](https://img.shields.io/github/issues-pr/bluewave-labs/capture)
![github-issues](https://img.shields.io/github/issues/bluewave-labs/capture)
[![go-reference](https://pkg.go.dev/badge/github.com/bluewave-labs/capture.svg)](https://pkg.go.dev/github.com/bluewave-labs/capture)
[![github-actions-check](https://github.com/bluewave-labs/capture/actions/workflows/check.yml/badge.svg)](https://github.com/bluewave-labs/capture/actions/workflows/check.yml)
[![github-actions-go](https://github.com/bluewave-labs/capture/actions/workflows/go.yml/badge.svg)](https://github.com/bluewave-labs/capture/actions/workflows/go.yml)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/bluewave-labs/capture)

<h1 align="center"><a href="https://bluewavelabs.ca" target="_blank">Capture</a></h1>

<p align="center"><strong>An open source hardware monitoring agent</strong></p>

Capture is a hardware monitoring agent that collects hardware information from the host machine and exposes it through a RESTful API. The agent is designed to be lightweight and easy to use.

## Features

- CPU Monitoring
  - Temperature
  - Load
  - Frequency
  - Usage
- Memory Monitoring
- Disk Monitoring
  - Usage
  - Inode Usage
  - Read/Write Bytes
- S.M.A.R.T. Monitoring (Self-Monitoring, Analysis and Reporting Technology)
- Network Monitoring
- Docker Container Monitoring
- GPU Monitoring (coming soon)

## Quick Start (Docker)

```shell
docker run -d \
    -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=your-secret-key \
    ghcr.io/bluewave-labs/capture:latest
```

## Quick Start (Docker Compose)

```yaml
services:
  # Capture service
  capture:
    image: ghcr.io/bluewave-labs/capture:latest
    container_name: capture
    restart: unless-stopped
    ports:
      - "59232:59232"
    environment:
      - API_SECRET=REPLACE_WITH_YOUR_SECRET # Required authentication key. Do not forget to replace this with your actual secret key.
      - GIN_MODE=release
    volumes:
      - /etc/os-release:/etc/os-release:ro
```

## Configuration

Capture supports flexible configuration through YAML files, environment variables, or command-line flags. Configuration files are loaded using [Viper](https://github.com/spf13/viper) with automatic environment variable binding.

### Quick Configuration Setup

1. **Generate a configuration file:**
   ```shell
   ./capture --generate-config capture.yaml
   ```

2. **Edit the configuration file:**
   ```yaml
   server:
     api_secret: "your-secure-secret-here"  # Set your API secret
     port: "59232"
   ```

3. **Run with configuration file:**
   ```shell
   ./capture --config capture.yaml
   ```

### Configuration Methods

#### Method 1: Configuration File (Recommended)
```yaml
version: 1

server:
  port: "59232"
  api_secret: "your-secure-secret-here"

# Optional: Configure targets for data forwarding
targets:
  - name: "My Checkmate Instance"
    endpoint: "https://checkmate.example.com/api/v1/metrics"
    api_secret: "checkmate-api-secret"

# Optional: External monitoring plugins
plugins:
  - name: "custom-health-check"
    command: "/usr/local/bin/health-check.sh"

log_level: "info"
```

#### Method 2: Environment Variables
```shell
# Basic configuration
export API_SECRET=your-secret-key
export PORT=59232
./capture

# Advanced configuration with Viper naming
export CAPTURE_SERVER_API_SECRET=your-secret-key
export CAPTURE_SERVER_PORT=59232
export CAPTURE_LOG_LEVEL=debug
./capture
```

#### Method 3: Command Line Flags
```shell
./capture --config /path/to/config.yaml
./capture --show-config                    # Show current configuration
./capture --validate-config config.yaml    # Validate configuration file
```

### Configuration Options

| Variable/Config | Environment Variable | Description | Default | Required |
|-----------------|---------------------|-------------|---------|----------|
| `server.api_secret` | `API_SECRET` or `CAPTURE_SERVER_API_SECRET` | Authentication key ([Must match Checkmate secret](https://docs.checkmate.so/users-guide/infrastructure-monitor#step-2-configure-general-settings)) | - | Yes |
| `server.port` | `PORT` or `CAPTURE_SERVER_PORT` | Server port number | 59232 | No |
| `log_level` | `CAPTURE_LOG_LEVEL` | Logging level (error/warn/info/debug) | info | No |
| `targets` | - | Checkmate instances for data forwarding | [] | No |
| `plugins` | - | External monitoring scripts | [] | No |

### Configuration File Locations

Capture automatically searches for configuration files in:
1. Current directory: `./capture.yaml`
2. Config directory: `./config/capture.yaml` 
3. User home: `$HOME/.capture/capture.yaml`
4. System: `/etc/capture/capture.yaml`

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for complete configuration documentation.

### Legacy Environment Variables

For backward compatibility, these environment variables are still supported:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `API_SECRET` | Authentication key | - | Yes |
| `PORT` | Server port number | 59232 | No |
| `GIN_MODE` | Gin framework mode | release | No |

## Installation Options

### Docker (Recommended)

Pull and run the official image:

```shell
docker run -d \
    -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=your-secret-key \
    ghcr.io/bluewave-labs/capture:latest
```

Or build locally:

```shell
docker buildx build -t capture .
docker run -d -v /etc/os-release:/etc/os-release:ro -p 59232:59232 -e API_SECRET=your-secret-key capture
```

Docker options explained:

- `-v /etc/os-release:/etc/os-release:ro`: Platform detection
- `-p 59232:59232`: Port mapping
- `-e API_SECRET`: Required authentication key
- `-d`: Detached mode

## System Installation

Choose one of these methods:

1. **Pre-built Binaries**: Download from [GitHub Releases](https://github.com/bluewave-labs/capture/releases)

2. **Go Package**:

   ```shell
   go install github.com/bluewave-labs/capture/cmd/capture@latest
   ```

3. **Build from Source**:

   ```shell
   git clone git@github.com:bluewave-labs/capture
   cd capture
   just build   # or: go build -o dist/capture ./cmd/capture/
   ```

## API Documentation

Our API is documented in accordance with the OpenAPI spec.

You can find the OpenAPI specifications [here](https://github.com/bluewave-labs/capture/blob/develop/openapi.yml)

## Contributing

We welcome contributions! If you would like to contribute, please read the [CONTRIBUTING.md](./CONTRIBUTING.md) file for more information.

<a href="https://github.com/bluewave-labs/capture/graphs/contributors">
  <img alt="Contributors Graph" src="https://contrib.rocks/image?repo=bluewave-labs/capture" />
</a>

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=bluewave-labs/capture&type=Date)](https://www.star-history.com/#bluewave-labs/capture&Date)

## License

Capture is licensed under AGPLv3. You can find the license [here](./LICENSE)
