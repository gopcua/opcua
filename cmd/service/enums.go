package main

import (
	"io"
	"text/template"
)

type Enum struct {
	Name   string
	Type   string
	Values []*EnumValue
}

type EnumValue struct {
	Name  string
	Value int
}

func (e *Enum) Source(w io.Writer) error {
	return tmplEnum.Execute(w, e)
}

var tmplEnum = template.Must(template.New("").Parse(`
type {{.Name}} {{.Type}}

const (
	{{$Name := .Name}}
	{{range $i, $v := .Values}}{{$v.Name}} {{$Name}} = {{$v.Value}}
	{{end}}
)
`))
