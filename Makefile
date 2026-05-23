BINARY  := bin/reverse-proxy
MODULE  := $(shell go list -m)

.PHONY: all lint test build build-all clean

all: lint test build

lint:
	golangci-lint run ./...

test:
	go test -v -race -coverprofile=coverage.out ./...

build:
	go build -o $(BINARY) .

build-all:
	GOOS=linux  GOARCH=amd64  go build -o bin/reverse-proxy-linux-amd64 .
	GOOS=darwin GOARCH=amd64  go build -o bin/reverse-proxy-darwin-amd64 .
	GOOS=darwin GOARCH=arm64  go build -o bin/reverse-proxy-darwin-arm64 .

clean:
	rm -rf bin/ coverage.out
