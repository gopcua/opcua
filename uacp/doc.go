// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Package uacp provides encoding/decoding and connection handling for OPC UA Connection Protocol.

To establish the connection as a client, create Client with NewClient() and call Dial() method.

To wait for the client to connect to, create Server with NewServer() and call Listen() and Accept() methods.

The connection(=returned object *Conn) can be used to read, write, print addresses, etc.
in the same way as other kind of Conn which implements net.Conn interface.
*/
package uacp
