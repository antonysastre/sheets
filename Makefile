.PHONY: build install test lint clean

build:
	go build -o she ./cmd/she

install:
	go install ./cmd/she

test:
	go test -v ./...

lint:
	go vet ./...

clean:
	rm -f she