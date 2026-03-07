<!-- Disable markdownlint rule MD041 (first line in file should be a top-level heading) since we have badges at the top and h1. -->
<!-- markdownlint-disable MD041 -->
![github-license](https://img.shields.io/github/license/bluewave-labs/capture)
![github-repo-size](https://img.shields.io/github/repo-size/bluewave-labs/capture)
![github-commit-activity](https://img.shields.io/github/commit-activity/w/bluewave-labs/capture)
![github-last-commit-data](https://img.shields.io/github/last-commit/bluewave-labs/capture)
![github-languages](https://img.shields.io/github/languages/top/bluewave-labs/capture)
![github-issues-and-prs](https://img.shields.io/github/issues-pr/bluewave-labs/capture)
![github-issues](https://img.shields.io/github/issues/bluewave-labs/capture)
[![go-reference](https://pkg.go.dev/badge/github.com/bluewave-labs/capture.svg)](https://pkg.go.dev/github.com/bluewave-labs/capture)
[![github-actions-go](https://github.com/bluewave-labs/capture/actions/workflows/go.yml/badge.svg)](https://github.com/bluewave-labs/capture/actions/workflows/go.yml)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/bluewave-labs/capture)

<h1 align="center"><a href="https://bluewavelabs.ca" target="_blank">Capture</a></h1>

<p align="center"><strong>An open source hardware monitoring agent</strong></p>

Capture is a hardware monitoring agent that collects hardware information from the host machine and exposes it through a RESTful API. The agent is designed to be lightweight and easy to use.

- [Features](#features)
- [Quick Start (Docker)](#quick-start-docker)
- [Quick Start (Docker Compose)](#quick-start-docker-compose)
- [Configuration](#configuration)
- [Installation Options](#installation-options)
  - [Docker (Recommended)](#docker-recommended)
  - [Helm](#helm)
  - [System Installation](#system-installation)
  - [Linux Systemd Service](#linux-systemd-service)
  - [Windows Service](#windows-service)
  - [Reverse Proxy and SSL](#reverse-proxy-and-ssl)
    - [Caddy](#caddy)
- [Endpoints](#endpoints)
- [API Documentation](#api-documentation)
- [Contributing](#contributing)
- [Star History](#star-history)
- [License](#license)

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

| Variable     | Description                                                                                                                                                         | Default | Required |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | -------- |
| `API_SECRET` | Authentication key ([Must match the secret you enter on Checkmate](https://docs.checkmate.so/users-guide/infrastructure-monitor#step-2-configure-general-settings)) | -       | Yes      |
| `PORT`       | Server port number                                                                                                                                                  | 59232   | No       |
| `GIN_MODE`   | Gin(web framework) mode. Debug is for development                                                                                                                   | release | No       |

Example configurations:

```shell
# Minimal
API_SECRET=your-secret-key ./capture

# Complete
API_SECRET=your-secret-key PORT=59232 GIN_MODE=release ./capture
```

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

### Helm

You can deploy Capture on a Kubernetes cluster using the provided Helm chart.

1. Go to the Helm chart directory:

    ```shell
    cd deployment/helm/capture
    ```

2. Customize `values.yaml` and set your `API_SECRET`.

3. Install the chart:

    ```shell
    helm install capture .
    ```

For detailed instructions, refer to the [Helm Installation Guide](./deployment/helm/capture/INSTALLATION.md).

### System Installation

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

### Linux Systemd Service

An install script is provided at [`deployment/linux/install.sh`](./deployment/linux/install.sh). It automatically detects your CPU architecture, downloads the latest release from GitHub, installs the binary to `/usr/local/bin`, and registers a systemd service.

**Requirements:** `curl`, `tar`, and a system running systemd. Must be run as root.

1. Run the script:

    ```shell
    sudo bash deployment/linux/install.sh
    ```

    The script will prompt you for `API_SECRET` if it is not supplied as an argument.

2. Supply options inline if preferred:

    ```shell
    sudo bash deployment/linux/install.sh --api-secret "your-secret-key"
    ```

3. Full option reference:

    | Option           | Description                              | Default              |
    | ---------------- | ---------------------------------------- | -------------------- |
    | `--api-secret`   | Authentication key (required)            | *(prompted)*         |
    | `--port`         | Port the agent listens on                | `59232`              |
    | `--install-dir`  | Directory to install the binary          | `/usr/local/bin`     |
    | `--service-name` | systemd service name                     | `capture`            |

4. After installation, manage the service with `systemctl`:

    ```shell
    systemctl status capture
    systemctl stop capture
    systemctl start capture
    systemctl restart capture
    journalctl -u capture -f
    ```

#### Manual setup

If you prefer to configure the service manually, a template service file is provided at [`deployment/systemd/capture.service`](./deployment/systemd/capture.service).

1. Edit the service file with your binary path, user, group, and secret key:

    ```shell
    nano deployment/systemd/capture.service
    ```

2. Copy it to the systemd unit directory:

    ```shell
    cp deployment/systemd/capture.service /etc/systemd/system/
    ```

3. Reload, enable, and start the service:

    ```shell
    systemctl daemon-reload
    systemctl enable capture
    systemctl start capture
    ```

4. Verify the service is running:

    ```shell
    systemctl status capture
    ```

### Windows Service

A PowerShell install script is provided at [`deployment/windows/install.ps1`](./deployment/windows/install.ps1). It automatically detects your CPU architecture, downloads the latest release from GitHub, installs the binary, and registers a Windows service.

**Requirements:** PowerShell 5.1+ and an Administrator terminal.

1. Open PowerShell **as Administrator** and run:

    ```powershell
    Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
    .\deployment\windows\install.ps1
    ```

    The script will prompt you for `API_SECRET` if it is not supplied as a parameter.

2. Supply options inline if preferred:

    ```powershell
    .\deployment\windows\install.ps1 -APISecret "your-secret-key"
    ```

3. Full parameter reference:

    | Parameter      | Description                              | Default                      |
    | -------------- | ---------------------------------------- | ---------------------------- |
    | `-APISecret`   | Authentication key (required)            | *(prompted)*                 |
    | `-Port`        | Port the agent listens on                | `59232`                      |
    | `-InstallDir`  | Directory to install the binary          | `C:\Program Files\Capture`   |
    | `-ServiceName` | Windows service name                     | `capture`                    |

4. After installation, manage the service with standard PowerShell cmdlets:

    ```powershell
    Get-Service capture
    Stop-Service capture
    Start-Service capture
    Restart-Service capture
    ```

### Reverse Proxy and SSL

You can use a reverse proxy in front of the Capture service to handle HTTPS requests and SSL termination.

#### Caddy

```lua
├deployment/reverse-proxy-compose/
├── caddy/
│   └── Caddyfile
└── caddy.compose.yml
```

1. Go to the `deployment/reverse-proxy-compose` directory

    ```shell
    cd deployment/reverse-proxy-compose
    ```

2. Replace `replacewithyourdomain.com` with your actual domain in [deployment/reverse-proxy-compose/caddy/Caddyfile](./deployment/reverse-proxy-compose/caddy/Caddyfile)
3. Set `API_SECRET` environment variable for the Capture service in [deployment/reverse-proxy-compose/caddy.compose.yml](./deployment/reverse-proxy-compose/caddy.compose.yml).
4. Ensure your domain’s DNS A/AAAA records point to this server’s IP.
5. Open inbound TCP ports 80 and 443 on your firewall/security group.

Start the Caddy reverse proxy

```shell
docker compose -f caddy.compose.yml up -d
```

## Endpoints

- **Base URL**: `http://<host>:<PORT>` (default port `59232`)
- **Authentication**: Every `/api/v1/**` route requires `Authorization: Bearer $API_SECRET`. `/health` stays public so you can use it for liveness checks.

| Method | Path                     | Auth | Description                                                                                         | Notes                                          |
| ------ | ------------------------ | ---- | --------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `GET`  | `/health`                | ❌   | Liveness probe that returns `"OK"`.                                                                 | Useful for container orchestrators.            |
| `GET`  | `/api/v1/metrics`        | ✅   | Returns the complete capture payload with CPU, memory, disk, host, SMART, network, and Docker data. | Aggregates every collector.                    |
| `GET`  | `/api/v1/metrics/cpu`    | ✅   | CPU temps, load, and utilization.                                                                   |                                                |
| `GET`  | `/api/v1/metrics/memory` | ✅   | Memory totals and usage metrics.                                                                    |                                                |
| `GET`  | `/api/v1/metrics/disk`   | ✅   | Disk capacity, inode usage, and IO stats.                                                           |                                                |
| `GET`  | `/api/v1/metrics/host`   | ✅   | Host metadata (OS, uptime, kernel, etc.).                                                           |                                                |
| `GET`  | `/api/v1/metrics/smart`  | ✅   | S.M.A.R.T. drive health information.                                                                |                                                |
| `GET`  | `/api/v1/metrics/net`    | ✅   | Interface-level network throughput.                                                                 |                                                |
| `GET`  | `/api/v1/metrics/docker` | ✅   | Docker container metrics.                                                                           | Use `?all=true` to include stopped containers. |

All responses share the same envelope:

```jsonc
{
  "data": {
    // collector-specific payload
  },
  "capture": {
    "version": "1.0.0",
    "mode": "release"
  },
  "errors": [
    // optional array of error messages if any collectors failed, can be null
  ],
}
```

Collectors can partially fail; when that happens the API responds with HTTP `207 Multi-Status` and fills `errors` with detailed reasons so you can alert without dropping other metric data.

## API Documentation

Our API is documented in accordance with the OpenAPI spec.

You can find the OpenAPI specifications [in openapi.yml](https://github.com/bluewave-labs/capture/blob/develop/openapi.yml)

## Contributing

We welcome contributions! If you would like to contribute, please read the [CONTRIBUTING.md](./CONTRIBUTING.md) file for more information.

<a href="https://github.com/bluewave-labs/capture/graphs/contributors">
  <img alt="Contributors Graph" src="https://contrib.rocks/image?repo=bluewave-labs/capture" />
</a>

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=bluewave-labs/capture&type=Date)](https://www.star-history.com/#bluewave-labs/capture&Date)

## License

Capture is licensed under AGPLv3. You can find the license in the [LICENSE](./LICENSE) file.
