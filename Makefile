# go test -count=1 disables the test cache so that all tests are run every time.

all: test integration examples

test:
	go test -count=1 -race ./...

lint:
	staticcheck ./...

integration:
	go test -count=1 -race -v -tags=integration ./uatest/...

examples:
	go build -o build/ ./examples/...

test-race:
	go test -count=1 -race ./...
	go test -count=1 -race -v -tags=integration ./uatest/...

install-py-opcua:
	pip3 install opcua

gen:
	go install golang.org/x/tools/cmd/stringer@latest
	go generate ./...
	go mod tidy

release:
	GITHUB_TOKEN=$$(security find-generic-password -gs GITHUB_TOKEN -w) goreleaser --clean

.PHONY: all examples gen integration test release
