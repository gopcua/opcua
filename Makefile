# GOFLAGS="-count=1" disables the test cache so that all tests are run every time.

all: test integration examples

test:
	GOFLAGS="-count=1" go test -race ./...

lint:
	staticcheck ./...

integration:
	GOFLAGS="-count=1" go test -race -v -tags=integration ./uatest/...

examples:
	go build -o build/ ./examples/...

test-race:
	GOFLAGS="-count=1" go test -race ./...
	GOFLAGS="-count=1" go test -race -v -tags=integration ./uatest/...

install-py-opcua:
	pip3 install opcua

gen:
	go get -d golang.org/x/tools/cmd/stringer
	go generate ./...
	go mod tidy

release:
	GITHUB_TOKEN=$$(security find-generic-password -gs GITHUB_TOKEN -w) goreleaser --rm-dist

.PHONY: all examples gen integration test release
