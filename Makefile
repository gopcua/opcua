all: test integration examples

test:
	go test -race ./...

lint:
	staticcheck ./...

integration:
	go test -race -v -tags=integration ./uatest/...

examples:
	go build -o build/ ./examples/...

test-race:
	go test -race ./...
	go test -race -v -tags=integration ./uatest/...

install-py-opcua:
	pip3 install opcua

gen:
	go get -d golang.org/x/tools/cmd/stringer
	go generate ./...
	go mod tidy

release:
	GITHUB_TOKEN=$$(security find-generic-password -gs GITHUB_TOKEN -w) goreleaser --rm-dist

.PHONY: all examples gen integration test release
