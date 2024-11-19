APP_NAME := "bwuagent"

format:
    @gofmt -w ./

format-check:
    @gofmt -l ./

test:
	@go test \
		-v \
		-timeout 30s \
		github.com/bluewave-labs/bluewave-uptime-agent/test

build:
    @go build -o dist/bwuagent ./cmd/bwuagent/

build-all: build-linux build-macos build-windows

build-linux:
	@GOOS=linux GOARCH=amd64 go build -o bin/linux/{{APP_NAME}}-linux-amd64 ./cmd/bwuagent
	@GOOS=linux GOARCH=arm64 go build -o bin/linux/{{APP_NAME}}-linux-arm64 ./cmd/bwuagent

build-macos:
	@GOOS=darwin GOARCH=amd64 go build -o bin/darwin/{{APP_NAME}}-darwin-amd64 ./cmd/bwuagent
	@GOOS=darwin GOARCH=arm64 go build -o bin/darwin/{{APP_NAME}}-darwin-arm64 ./cmd/bwuagent

build-windows:
	@GOOS=windows GOARCH=amd64 go build -o bin/windows/{{APP_NAME}}-windows-amd64.exe ./cmd/bwuagent
	@GOOS=windows GOARCH=arm64 go build -o bin/windows/{{APP_NAME}}-windows-arm64.exe ./cmd/bwuagent
