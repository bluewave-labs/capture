APP_NAME := "capture"

# Keep tool versions in sync with .github/workflows/go.yml
STATICCHECK := "honnef.co/go/tools/cmd/staticcheck@v0.8.2"
GOLANGCI_LINT := "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0"
GOVULNCHECK := "golang.org/x/vuln/cmd/govulncheck@v1.10.0"

format:
	@go run {{GOLANGCI_LINT}} fmt

# Fails when any file is not formatted, like CI
format-check:
	@go run {{GOLANGCI_LINT}} fmt --diff

unit-test:
	@go test \
		-v \
		-timeout 3m \
		./test/ \
		./internal/...

integration-test:
	@go test \
		-v \
		-timeout 3m \
		./test/integration/...

# All tests, with the race detector, like CI
test:
	@go test \
		-race \
		-v \
		-timeout 3m \
		./...

# Same checks as the CI check job, for every target OS
lint:
	go mod tidy -diff
	@for goos in linux darwin windows; do \
		echo "GOOS=$goos"; \
		GOOS=$goos go vet ./... && \
		GOOS=$goos go run {{STATICCHECK}} -checks=all,-ST1000,-U1000 ./... && \
		GOOS=$goos go run {{GOLANGCI_LINT}} run || exit 1; \
	done
	GOOS=linux GOARCH=arm GOARM=6 go vet ./...

# Fails when the code reaches a vulnerability that has a fix
vuln:
	@go run {{GOVULNCHECK}} -format json ./... > .govulncheck.json
	@./.github/scripts/govulncheck-gate.sh .govulncheck.json; status=$?; rm -f .govulncheck.json; exit $status

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
