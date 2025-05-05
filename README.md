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

Capture is only available for **Linux**.

## Docker Installation

Docker installation is **recommended** for running the Capture. Please see the [Docker run flags](#docker-run-flags) section for more information.

Pull the image from the registry and then run it with one command.

```shell
docker run -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=REPLACE_WITH_YOUR_SECRET \
    -d \
    ghcr.io/bluewave-labs/capture:latest
```

If you don't want to pull the image, you can build and run it locally.

```shell
docker buildx build -f Dockerfile -t capture .
```

```shell
docker run -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=REPLACE_WITH_YOUR_SECRET \
    -d \
    capture:latest
```

### Docker run flags

Before running the container, please make sure to replace the `REPLACE_WITH_YOUR_SECRET` with your own secret.

! **You need to put this secret to Checkmate's infrastructure monitoring dashboard**

- `-v /etc/os-release:/etc/os-release:ro` to get platform information correctly
- `-p 59232:59232` to expose the port 59232
- `-d` to run the container in detached mode
- `-e API_SECRET=REPLACE_WITH_YOUR_SECRET` to set the API secret
- (optional) `-e GIN_MODE=release/debug` to switch between release and debug mode

```shell
docker run -v /etc/os-release:/etc/os-release:ro \
    -p 59232:59232 \
    -e API_SECRET=REPLACE_WITH_YOUR_SECRET \
    -d \
    ghcr.io/bluewave-labs/capture:latest
```

## System Installation

### Pre-built Binaries

You can download the pre-built binaries from the [GitHub Releases](https://github.com/bluewave-labs/capture/releases) page.

### Go Package

You can install the Capture using the `go install` command.

```shell
go install github.com/bluewave-labs/capture/cmd/capture@latest
```

### Build from Source

You can build the Capture from the source code.

#### Prerequisites

- [Git](https://git-scm.com/downloads) is essential for cloning the repository.
- [Go](https://go.dev/dl/) is required to build the project.
- [Just](https://github.com/casey/just/releases) is optional but **recommended** for building the project with pre-defined commands.

#### Steps

1. Clone the repository

    ```shell
    git clone git@github.com:bluewave-labs/capture
    ```

2. Change the directory

    ```shell
    cd capture
    ```

3. Build the Capture

    ```shell
    just build
    ```

    or

    ```shell
    go build -o dist/capture ./cmd/capture/
    ```

4. Run the Capture

    ```shell
    ./dist/capture
    ```

## Environment Variables

Configure the capture with the following environment variables:

| Variable     | Description                          | Required/Optional |
| ------------ | ------------------------------------ | ----------------- |
| `PORT`       | The port that the Capture listens on | Optional          |
| `API_SECRET` | The secret key for the API           | Required          |
| `GIN_MODE`   | The mode of the Gin framework        | Optional          |

### Example

Please make sure to replace the default `your_secret` with your own secret.

! **You need to put this secret to Checkmate's infrastructure monitoring dashboard**

```shell
PORT = your_port
API_SECRET = your_secret
GIN_MODE = release/debug
```

```shell
# API_SECRET is required
API_SECRET=your_secret GIN_MODE=release ./capture

# Minimal required configuration
API_SECRET=your_secret ./dist/capture
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
