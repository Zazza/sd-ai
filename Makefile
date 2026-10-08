.PHONY: setup dev build test lint tidy clean mcp

setup:
	go mod download
	cd frontend && npm install

dev:
	wails dev

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo dev)
WAILS ?= $(shell go env GOPATH)/bin/wails

build:
	cd frontend && npm install && cd ..
	$(WAILS) build -ldflags "-X main.version=$(VERSION)"

test:
	go vet ./...
	go test ./... -v
	cd frontend && npm run build

lint:
	go vet ./...
	cd frontend && npx vue-tsc --noEmit

tidy:
	go mod tidy

clean:
	rm -rf build/

mcp:
	go build -o build/bin/sd-mcp ./cmd/sd-mcp
