# go test -count=1 disables the test cache so that all tests are run every time.

all: fmt test integration conformance selfintegration examples

test:
	go test -count=1 -race ./...

lint:
	staticcheck ./...

fmt:
	gofmt -w .

integration:
	go test -count=1 -race -v -tags=integration ./tests/python...

conformance:
	go test -count=1 -race -v -tags=conformance ./tests/reconnect/... ./tests/encoding/... ./tests/server/... ./tests/transport/... ./tests/securitypolicy/... ./tests/securechannel/... ./tests/session/...

selfintegration:
	go test -count=1 -race -v -tags=integration ./tests/go...

examples:
	go build -o build/ ./examples/...

# The race detector is not available on 386 or arm/v7. Both targets need a
# Linux host; for arm/v7 register QEMU first with
# docker run --privileged --rm tonistiigi/binfmt --install arm
test-386:
	GOOS=linux GOARCH=386 go test -count=1 ./...

# arm/v7 runs under emulation, so only the packages that decode and frame wire data.
test-armv7:
	GOOS=linux GOARCH=arm GOARM=7 go test -count=1 ./ua/... ./uasc/...

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

.PHONY: all examples gen integration conformance test test-386 test-armv7 release
