# Gospan — Makefile
# Thin wrapper around the Go toolchain. Works with `make` on Windows (e.g. via
# `scoop install make`, Git Bash, or MSYS2) and on any Unix.

BINARY  := gospan
PKG     := ./cmd/gospan
DIST    := dist
VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all fmt vet test test-race bench build gui release clean tidy

all: fmt vet test

fmt:
	gofmt -s -w .

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

bench:
	go test "-bench=." "-benchmem" "./internal/engine/..."

build:
	go build -o $(DIST)/$(BINARY).exe $(PKG)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o $(DIST)/$(BINARY)-linux ./cmd/gospan

build-macos:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o $(DIST)/$(BINARY)-macos ./cmd/gospan

gui:
	cd gui/fyne && go build -trimpath -ldflags "-s -w -H=windowsgui" -o ../../$(DIST)/$(BINARY)-gui.exe .

release:
	set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY).exe $(PKG)

tidy:
	go mod tidy

clean:
	go clean
	rm -rf $(DIST)
