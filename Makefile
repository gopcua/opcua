# go test -count=1 disables the test cache so that all tests are run every time.

all: test integration selfintegration examples

test:
	go test -count=1 -race ./...

lint:
	staticcheck ./...

integration:
	go test -count=1 -race -v -tags=integration ./tests/python...

selfintegration:
	go test -count=1 -race -v -tags=integration ./tests/go...

examples:
	go build -o build/ ./examples/...

test-race:
	go test -count=1 -race ./...
	go test -count=1 -race -v -tags=integration ./tests/python...
	go test -count=1 -race -v -tags=integration ./tests/go...

install-py-opcua:
	pip3 install opcua

gen:
	which stringer || go install golang.org/x/tools/cmd/stringer@latest
	find . -name '*_gen.go' -delete
	go generate ./...

release:
	GITHUB_TOKEN=$$(security find-generic-password -gs GITHUB_TOKEN -w) goreleaser --clean

.PHONY: all examples gen integration test release
