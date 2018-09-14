// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Package uasc provides encoding/decoding and automated secure channel and session handling for OPC UA Secure Conversation.

To establish Secure Channel as a client, use OpenSecureChannel().

To establish Secure Channel as a server, use ListenAndAccept().

Both returns *SecureChannel, which implements net.Conn interface,
*/
package uasc
