// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Package uacp provides encoding/decoding and automated connection handling for OPC UA Connection Protocol.

To establish the connection(=get *Conn) as a client, call Dial() method.

To wait for the client to connect to, call Listen() method, and to establish connection(=get *Conn)
with Accept() method.

Once you get *Conn, you can Read(), Write(), and print Local/RemoteAddr(), etc.
in the same way as other kind of Conn which implements net.Conn interface.

In uacp, *Conn also implements Local/RemoteEndpoint() methods which returns EndpointURL of
client or server.

The data on top of UACP connection is passed as it is as long as the connection is established.
In other words, uacp never cares the data even if it seems invalid. Users of this package should
check the data to make sure it is what they want or not.
*/
package uacp
