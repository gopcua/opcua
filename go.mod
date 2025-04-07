module github.com/gopcua/opcua

go 1.23

require (
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d
	golang.org/x/term v0.27.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v0.7.2 // tagged the wrong branch
	v0.2.5 // https://github.com/gopcua/opcua/issues/538
	v0.2.4 // https://github.com/gopcua/opcua/issues/538
)
