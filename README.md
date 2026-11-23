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

<h1 align="center"><a href="https://bluewavelabs.ca" target="_blank">Capture</a></h1>

<p align="center"><strong>An open source hardware monitoring agent</strong></p>

Capture is a hardware monitoring agent that collects hardware information from the host machine and exposes it through a RESTful API. The agent is designed to be lightweight and easy to use.

- [Features](#features)
  - [Platform support](#platform-support)
- [Quick Start (Docker)](#quick-start-docker)
- [Quick Start (Docker Compose)](#quick-start-docker-compose)
- [Configuration](#configuration)
- [Installation Options](#installation-options)
  - [Docker (Recommended)](#docker-recommended)
    - [Docker container metrics](#docker-container-metrics)
  - [Helm](#helm)
  - [System Installation](#system-installation)
  - [Linux Systemd Service](#linux-systemd-service)
  - [macOS Launchd Service](#macos-launchd-service)
  - [Windows Service](#windows-service)
  - [Reverse Proxy and SSL](#reverse-proxy-and-ssl)
    - [Caddy](#caddy)
- [Using Capture with Checkmate](#using-capture-with-checkmate)
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
- S.M.A.R.T. Monitoring (Self-Monitoring, Analysis and Reporting Technology) for NVMe, SATA and SAS drives, one entry per drive. Requires smartmontools 7.0 or newer and, on most systems, running Capture as root. Not available in the Docker image, which does not include smartctl
  - Model, serial number, firmware and capacity
  - Overall health, temperature, power-on hours
  - Reallocated, pending and uncorrectable sectors (SATA/SAS); wear, spare and media errors (NVMe)
  - Warnings from smartctl, e.g. failing attributes or errors in the drive's error log
- Network Monitoring
- Software RAID (mdadm) Monitoring on Linux
  - Array state and missing disks
  - Rebuild progress
  - Member disk state
- Docker Container Monitoring (needs access to the Docker socket; see [Docker container metrics](#docker-container-metrics))

### Platform support

| Metric | Linux | macOS | Windows |
| ------ | ----- | ----- | ------- |
| CPU, memory, disk, network, host | ✅ | ✅ | ✅ |
| CPU temperature | ✅ | ❌ | ✅ where the machine exposes ACPI thermal zones |
| S.M.A.R.T. | ✅ | ✅ | ✅ |
| Software RAID (mdadm) | ✅ | ❌ | ❌ |
| Docker containers | ✅ | ✅ | ✅ |

S.M.A.R.T. needs smartmontools 7.0 or newer on the host, so it works with a system installation but not in the Docker image.

## Quick Start (Docker)

```shell
docker run -d \
    -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=your-secret-key \
    -e GIN_MODE=release \
    ghcr.io/bluewave-labs/capture:latest
```

To include Docker container metrics, see [Docker container metrics](#docker-container-metrics).

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
      - LISTEN=0.0.0.0
    volumes:
      - /etc/os-release:/etc/os-release:ro
      # Optional, for Docker container metrics. See "Docker container metrics" below.
      # - /var/run/docker.sock:/var/run/docker.sock:ro
    # group_add:
    #   - "999" # group ID of /var/run/docker.sock on the host: stat -c '%g' /var/run/docker.sock
```

## Configuration

| Variable     | Description                                                                                                                                                         | Default | Required |
| ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | -------- |
| `API_SECRET` | Authentication key ([Must match the secret you enter on Checkmate](https://docs.checkmate.so/users-guide/infrastructure-monitor#step-2-configure-general-settings)) | -       | Yes      |
| `PORT`       | Server port number                                                                                                                                                  | 59232   | No       |
| `LISTEN`     | IP address to bind to (e.g. `127.0.0.1` or `::1`), without a port or IPv6 brackets. Use an interface's IP address, not its name. | 0.0.0.0 | No |
| `GIN_MODE`   | Gin(web framework) mode. Debug is for development                                                                                                                   | release | No       |
| `SMART_DEVICES` | Comma-separated SMART devices as `path` or `path:type`, e.g. `/dev/sdb:sat,/dev/sdc:sat`. Overrides the device type from `smartctl --scan` for those paths (symlinks such as `/dev/disk/by-id/...` are matched to the scanned disk) and adds devices the scan misses. Checked at startup; an invalid value stops Capture with an error | - | No |

Example configurations:

```shell
# Minimal
API_SECRET=your-secret-key ./capture

# Complete
API_SECRET=your-secret-key LISTEN=0.0.0.0 PORT=59232 GIN_MODE=release ./capture

# Local connections only (for a host installation with a local reverse proxy)
API_SECRET=your-secret-key LISTEN=127.0.0.1 ./capture
```

When `LISTEN` is unset or empty, Capture listens on all IPv4 interfaces (`0.0.0.0`).
For containers, keep `LISTEN=0.0.0.0` to allow connections through Docker port mappings or from other containers.

## Installation Options

### Docker (Recommended)

Pull and run the official image:

```shell
docker run -d \
    -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=your-secret-key \
    -e GIN_MODE=release \
    ghcr.io/bluewave-labs/capture:latest
```

Image tags: `latest` points at the newest published release and moves only when a release is published. To choose when you upgrade, use a version tag instead, for example `ghcr.io/bluewave-labs/capture:v2.0.0`. Images are published for `linux/amd64` and `linux/arm64`.

S.M.A.R.T. is not available in the Docker image, because it needs smartctl and direct access to the drives. Use a [system installation](#system-installation) if you need drive health.

Or build locally:

```shell
docker buildx build -t capture .
docker run -d -v /etc/os-release:/etc/os-release:ro -p 59232:59232 -e API_SECRET=your-secret-key capture
```

Docker options explained:

- `-v /etc/os-release:/etc/os-release:ro`: Platform detection
- `-p 59232:59232`: Port mapping
- `-e API_SECRET`: Required authentication key
- `-e GIN_MODE=release`: Production logging. Without it the image currently logs in debug mode
- `-d`: Detached mode

#### Docker container metrics

`/api/v1/metrics/docker` reads from the Docker API, so the container needs the Docker socket. The image runs as a non-root user, so also give it the socket's group:

```shell
docker run -d \
    -v /etc/os-release:/etc/os-release:ro \
    -v /var/run/docker.sock:/var/run/docker.sock:ro \
    --group-add $(stat -c '%g' /var/run/docker.sock) \
    -p 59232:59232 \
    -e API_SECRET=your-secret-key \
    -e GIN_MODE=release \
    ghcr.io/bluewave-labs/capture:latest
```

On Docker Desktop (macOS and Windows) the socket's group inside containers is `0`, so use `--group-add 0`. Without the group, the endpoint returns a `permission denied` error.

Access to the Docker socket gives control over the Docker host. Only mount it on hosts where you need container metrics.

### Helm

You can deploy Capture on a Kubernetes cluster using the provided Helm chart.

1. Go to the Helm chart directory:

    ```shell
    cd deployment/helm/capture
    ```

2. Customize `values.yaml` and set your `API_SECRET`. The chart uses the `latest` image by default; set `image.tag` (for example `v2.0.0`) to pin a version.

3. Install the chart:

    ```shell
    helm install capture .
    ```

For detailed instructions, refer to the [Helm Installation Guide](./deployment/helm/capture/INSTALLATION.md).

### System Installation

Choose one of these methods:

1. **Pre-built Binaries**: Download from [GitHub Releases](https://github.com/bluewave-labs/capture/releases)

2. **Build from Source**:

   ```shell
   git clone https://github.com/bluewave-labs/capture.git
   cd capture
   just build   # or: go build -o dist/capture ./cmd/capture/
   ```

### Linux Systemd Service

An install script is provided at [`deployment/linux/install.sh`](./deployment/linux/install.sh). It automatically detects your CPU architecture, downloads the latest release from GitHub, installs the binary to `/usr/local/bin`, and registers a systemd service.

**Requirements:** `curl`, `tar`, and a system running systemd. Must be run as root.

1. Download and run in one step:

    ```shell
    curl -fsSL https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/linux/install.sh | sudo bash
    ```

    The script will prompt you for `API_SECRET` if it is not supplied as an argument.

2. Supply options inline if preferred:

    ```shell
    curl -fsSL https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/linux/install.sh \
        | sudo bash -s -- --api-secret "your-secret-key"
    ```

3. All available options:

    ```shell
    curl -fsSL https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/linux/install.sh \
        | sudo bash -s -- \
            --api-secret "your-secret-key" \
            --port 59232 \
            --listen 0.0.0.0 \
            --install-dir /usr/local/bin \
            --service-name capture
    ```

    Or download the script first and run it locally:

    ```shell
    curl -fsSL -o install.sh https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/linux/install.sh
    sudo bash install.sh --api-secret "your-secret-key"
    ```

4. Full option reference:

    | Option           | Description                              | Default              |
    | ---------------- | ---------------------------------------- | -------------------- |
    | `--api-secret`   | Authentication key (required)            | *(prompted)*         |
    | `--port`         | Port the agent listens on                | `59232`              |
    | `--listen`       | IP address the agent binds to            | `0.0.0.0`            |
    | `--install-dir`  | Directory to install the binary          | `/usr/local/bin`     |
    | `--service-name` | systemd service name                     | `capture`            |

5. After installation, manage the service with `systemctl`:

    ```shell
    systemctl status capture
    systemctl stop capture
    systemctl start capture
    systemctl restart capture
    journalctl -u capture -f
    ```

#### Manual setup

If you prefer to configure the service manually, a template service file is provided at [`deployment/linux/capture.service`](./deployment/linux/capture.service).

1. Edit the service file with your binary path, user, group, and secret key:

    ```shell
    nano deployment/linux/capture.service
    ```

    S.M.A.R.T. usually needs root to read the drives. The install script runs the service as root; if you set a different `User`, `/api/v1/metrics/smart` may return permission errors.

2. Copy it to the systemd unit directory:

    ```shell
    cp deployment/linux/capture.service /etc/systemd/system/
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

### macOS Launchd Service

An install script is provided at [`deployment/macos/install.sh`](./deployment/macos/install.sh). It automatically detects your CPU architecture, downloads the latest release from GitHub, installs the binary to `/usr/local/bin`, and registers a system-wide launchd daemon.

**Requirements:** `curl`, `tar`, macOS, and root access via `sudo`.

1. Download and run in one step:

    ```shell
    curl -fsSL https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/macos/install.sh | sudo bash
    ```

    The script will prompt you for `API_SECRET` if it is not supplied as an argument.

2. Supply options inline if preferred:

    ```shell
    curl -fsSL https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/macos/install.sh \
        | sudo bash -s -- --api-secret "your-secret-key"
    ```

3. All available options:

    ```shell
    curl -fsSL https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/macos/install.sh \
        | sudo bash -s -- \
            --api-secret "your-secret-key" \
            --port 59232 \
            --listen 0.0.0.0 \
            --install-dir /usr/local/bin \
            --service-name com.bluewavelabs.capture
    ```

    Or download the script first and run it locally:

    ```shell
    curl -fsSL -o install.sh https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/macos/install.sh
    sudo bash install.sh --api-secret "your-secret-key"
    ```

4. Full option reference:

    | Option           | Description                              | Default                      |
    | ---------------- | ---------------------------------------- | ---------------------------- |
    | `--api-secret`   | Authentication key (required)            | *(prompted)*                 |
    | `--port`         | Port the agent listens on                | `59232`                      |
    | `--listen`       | IP address the agent binds to            | `0.0.0.0`                    |
    | `--install-dir`  | Directory to install the binary          | `/usr/local/bin`             |
    | `--service-name` | launchd label and plist filename         | `com.bluewavelabs.capture`   |

5. After installation, manage the service with `launchctl`:

    ```shell
    sudo launchctl load /Library/LaunchDaemons/com.bluewavelabs.capture.plist
    sudo launchctl start com.bluewavelabs.capture
    sudo launchctl list | grep com.bluewavelabs.capture
    tail -f /var/log/com.bluewavelabs.capture.log
    ```

<!-- markdownlint-disable MD024 -->
#### Manual setup

If you prefer to configure the service manually, a template plist is provided at [`deployment/macos/capture.plist`](./deployment/macos/capture.plist).

1. Edit the plist with your binary path, working directory, secret key, and any other settings you need:

    ```shell
    nano deployment/macos/capture.plist
    ```

    The install script runs the daemon as root. If you set a different `UserName`, S.M.A.R.T. may not be able to read every drive.

2. Copy it to the launchd daemon directory and set the required permissions:

    ```shell
    cp deployment/macos/capture.plist /Library/LaunchDaemons/com.bluewavelabs.capture.plist
    chown root:wheel /Library/LaunchDaemons/com.bluewavelabs.capture.plist
    chmod 644 /Library/LaunchDaemons/com.bluewavelabs.capture.plist
    ```

3. Load and start the service:

    ```shell
    sudo launchctl load /Library/LaunchDaemons/com.bluewavelabs.capture.plist
    sudo launchctl start com.bluewavelabs.capture
    ```

4. Verify the service is running:

    ```shell
    sudo launchctl list | grep com.bluewavelabs.capture
    ```

### Windows Service

A PowerShell install script is provided at [`deployment/windows/install.ps1`](./deployment/windows/install.ps1). It automatically detects your CPU architecture, downloads the latest release from GitHub, installs the binary, and registers a Windows service.

**Requirements:** PowerShell 5.1+ and an Administrator terminal.

1. Open PowerShell **as Administrator** and run in one step:

    ```powershell
    Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
    Invoke-RestMethod https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/windows/install.ps1 | Invoke-Expression
    ```

    The script will prompt you for `API_SECRET` if it is not supplied as a parameter.

2. Supply parameters inline if preferred:

    ```powershell
    & ([scriptblock]::Create(
        (Invoke-RestMethod https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/windows/install.ps1)
    )) -APISecret "your-secret-key"
    ```

3. All available parameters:

    ```powershell
    & ([scriptblock]::Create(
        (Invoke-RestMethod https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/windows/install.ps1)
    )) -APISecret "your-secret-key" -Port 59232 -Listen "0.0.0.0" -InstallDir "C:\Program Files\Capture" -ServiceName "capture"
    ```

    Or download the script first and run it locally:

    ```powershell
    Invoke-RestMethod https://raw.githubusercontent.com/bluewave-labs/capture/main/deployment/windows/install.ps1 -OutFile install.ps1
    Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
    .\install.ps1 -APISecret "your-secret-key"
    ```

4. Full parameter reference:

    | Parameter      | Description                              | Default                      |
    | -------------- | ---------------------------------------- | ---------------------------- |
    | `-APISecret`   | Authentication key (required)            | *(prompted)*                 |
    | `-Port`        | Port the agent listens on                | `59232`                      |
    | `-Listen`      | IP address the agent binds to            | `0.0.0.0`                    |
    | `-InstallDir`  | Directory to install the binary          | `C:\Program Files\Capture`   |
    | `-ServiceName` | Windows service name                     | `capture`                    |

5. After installation, manage the service with standard PowerShell cmdlets:

    ```powershell
    Get-Service capture
    Stop-Service capture
    Start-Service capture
    Restart-Service capture
    ```

6. If the port is taken or the `-Listen` address does not exist yet (for example at boot, before DHCP assigns it), the service waits in *Start pending* and retries for up to 60 seconds. A configuration error, or a port that is still unavailable after that, stops the service with an error in the Windows Event Log (Application log, source `capture`).

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

## Using Capture with Checkmate

[Checkmate](https://github.com/bluewave-labs/checkmate) shows Capture's data in its infrastructure monitors.

1. Install and start Capture on the machine you want to monitor.
2. In Checkmate, create an infrastructure monitor with the URL `http://<capture-host>:59232/api/v1/metrics` and the same secret as `API_SECRET`. The [Checkmate guide](https://docs.checkmate.so/users-guide/infrastructure-monitor#step-2-configure-general-settings) has the details.
3. Make sure Checkmate can reach Capture's port. If Capture uses `LISTEN=127.0.0.1`, only local connections work.

Checkmate currently shows CPU, memory, disk, network and host data. S.M.A.R.T. and software RAID data from Capture 2.0 are not shown in Checkmate yet.

## Endpoints

- **Base URL**: `http://<host>:<PORT>` (default port `59232`)
- **Authentication**: Every `/api/v1/**` route requires `Authorization: Bearer $API_SECRET`. `/health` stays public so you can use it for liveness checks.

| Method | Path                     | Auth | Description                                                                                         | Notes                                          |
| ------ | ------------------------ | ---- | --------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `GET`  | `/health`                | ❌   | Liveness probe that returns `"OK"`.                                                                 | Useful for container orchestrators.            |
| `GET`  | `/api/v1/metrics`        | ✅   | Returns CPU, memory, disk, host, network, and software RAID data.                                   | SMART and Docker have their own endpoints.     |
| `GET`  | `/api/v1/metrics/cpu`    | ✅   | CPU temps, load, and utilization.                                                                   |                                                |
| `GET`  | `/api/v1/metrics/memory` | ✅   | Memory totals and usage metrics.                                                                    |                                                |
| `GET`  | `/api/v1/metrics/disk`   | ✅   | Disk capacity, inode usage, and IO stats.                                                           |                                                |
| `GET`  | `/api/v1/metrics/host`   | ✅   | Host metadata (OS, uptime, kernel, etc.).                                                           |                                                |
| `GET`  | `/api/v1/metrics/smart`  | ✅   | S.M.A.R.T. health, one entry per drive.                                                             | Since v2.0.0 `data` is a list (before: one object). Identify drives by `serial_number`; device paths can change after a reboot. |
| `GET`  | `/api/v1/metrics/net`    | ✅   | Interface-level network throughput.                                                                 |                                                |
| `GET`  | `/api/v1/metrics/raid`   | ✅   | Software RAID (mdadm) array state, missing disks, rebuild progress, and member state.               | Linux only; empty list when there are no arrays. |
| `GET`  | `/api/v1/metrics/docker` | ✅   | Docker container metrics.                                                                           | Use `?all=true` to include stopped containers. |

All responses share the same envelope:

```jsonc
{
  "data": {
    // collector-specific payload: an object, or a list for smart, raid and docker
  },
  "capture": {
    "version": "2.0.0",
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

You can find the OpenAPI specifications [in openapi.yml](https://github.com/bluewave-labs/capture/blob/main/openapi.yml)

## Contributing

We welcome contributions! If you would like to contribute, please read the [CONTRIBUTING.md](./CONTRIBUTING.md) file for more information.

<a href="https://github.com/bluewave-labs/capture/graphs/contributors">
  <img alt="Contributors Graph" src="https://contrib.rocks/image?repo=bluewave-labs/capture" />
</a>

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=bluewave-labs/capture&type=Date)](https://www.star-history.com/#bluewave-labs/capture&Date)

## License

Capture is licensed under AGPLv3. You can find the license in the [LICENSE](./LICENSE) file.
