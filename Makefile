APP := helm
BIN ?= bin/$(APP)
CMD ?= run
ARGS ?=

.PHONY: run build deps test vet tidy
.PHONY: all

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
