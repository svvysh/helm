APP := helm
BIN ?= bin/$(APP)
CMD ?= run
ARGS ?=
RELEASE_PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: run build deps test vet tidy all release clean

all: deps tidy vet test build

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
