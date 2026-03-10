VERSION ?= $(shell grep -oP 'Version\s*=\s*"\K[^"]+' main.go 2>/dev/null || echo "0.0.0")
BINARY  := mgc
LDFLAGS := -s -w -X main.Version=$(VERSION)

.PHONY: all build test lint clean cross-build test-coverage vet

all: build

## build: Build the binary for the current platform
build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) .

## test: Run all tests with race detector
test:
	go test ./... -race -count=1 -timeout=120s

## test-coverage: Run tests and generate HTML coverage report
test-coverage:
	go test ./... -race -count=1 -coverprofile=coverage.out -timeout=120s
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

## vet: Run go vet
vet:
	go vet ./...

## lint: Run golangci-lint (must be installed)
lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not found — install from https://golangci-lint.run/"; exit 1)
	golangci-lint run

## cross-build: Build for all supported platforms
cross-build:
	mkdir -p bin
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64   .
	GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-linux-arm64   .
	GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-darwin-amd64  .
	GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-darwin-arm64  .
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY)-windows-amd64.exe .
	@echo "Cross-compilation complete:"
	@ls -la bin/

## clean: Remove build artifacts
clean:
	rm -rf bin/ dist/ coverage.out coverage.html

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
