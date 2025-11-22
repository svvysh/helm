APP := helm
BIN ?= bin/$(APP)
CMD ?= run
ARGS ?=
RELEASE_PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
GO_BIN := $(shell go env GOBIN)
ifeq ($(GO_BIN),)
GO_BIN := $(shell go env GOPATH)/bin
endif

.PHONY: run build deps test vet tidy fmt fmt-tools lint lint-tools all release clean

all: deps tidy fmt vet lint test build release

run:
	go run ./cmd/$(APP) $(CMD) $(ARGS)

build:
	go build -o $(BIN) ./cmd/$(APP)

deps:
	go mod download

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

fmt: fmt-tools
	$(GO_BIN)/gofumpt -w .
	$(GO_BIN)/goimports -w .

fmt-tools:
	@mkdir -p "$(GO_BIN)"
	@command -v "$(GO_BIN)/gofumpt" >/dev/null 2>&1 || GOBIN="$(GO_BIN)" go install mvdan.cc/gofumpt@latest
	@command -v "$(GO_BIN)/goimports" >/dev/null 2>&1 || GOBIN="$(GO_BIN)" go install golang.org/x/tools/cmd/goimports@latest

lint: lint-tools
	$(GO_BIN)/golangci-lint run ./...

lint-tools:
	@mkdir -p "$(GO_BIN)"
	@command -v "$(GO_BIN)/golangci-lint" >/dev/null 2>&1 || GOBIN="$(GO_BIN)" go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

clean:
	rm -rf dist bin

release: clean
	mkdir -p dist
	for platform in $(RELEASE_PLATFORMS); do \
		GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
		EXT=$$( [ "$${GOOS}" = "windows" ] && echo ".exe" ); \
		OUTPUT=dist/$(APP)_$${GOOS}_$${GOARCH}$${EXT}; \
		echo "Building $$OUTPUT"; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -trimpath -ldflags "-s -w" -o $$OUTPUT ./cmd/$(APP); \
	done
