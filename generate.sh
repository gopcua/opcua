#!/bin/sh

rm -f */*_gen.go
go run cmd/id/main.go
go run cmd/status/main.go
go run cmd/service/*.go
