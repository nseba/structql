.PHONY: build test lint vet clean install

BINARY := structql
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/structql

install:
	go install $(LDFLAGS) ./cmd/structql

test:
	go test -race ./...

test-update:
	go test -race ./... -update

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/

all: vet lint test build
