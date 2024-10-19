APP_NAME := "bwuagent"

format:
    @gofmt -w ./

format-check:
    @gofmt -l ./

test:
	@go test \
		-v \
		-timeout 30s \
		bluewave-uptime-agent/test

build:
    @go build -o bwuagent ./cmd/api/

build-all: build-linux build-macos build-windows

build-linux:
	@GOOS=linux GOARCH=amd64 go build -o bin/linux/{{APP_NAME}}-linux-amd64 ./cmd/api
	@GOOS=linux GOARCH=arm64 go build -o bin/linux/{{APP_NAME}}-linux-arm64 ./cmd/api

build-macos:
	@GOOS=darwin GOARCH=amd64 go build -o bin/darwin/{{APP_NAME}}-darwin-amd64 ./cmd/api
	@GOOS=darwin GOARCH=arm64 go build -o bin/darwin/{{APP_NAME}}-darwin-arm64 ./cmd/api

build-windows:
	@GOOS=windows GOARCH=amd64 go build -o bin/windows/{{APP_NAME}}-windows-amd64.exe ./cmd/api
	@GOOS=windows GOARCH=arm64 go build -o bin/windows/{{APP_NAME}}-windows-arm64.exe ./cmd/api