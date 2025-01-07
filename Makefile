VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X github.com/y0ug/ai-helper/internal/version.Version=$(VERSION) \
           -X github.com/y0ug/ai-helper/internal/version.CommitHash=$(COMMIT_HASH) \
           -X github.com/y0ug/ai-helper/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" ./cmd/ai-helper

.PHONY: release
release:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/ai-helper_linux_amd64 ./cmd/ai-helper
	GOOS=linux GOARCH=386 go build -ldflags "$(LDFLAGS)" -o dist/ai-helper_linux_386 ./cmd/ai-helper
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/ai-helper_linux_arm64 ./cmd/ai-helper
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/ai-helper_darwin_amd64 ./cmd/ai-helper
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/ai-helper_darwin_arm64 ./cmd/ai-helper

.PHONY: test
test:
	go test -v ./...
