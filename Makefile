all: test integration

test:
	go test ./...

integration:
	go test -v -tags=integration ./uatest/...

install-py-opcua:
	pip3 install opcua

release:
	GITHUB_TOKEN=$$(security find-generic-password -gs GITHUB_TOKEN -w) goreleaser --rm-dist
