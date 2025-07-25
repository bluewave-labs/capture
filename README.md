![github-license](https://img.shields.io/github/license/bluewave-labs/capture)
![github-repo-size](https://img.shields.io/github/repo-size/bluewave-labs/capture)
![github-commit-activity](https://img.shields.io/github/commit-activity/w/bluewave-labs/capture)
![github-last-commit-data](https://img.shields.io/github/last-commit/bluewave-labs/capture)
![github-languages](https://img.shields.io/github/languages/top/bluewave-labs/capture)
![github-issues-and-prs](https://img.shields.io/github/issues-pr/bluewave-labs/capture)
![github-issues](https://img.shields.io/github/issues/bluewave-labs/capture)
[![go-reference](https://pkg.go.dev/badge/github.com/bluewave-labs/capture.svg)](https://pkg.go.dev/github.com/bluewave-labs/capture)
[![github-actions-lint](https://github.com/bluewave-labs/capture/actions/workflows/lint.yml/badge.svg)](https://github.com/bluewave-labs/capture/actions/workflows/lint.yml)

<h1 align="center"><a href="https://bluewavelabs.ca" target="_blank">Capture</a></h1>

<p align="center"><strong>An open source hardware monitoring agent</strong></p>

Capture is a hardware monitoring agent that collects hardware information from the host machine and exposes it through a RESTful API. The agent is designed to be lightweight and easy to use.

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
| `GIN_MODE`   | Gin(web framework) mode. Debug is for development                                                                                                                                  | release | No       |

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
