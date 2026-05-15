.PHONY: build install test lint clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags '$(LDFLAGS)' -o she ./cmd/she

install:
	go install -ldflags '$(LDFLAGS)' ./cmd/she

test:
	go test -v ./...

lint:
	go vet ./...

clean:
	rm -f she
