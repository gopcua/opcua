module github.com/gopcua/opcua

go 1.20

require (
	github.com/pascaldekloe/goe v0.1.1
	github.com/pkg/errors v0.9.1
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63
	golang.org/x/term v0.18.0
)

require golang.org/x/sys v0.18.0 // indirect

retract (
	v0.2.5 // https://github.com/gopcua/opcua/issues/538
	v0.2.4 // https://github.com/gopcua/opcua/issues/538
)
