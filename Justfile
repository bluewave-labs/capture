APP_NAME := "capture"

format:
    @gofmt -w ./

format-check:
    @gofmt -l ./

unit-test:
	@go test \
		-v \
		-timeout 30s \
		./test/ \
		./internal/...

integration-test:
	@go test \
		-v \
		-timeout 30s \
		./test/integration/...

test:
	@go test \
		-v \
		-timeout 30s \
		./test/...

lint:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run

build:
    @go build -o dist/capture ./cmd/capture/

build-all: build-linux build-macos build-windows

build-linux:
	@GOOS=linux GOARCH=amd64 go build -o dist/linux/{{APP_NAME}}-linux-amd64 ./cmd/capture
	@GOOS=linux GOARCH=arm64 go build -o dist/linux/{{APP_NAME}}-linux-arm64 ./cmd/capture

build-macos:
	@GOOS=darwin GOARCH=amd64 go build -o dist/darwin/{{APP_NAME}}-darwin-amd64 ./cmd/capture
	@GOOS=darwin GOARCH=arm64 go build -o dist/darwin/{{APP_NAME}}-darwin-arm64 ./cmd/capture

build-windows:
	@GOOS=windows GOARCH=amd64 go build -o dist/windows/{{APP_NAME}}-windows-amd64.exe ./cmd/capture
	@GOOS=windows GOARCH=arm64 go build -o dist/windows/{{APP_NAME}}-windows-arm64.exe ./cmd/capture

openapi-contract-test:
	@uvx schemathesis \
		--config-file ./schemathesis.toml \
		run ./openapi.yml \
		--url http://127.0.0.1:59232 \
		--report har \
		--report-dir .schemathesis-report
