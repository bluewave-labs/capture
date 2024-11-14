# Installing Uptime Agent

BlueWave Uptime Agent currently only available for **Linux**. You can install the agent using Docker or manually.

## Docker Installation

1. Pull the docker image

    ```shell
    docker pull URL_TO_IMAGE
    ```

2. Run the container with specified flags

    ```shell
    docker run -v /etc/os-release:/etc/os-release:ro \
        -p 59232:59232 \
        -e API_SECRET=REPLACE_WITH_YOUR_SECRET \
        bwuagent:latest
    ```

3. You can access the agent at `http://localhost:59232`

## Manual Installation

### Build from Source

1. Git Clone

    ```shell
    git clone git@github.com:bluewave-labs/bluewave-uptime-agent.git
    ```

2. Change your directory

    ```shell
    cd bluewave-uptime-agent
    ```

3. Install dependencies

    ```shell
    go mod download
    ```

4. Build the project

    ```shell
    just build
    ```

    or

    ```shell
    go build -o bwuagent ./cmd/bwuagent/
    ```

5. Run the project

    ```shell
    # Use the compiled binary for production
    ./bwuagent
    ```

    or

    ```shell
    go run ./cmd/bwuagent/
    ```

6. You can access the agent at `http://localhost:59232`

### `go install`

You can also install the agent using `go install` command.

```shell
go install github.com/bluewave-labs/bluewave-uptime-agent/cmd/bwuagent@latest
```

Make sure the installed binary is executable

```shell
# Make sure $GOPATH/bin is in your PATH
chmod +x $(go env GOPATH)/bin/bwuagent
```

Run the installed binary

```shell
bwuagent
```

## Environment Variables

| ENV Variable Name | Required/Optional | Type      | Description                         | Accepted Values |
|-------------------|-------------------|-----------|-------------------------------------|-----------------|
| PORT              | Optional          | `integer` | Specifies Port for Server           | 0 - 65535       |
| API_SECRET        | Required          | `string`  | API Secret                          | Any string      |
| ALLOW_PUBLIC_API  | Optional          | `boolean` | Allow or deny publicly avaiable api | true, false     |
| GIN_MODE          | Optional          | `string`  | Gin mode                            | debug, release  |
