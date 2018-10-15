// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"text/template"
)

// WriteFile writes a go source file and formats it with goimports.
func WriteFile(filename string, data []byte) error {
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return err
	}
	return exec.Command("goimports", "-w", filename).Run()
}

// GenType generates the go source code for the given enum, struct or array type.
func GenType(pkg string, t *Type) ([]byte, error) {
	if t.Builtin {
		return nil, nil
	}
	switch t.Kind {
	case Enum:
		return gen(tmplEnum, map[string]interface{}{"Package": pkg, "T": t})
	case Struct:
		return gen(tmplStruct, map[string]interface{}{"Package": pkg, "T": t})
	case LenSlice:
		return gen(tmplLenSlice, map[string]interface{}{"Package": pkg, "T": t})
	default:
		return nil, nil
	}
}

// GenObject generates the go source code for the encoded_object helper
// which returns an empty object based on the id.
func GenObject(pkg string, types []*Type) ([]byte, error) {
	nodeIDs := map[string]int{}
	for _, t := range types {
		if t.NodeID > 0 {
			nodeIDs[t.Name] = t.NodeID
		}
	}
	return gen(tmplObject, map[string]interface{}{"Package": pkg, "NodeIDs": nodeIDs})
}

func gen(tmpl *template.Template, data map[string]interface{}) ([]byte, error) {
	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

var funcMap = template.FuncMap{
	"hex":      func(n int) string { return fmt.Sprintf("0x%x", n) },
	"isStruct": func(k Kind) bool { return k == Struct },
}

// tmplEnum is the go source code template for an enum type.
var tmplEnum = template.Must(template.New("Enum").Funcs(funcMap).Parse(`
// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package {{.Package}}

import (
	"bytes"

	ua "github.com/wmnsk/gopcua/datatypes"
)

type {{.T.Name}} {{.T.ElemType.GoType}}

const (
	{{range .T.Values}}{{.Name}} {{$.T.Name}} = {{hex .Value}}
	{{end}}
)

func Read{{.T.Name}}(b *bytes.Buffer) ({{.T.Name}}, error) {
	// fmt.Println("Read{{.T.Name}}")
	n, err := {{.T.ElemType.RWPackagePrefix}}Read{{.T.ElemType.Name}}(b)
	if err != nil {
		return 0, err
	}
	return {{.T.Name}}(n), nil
}

func Write{{.T.Name}}(b *bytes.Buffer, v {{.T.Name}}) error {
	// fmt.Println("Write{{.T.Name}}")
	return {{.T.ElemType.RWPackagePrefix}}Write{{.T.ElemType.Name}}(b, {{.T.ElemType.GoType}}(v))
}
`))

// tmplStruct is the go source code template for a struct type.
var tmplStruct = template.Must(template.New("Struct").Funcs(funcMap).Parse(`
// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package {{.Package}}

import (
	"bytes"

	ua "github.com/wmnsk/gopcua/datatypes"
)

type {{.T.Name}} struct {
	{{range .T.Fields}}{{.Name}} {{.Type.GoType}}
	{{end}}
}

func New{{.T.Name}}() {{.T.GoType}} {
	return &{{.T.Name}} {
		{{range .T.Fields}}{{if isStruct .Type.Kind}}{{.Name}} : {{.Type.RWPackagePrefix}}New{{.Type.Name}}(),{{end}}
		{{end -}}
	}
}

func (m *{{.T.Name}}) ID() uint16 {
	return {{.T.NodeID}}
}

func (m *{{.T.Name}}) GetName() string {
	return "{{.T.Name}}"
}

func (m *{{.T.Name}}) Read(b *bytes.Buffer) error {
	var err error
	{{range .T.Fields -}}
	if m.{{.Name}}, err = {{.Type.RWPackagePrefix}}Read{{.Type.Name}}(b); err != nil {
		return err
	}
	{{end -}}
	return nil
}

func (m *{{.T.Name}}) Write(b *bytes.Buffer) error {
	{{range .T.Fields -}}
	if err := {{.Type.RWPackagePrefix}}Write{{.Type.Name}}(b, m.{{.Name}}); err != nil {
		return err
	}
	{{end -}}
	return nil
}

func Read{{.T.Name}}(b *bytes.Buffer) ({{.T.GoType}}, error) {
	m := new({{.T.Name}})
	if err := m.Read(b); err != nil {
		return nil, err
	}
	return m, nil
}

func Write{{.T.Name}}(b *bytes.Buffer, m {{.T.GoType}}) error {
	return m.Write(b)
}
`))

// tmplLenSlice is the go source code template for a length encoded array type.
var tmplLenSlice = template.Must(template.New("LenSlice").Parse(`
// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package {{.Package}}

import(
	"bytes"

	ua "github.com/wmnsk/gopcua/datatypes"
)

type {{.T.Name}} {{.T.GoType}}

func (m {{.T.Name}}) GetName() string {
	return "{{.T.Name}}"
}

func Read{{.T.Name}}(b *bytes.Buffer) ({{.T.GoType}}, error) {
	n, err := ua.ReadUInt32(b)
	if err != nil {
		return nil, err
	}
	if n == ua.NilValue {
		return nil, nil
	}
	m := make({{.T.GoType}}, n)
	for i:=uint32(0); i < n; i++ {
		if m[i], err = {{.T.ElemType.RWPackagePrefix}}Read{{.T.ElemType.Name}}(b); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func Write{{.T.Name}}(b *bytes.Buffer, m {{.T.GoType}}) error {
	if m == nil {
		return ua.WriteUInt32(b, ua.NilValue)
	}
	if err := ua.WriteUInt32(b, uint32(len(m))); err != nil {
		return err
	}
	for i := range m {
		if err := {{.T.ElemType.RWPackagePrefix}}Write{{.T.ElemType.Name}}(b, m[i]); err != nil {
			return err
		}
	}
	return nil
}
`))

// tmplObject is the go source code template for the encoded object helper.
var tmplObject = template.Must(template.New("Object").Parse(`
// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package {{.Package}}

import(
	"bytes"

	ua "github.com/wmnsk/gopcua/datatypes"
)

const (
	{{range $k, $v := .NodeIDs -}}
	{{$k}}ID = {{$v}}
	{{end -}}
)

func NewObject(n uint16) Object {
	switch n {
		{{range $k, $v := .NodeIDs -}}
		case {{$v}}:
			return new({{$k}})
		{{end -}}
	default:
		return nil
	}
}
`))
