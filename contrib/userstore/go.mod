module github.com/gopcua/opcua/contrib/userstore

go 1.23.0

require (
	github.com/gopcua/opcua v0.8.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.31.0
	golang.org/x/term v0.27.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
)

replace github.com/gopcua/opcua => ../..
