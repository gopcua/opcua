module github.com/gopcua/opcua

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/onsi/ginkgo/v2 v2.33.0
	github.com/onsi/gomega v1.44.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d
	golang.org/x/term v0.44.0
)

require (
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260402051712-545e8a4df936 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v0.7.2 // tagged the wrong branch
	v0.2.5 // https://github.com/gopcua/opcua/issues/538
	v0.2.4 // https://github.com/gopcua/opcua/issues/538
)
