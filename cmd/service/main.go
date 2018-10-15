// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unicode"

	"github.com/wmnsk/gopcua/cmd/service/opc"
)

// wantTypes is the list of service objects and length encoded arrays
// we should generate code for. Leave empty to generate code for all
// service objects.
var wantTypes = []string{
	// services
	"ActivateSessionRequest",
	"ActivateSessionResponse",
	"AnonymousIdentityToken",
	"ApplicationDescription",
	"ApplicationType",
	"ChannelSecurityToken",
	"CloseSecureChannelRequest",
	"CloseSessionRequest",
	"CloseSessionResponse",
	"CreateMonitoredItemsRequest",
	"CreateMonitoredItemsResponse",
	"CreateSessionRequest",
	"CreateSessionResponse",
	"CreateSubscriptionRequest",
	"CreateSubscriptionResponse",
	"EndpointDescription",
	"GetEndpointsRequest",
	"GetEndpointsResponse",
	"MessageSecurityMode",
	"MonitoredItemCreateRequest",
	"MonitoredItemCreateResult",
	"MonitoringMode",
	"MonitoringParameters",
	"NotificationMessage",
	"OpenSecureChannelRequest",
	"OpenSecureChannelResponse",
	"PublishRequest",
	"PublishResponse",
	"ReadRequest",
	"ReadResponse",
	"ReadValueId",
	"RequestHeader",
	"ResponseHeader",
	"SecurityTokenRequestType",
	"SignatureData",
	"SignedSoftwareCertificate",
	"SubscriptionAcknowledgement",
	"TimestampsToReturn",
	"UserTokenPolicy",
	"UserTokenType",

	// arrays
	"DataValueArray",
	"DiagnosticInfoArray",
	"EndpointDescriptionArray",
	"ExtensionObjectArray",
	"MonitoredItemCreateRequestArray",
	"MonitoredItemCreateResultArray",
	"ReadValueIdArray",
	"SignedSoftwareCertificateArray",
	"StatusCodeArray",
	"StringArray",
	"SubscriptionAcknowledgementArray",
	"UInt32Array",
	"UserTokenPolicyArray",
}

func main() {
	var typesFile, nodeIDsFile, out, pkg string
	flag.StringVar(&typesFile, "types", "schema/Opc.Ua.Types.bsd", "Path to Opc.Ua.Types.bsd file")
	flag.StringVar(&nodeIDsFile, "node-ids", "schema/NodeIDs.csv", "Path to NodeIds.csv file")
	flag.StringVar(&out, "out", "gen", "Path to output directory")
	flag.StringVar(&pkg, "pkg", "service", "Go package name")
	flag.Parse()

	path := filepath.Join(out, pkg)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %s", err)
	}

	// read OPC UA type definitions
	dict, err := opc.ReadTypes(typesFile)
	if err != nil {
		log.Fatalf("Failed to read type definitions: %s", err)
	}

	// read node id values
	nodeIDs, err := opc.ReadNodeIDs(nodeIDsFile)
	if err != nil {
		log.Fatalf("Failed to read node ids: %s", err)
	}

	// generate Type definitions for code generation
	types := Types(dict, nodeIDs)

	// filter types we are interested in.
	if types, err = filterTypes(types, wantTypes); err != nil {
		log.Fatal(err)
	}

	// generate the helper for encoded_object
	b, err := GenObject(pkg, types)
	if err != nil {
		log.Fatalf("Failed to generate object_gen.go: %s", err)
	}
	filename := filepath.Join(path, "object_gen.go")
	if err := WriteFile(filename, b); err != nil {
		fmt.Println(string(b))
		log.Fatalf("Failed to write %s: %s", filename, err)
	}

	// generate a file per type
	for _, t := range types {
		b, err := GenType(pkg, t)
		if err != nil {
			log.Fatalf("Failed to generate %s: %s", t.Name, err)
		}
		if b == nil {
			log.Printf("Skipping %s", t.Name)
			continue
		}

		filename := filepath.Join(path, snakeCase(t.Name)+"_gen.go")
		if err := WriteFile(filename, b); err != nil {
			fmt.Println(string(b))
			log.Fatalf("Failed to write %s: %s", filename, err)
		}
	}
}

func filterTypes(types []*Type, names []string) ([]*Type, error) {
	var x []*Type

outer:
	for _, n := range names {
		for _, t := range types {
			if t.Name == n {
				x = append(x, t)
				continue outer
			}
		}
		return nil, fmt.Errorf("cannot find type %s", n)
	}
	return x, nil
}

func snakeCase(s string) string {
	var x []rune

	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				x = append(x, '_')
			}
			x = append(x, unicode.ToLower(r))
		} else {
			x = append(x, r)
		}
	}
	return string(x)
}
