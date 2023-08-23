module github.com/gopcua/opcua

go 1.22.0

require (
	github.com/google/uuid v1.3.0
	github.com/pascaldekloe/goe v0.1.1
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d
	golang.org/x/term v0.8.0
)

require golang.org/x/sys v0.8.0 // indirect

retract (
	v0.2.5 // https://github.com/gopcua/opcua/issues/538
	v0.2.4 // https://github.com/gopcua/opcua/issues/538
)
