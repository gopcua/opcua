// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"github.com/gopcua/opcua/ua"
)

func main() {
	in := flag.String("in", "../schema/Opc.Ua.PredefinedNodes.xml", "XML of predefined nodes")
	out := flag.String("out", "nodes_gen.go", "generated file")
	pkg := flag.String("pkg", "server", "package name")
	flag.Parse()

	log.SetFlags(0)

	f, err := os.Open(*in)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var nodes []Node
	d := xml.NewDecoder(f)
	for {
		tok, err := d.Token()
		if tok == nil || err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		switch ty := tok.(type) {
		case xml.StartElement:
			if ty.Name.Space != "http://opcfoundation.org/UA/" {
				continue
			}
			var n Node
			if err := d.DecodeElement(&n, &ty); err != nil {
				log.Fatal(err)
			}
			n.Type = ty.Name.Local
			fmt.Printf("%#v\n", n)
			nodes = append(nodes, n)
		}
	}

	data := map[string]interface{}{
		"Package": *pkg,
		"Nodes":   nodes,
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(*out, b.Bytes(), 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote %s/%s", *pkg, *out)
}

type Node struct {
	Type      string `xml:"-"`
	Xmlns     string `xml:",attr"`
	NodeClass string
	NodeId    struct {
		Identifier *ua.NodeID
	}
	BrowseName struct {
		NamespaceIndex string
		Name           string
	}
	ReferenceTypeId struct {
		Identifier *ua.NodeID
	}
	TypeDefinitionId struct {
		Identifier *ua.NodeID
	}
	IsAbstract bool
}

var tmpl = template.Must(template.New("").Parse(`// Generated code. DO NOT EDIT

// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package {{.Package}}

import 	"github.com/gopcua/opcua/ua"

func PredefinedNodes() []Node{
	return []Node{
{{- range .Nodes }}
{{- if not .IsAbstract }}
		&node{
			id: ua.NewNumericNodeID({{.NodeId.Identifier.Namespace}}, {{.NodeId.Identifier.IntID}}),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("{{.NodeClass}}"),
				ua.AttributeIDBrowseName: ua.MustVariant("{{.BrowseName.Name}}"),
			},
		},
{{- end }}
{{- end }}
	}
}
`))
