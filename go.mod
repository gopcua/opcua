module github.com/gopcua/opcua

go 1.20

require (
	github.com/pascaldekloe/goe v0.1.0
	github.com/pkg/errors v0.8.1
	golang.org/x/crypto v0.7.0
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1
)

require (
	golang.org/x/sys v0.6.0 // indirect
	golang.org/x/term v0.6.0 // indirect
)

retract (
	v0.2.5 // https://github.com/gopcua/opcua/issues/538
	v0.2.4 // https://github.com/gopcua/opcua/issues/538
)
