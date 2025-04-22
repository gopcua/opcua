// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uatest2

import (
	"context"
	"log"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/server/attrs"
	"github.com/gopcua/opcua/ua"
)

func startServer() *server.Server {
	var opts []server.Option
	port := 4840

	opts = append(opts,
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableSecurity("Basic128Rsa15", ua.MessageSecurityModeSign),
		server.EnableSecurity("Basic128Rsa15", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableSecurity("Basic256", ua.MessageSecurityModeSign),
		server.EnableSecurity("Basic256", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSign),
		server.EnableSecurity("Aes128_Sha256_RsaOaep", ua.MessageSecurityModeSign),
		server.EnableSecurity("Aes128_Sha256_RsaOaep", ua.MessageSecurityModeSignAndEncrypt),
		server.EnableSecurity("Aes256_Sha256_RsaPss", ua.MessageSecurityModeSign),
		server.EnableSecurity("Aes256_Sha256_RsaPss", ua.MessageSecurityModeSignAndEncrypt),
	)

	opts = append(opts,
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EnableAuthMode(ua.UserTokenTypeUserName),
		server.EnableAuthMode(ua.UserTokenTypeCertificate),
		//		server.EnableAuthWithoutEncryption(), // Dangerous and not recommended, shown for illustration only
	)

	opts = append(opts,
		server.EndPoint("localhost", port),
	)

	s := server.New(opts...)

	root_ns, _ := s.Namespace(0)
	obj_node := root_ns.Objects()

	// Create a new node namespace.  You can add namespaces before or after starting the server.
	nodeNS := server.NewNodeNameSpace(s, "NodeNamespace")
	// add it to the server.
	s.AddNamespace(nodeNS)
	nns_obj := nodeNS.Objects()
	// add the reference for this namespace's root object folder to the server's root object folder
	obj_node.AddRef(nns_obj, id.HasComponent, true)

	// Create some nodes for it.
	n := nodeNS.AddNewVariableStringNode("ro_bool", true)
	n.SetAttribute(ua.AttributeIDUserAccessLevel, &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(uint32(1))})
	nns_obj.AddRef(n, id.HasComponent, true)
	n = nodeNS.AddNewVariableStringNode("rw_bool", true)
	nns_obj.AddRef(n, id.HasComponent, true)

	n = nodeNS.AddNewVariableStringNode("ro_int32", int32(5))
	n.SetAttribute(ua.AttributeIDUserAccessLevel, &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(uint32(1))})
	nns_obj.AddRef(n, id.HasComponent, true)
	n = nodeNS.AddNewVariableStringNode("rw_int32", int32(5))
	nns_obj.AddRef(n, id.HasComponent, true)

	var3 := server.NewNode(
		ua.NewStringNodeID(nodeNS.ID(), "NoPermVariable"), // you can use whatever node id you want here, whether it's numeric, string, guid, etc...
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: server.DataValueFromValue(attrs.BrowseName("NoPermVariable")),
			ua.AttributeIDNodeClass:  server.DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return server.DataValueFromValue(int32(742)) },
	)
	nodeNS.AddNode(var3)
	nns_obj.AddRef(var3, id.HasComponent, true)

	var4 := server.NewNode(
		ua.NewStringNodeID(nodeNS.ID(), "ReadWriteVariable"), // you can use whatever node id you want here, whether it's numeric, string, guid, etc...
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDAccessLevel:     server.DataValueFromValue(byte(ua.AccessLevelTypeCurrentRead | ua.AccessLevelTypeCurrentWrite)),
			ua.AttributeIDUserAccessLevel: server.DataValueFromValue(byte(ua.AccessLevelTypeCurrentRead | ua.AccessLevelTypeCurrentWrite)),
			ua.AttributeIDBrowseName:      server.DataValueFromValue(attrs.BrowseName("ReadWriteVariable")),
			ua.AttributeIDNodeClass:       server.DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return server.DataValueFromValue(12.34) },
	)
	nodeNS.AddNode(var4)
	nns_obj.AddRef(var4, id.HasComponent, true)

	var5 := server.NewNode(
		ua.NewStringNodeID(nodeNS.ID(), "ReadOnlyVariable"), // you can use whatever node id you want here, whether it's numeric, string, guid, etc...
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDAccessLevel:     server.DataValueFromValue(byte(ua.AccessLevelTypeCurrentRead)),
			ua.AttributeIDUserAccessLevel: server.DataValueFromValue(byte(ua.AccessLevelTypeCurrentRead)),
			ua.AttributeIDBrowseName:      server.DataValueFromValue(attrs.BrowseName("ReadOnlyVariable")),
			ua.AttributeIDNodeClass:       server.DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return server.DataValueFromValue(9.87) },
	)
	nodeNS.AddNode(var5)
	nns_obj.AddRef(var5, id.HasComponent, true)

	var6 := server.NewNode(
		ua.NewStringNodeID(nodeNS.ID(), "NoAccessVariable"), // you can use whatever node id you want here, whether it's numeric, string, guid, etc...
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDAccessLevel:     server.DataValueFromValue(byte(ua.AccessLevelTypeNone)),
			ua.AttributeIDUserAccessLevel: server.DataValueFromValue(byte(ua.AccessLevelTypeNone)),
			ua.AttributeIDBrowseName:      server.DataValueFromValue(attrs.BrowseName("NoAccessVariable")),
			ua.AttributeIDNodeClass:       server.DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return server.DataValueFromValue(55.43) },
	)
	nodeNS.AddNode(var6)
	nns_obj.AddRef(var6, id.HasComponent, true)

	// Create a new node namespace.  You can add namespaces before or after starting the server.
	gopcuaNS := server.NewNodeNameSpace(s, "http://gopcua.com/")
	// add it to the server.
	s.AddNamespace(gopcuaNS)
	nns_obj = gopcuaNS.Objects()
	// add the reference for this namespace's root object folder to the server's root object folder
	obj_node.AddRef(nns_obj, id.HasComponent, true)

	// Create a new node namespace.  You can add namespaces before or after starting the server.
	// Start the server
	if err := s.Start(context.Background()); err != nil {
		log.Fatalf("Error starting server, exiting: %s", err)
	}
	return s
}
