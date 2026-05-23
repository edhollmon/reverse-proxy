BINARY  := bin/reverse-proxy
MODULE  := $(shell go list -m)

.PHONY: all lint test build clean

all: lint test build

lint:
	golangci-lint run ./...

test:
	go test -v -race -coverprofile=coverage.out ./...

build:
	go build -o $(BINARY) .

clean:
	rm -rf bin/ coverage.out
