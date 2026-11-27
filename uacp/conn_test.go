// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

func TestConn(t *testing.T) {
	t.Run("server exists ", func(t *testing.T) {
		ep := "opc.tcp://127.0.0.1:4840/foo/bar"
		ln, err := Listen(context.Background(), ep, nil)
		require.NoError(t, err)
		defer ln.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		acceptErr := make(chan error, 1)
		go func() {
			c, err := ln.Accept(ctx)
			if err != nil {
				acceptErr <- err
				return
			}
			defer c.Close()
			close(done)
		}()

		if _, err = Dial(ctx, ep); err != nil {
			t.Error(err)
		}

		select {
		case <-done:
		case err := <-acceptErr:
			require.Fail(t, "accept fail: %v", err)
		case <-time.After(time.Second):
			require.Fail(t, "timed out")
		}
	})

	t.Run("Address resolves, but does not implement a opcua-server", func(t *testing.T) {
		ep := "opc.tcp://example.com:56789"

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := Dial(ctx, ep)
		var operr *net.OpError
		if errors.As(err, &operr) && !operr.Timeout() {
			t.Error(err)
		}
	})
}

func TestClientWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(context.Background(), ep, nil)
	require.NoError(t, err, "Listen failed")
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var srvConn *Conn
	done := make(chan int)
	acceptErr := make(chan error, 1)
	go func() {
		defer ln.Close()
		var err error
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			acceptErr <- err
			return
		}
		done <- 0
	}()

	cliConn, err := Dial(ctx, ep)
	require.NoError(t, err, "Dial failed")

	for {
		select {
		case _, ok := <-done:
			require.True(t, ok, "failed to setup secure channel")
			goto NEXT
		case err := <-acceptErr:
			require.Fail(t, "accept fail: %v", err)
		case <-time.After(time.Second):
			require.Fail(t, "timed out")
		}
	}

NEXT:
	msg := &Message{Data: []byte{0xde, 0xad, 0xbe, 0xef}}
	err = cliConn.Send("MSGF", msg)
	require.NoError(t, err, "Send failed")

	got, err := srvConn.Receive()
	require.NoError(t, err, "Receive failed")

	got = got[hdrlen:]

	want, err := msg.Encode()
	require.NoError(t, err, "Encode failed")

	require.Equal(t, want, got)
}

func TestServerWrite(t *testing.T) {
	ep := "opc.tcp://127.0.0.1:4840/foo/bar"
	ln, err := Listen(context.Background(), ep, nil)
	require.NoError(t, err, "Listen failed")
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var srvConn *Conn
	done := make(chan int)
	acceptErr := make(chan error, 1)
	go func() {
		defer ln.Close()
		var err error
		srvConn, err = ln.Accept(ctx)
		if err != nil {
			acceptErr <- err
			return
		}
		done <- 0
	}()

	cliConn, err := Dial(ctx, ep)
	require.NoError(t, err, "Dial failed")

	for {
		select {
		case _, ok := <-done:
			require.True(t, ok, "failed to setup secure channel")
			goto NEXT
		case err := <-acceptErr:
			require.Fail(t, "accept fail: %v", err)
		case <-time.After(time.Second):
			require.Fail(t, "timed out")
		}
	}

NEXT:
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	_, err = srvConn.Write(want)
	require.NoError(t, err, "Write failed")

	got := make([]byte, cliConn.ReceiveBufSize())
	n, err := cliConn.Read(got)
	require.NoError(t, err, "Read failed")

	got = got[:n]
	require.Equal(t, want, got)
}

// A name whose first address does not answer still reaches the server on the
// next one: localhost often resolves to ::1 before 127.0.0.1, and a server that
// listens on IPv4 only must still be found through it.
func TestDialTriesEveryAddressOfAName(t *testing.T) {
	ln, err := Listen(context.Background(), "opc.tcp://127.0.0.1:0", nil)
	require.NoError(t, err)
	defer ln.Close()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accepted := make(chan error, 1)
	go func() {
		c, err := ln.Accept(ctx)
		if err == nil {
			c.Close()
		}
		accepted <- err
	}()

	c, err := Dial(ctx, "opc.tcp://localhost:"+port+"/foo/bar")
	require.NoError(t, err)
	defer c.Close()
	require.Equal(t, "127.0.0.1", c.RemoteAddr().(*net.TCPAddr).IP.String())
	require.NoError(t, <-accepted)
}

// readUACPRaw reads one UACP message from c and returns it with the header.
// The fake server below uses it to see the Hello a client sends.
func readUACPRaw(c net.Conn) ([]byte, error) {
	hdr := make([]byte, hdrlen)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return nil, err
	}
	var h Header
	if _, err := h.Decode(hdr); err != nil {
		return nil, err
	}
	b := make([]byte, h.MessageSize)
	copy(b, hdr)
	if _, err := io.ReadFull(c, b[hdrlen:]); err != nil {
		return nil, err
	}
	return b, nil
}

// writeUACPRaw writes one UACP message to c, the way Conn.Send does, for a
// connection that is not a Conn.
func writeUACPRaw(c net.Conn, typ string, msg interface{}) error {
	body, err := ua.Encode(msg)
	if err != nil {
		return err
	}
	h := Header{MessageType: typ[:3], ChunkType: typ[3], MessageSize: uint32(len(body) + hdrlen)}
	hdr, err := h.Encode()
	if err != nil {
		return err
	}
	_, err = c.Write(append(hdr, body...))
	return err
}

// fakeAckServer listens on a local address and answers the first client's
// Hello with ack, handing the Hello it read back on the returned channel.
// afterACK, if not nil, runs on the accepted connection once the
// Acknowledge is written; the connection closes when it returns.
func fakeAckServer(t *testing.T, ack *Acknowledge, afterACK func(net.Conn)) (addr string, hello <-chan *Hello) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	hellos := make(chan *Hello, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()

		b, err := readUACPRaw(c)
		if err != nil {
			return
		}
		hel := new(Hello)
		if _, err := hel.Decode(b[hdrlen:]); err != nil {
			return
		}
		hellos <- hel

		if err := writeUACPRaw(c, "ACKF", ack); err != nil {
			return
		}
		if afterACK != nil {
			afterACK(c)
		}
	}()
	return ln.Addr().String(), hellos
}

// The client keeps the receive buffer its Hello offered, whatever the
// Acknowledge says: the value is unauthenticated before the secure channel
// exists, so it must not decide an allocation. An offer below the spec
// minimum still reads with the minimum, which the test below covers.
func TestHandshakeDoesNotGrowTheReceiveBufferPastTheHello(t *testing.T) {
	for _, tt := range []struct {
		name string
		ack  *Acknowledge
		want uint32
	}{
		{
			name: "larger than the Hello",
			ack: &Acknowledge{
				ReceiveBufSize: 0xffffffff, // no Hello offered this
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			want: DefaultClientACK.ReceiveBufSize,
		},
		{
			name: "smaller than the Hello",
			ack: &Acknowledge{
				ReceiveBufSize: 8192,
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			want: 8192,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			addr, hello := fakeAckServer(t, tt.ack, func(c net.Conn) {
				// small enough for any buffer: the one the client settled
				// on must still read it
				_ = writeUACPRaw(c, "MSGF", &Message{Data: []byte{0xde, 0xad, 0xbe, 0xef}})
			})

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c, err := Dial(ctx, "opc.tcp://"+addr+"/foo/bar")
			require.NoError(t, err)
			defer c.Close()

			select {
			case hel := <-hello:
				require.Equal(t, DefaultClientACK.ReceiveBufSize, hel.ReceiveBufSize, "the Hello offered the default")
			case <-time.After(5 * time.Second):
				require.Fail(t, "the fake server never saw the Hello")
			}
			require.Equal(t, tt.want, c.ReceiveBufSize(), "the client kept the receive buffer it offered")

			// the receive buffer it settled on still works
			got, err := c.Receive()
			require.NoError(t, err)
			want, err := (&Message{Data: []byte{0xde, 0xad, 0xbe, 0xef}}).Encode()
			require.NoError(t, err)
			require.Equal(t, want, got[hdrlen:])
		})
	}
}

// The client never reads with less than the 8192-byte minimum OPC 10000-6
// §7.1.2.4 sets for any buffer, even when its Hello offered less: opc-plc
// answers an offer of 4096 with 8192 and then sends chunks that size, which
// a 4096-byte buffer cannot read.
func TestHandshakeNeverReadsWithLessThanTheSpecMinimum(t *testing.T) {
	clientACK := *DefaultClientACK
	clientACK.ReceiveBufSize = 4096

	// a body of 8184 bytes makes an 8192-byte message with the header
	msg := &Message{Data: make([]byte, minLargeAckBufSize-hdrlen)}
	addr, hello := fakeAckServer(t, &Acknowledge{
		ReceiveBufSize: minLargeAckBufSize,
		SendBufSize:    0xffff,
		MaxMessageSize: 2 * MB,
		MaxChunkCount:  DefaultMaxChunkCount,
	}, func(c net.Conn) {
		_ = writeUACPRaw(c, "MSGF", msg)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d := &Dialer{ClientACK: &clientACK}
	c, err := d.Dial(ctx, "opc.tcp://"+addr+"/foo/bar")
	require.NoError(t, err)
	defer c.Close()

	select {
	case hel := <-hello:
		require.Equal(t, uint32(4096), hel.ReceiveBufSize, "the Hello offered 4096")
	case <-time.After(5 * time.Second):
		require.Fail(t, "the fake server never saw the Hello")
	}
	require.Equal(t, uint32(minLargeAckBufSize), c.ReceiveBufSize(), "the client reads with the 8192-byte spec minimum, not the 4096 offered")

	got, err := c.Receive()
	require.NoError(t, err, "the 8192-byte message must fit the buffer the client reads with")
	want, err := msg.Encode()
	require.NoError(t, err)
	require.Equal(t, want, got[hdrlen:])
}

// The handshake refuses an Acknowledge whose buffer sizes fall below the
// floor OPC 10000-6 §7.1.2.4 Table 75 sets for them, computed from the
// matching Hello size (see minAckBufSize): below the floor the buffer
// Receive allocates cannot hold the 8-byte UACP header, so the first chunk
// read panics. The cases below cover the zero sentinel, both sides of the
// 8192 floor, and both sides of the 1024 floor a small Hello lowers it to.
func TestHandshakeRefusesAnAcknowledgeBelowTheSpecFloor(t *testing.T) {
	defaultACK := func() *Acknowledge {
		ack := *DefaultClientACK
		return &ack
	}

	for _, tt := range []struct {
		name       string
		clientACK  *Acknowledge // what the client offers in its Hello
		ack        *Acknowledge // what the server answers with
		wantDialOK bool
		wantErr    string // the floor refusal, when wantDialOK is false
		wantRecv   uint32 // ReceiveBufSize the client settles on
		wantSend   uint32 // SendBufSize the client settles on
	}{
		{
			name:      "receive buffer 0",
			clientACK: defaultACK(),
			ack: &Acknowledge{
				ReceiveBufSize: 0,
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: false,
			wantErr: "uacp: server receive buffer 0 is below the 8192 byte minimum " +
				"OPC 10000-6 §7.1.2.4 requires for a Hello send buffer of 65535 bytes",
		},
		{
			name:      "receive buffer 8191, one byte below the floor",
			clientACK: defaultACK(),
			ack: &Acknowledge{
				ReceiveBufSize: 8191,
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: false,
			wantErr: "uacp: server receive buffer 8191 is below the 8192 byte minimum " +
				"OPC 10000-6 §7.1.2.4 requires for a Hello send buffer of 65535 bytes",
		},
		{
			name:      "receive buffer 8192, at the floor",
			clientACK: defaultACK(),
			ack: &Acknowledge{
				ReceiveBufSize: 8192,
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: true,
			wantRecv:   8192,
			wantSend:   0xffff,
		},
		{
			name:      "send buffer 8192, at the floor",
			clientACK: defaultACK(),
			ack: &Acknowledge{
				ReceiveBufSize: 0xffff,
				SendBufSize:    8192,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: true,
			wantRecv:   0xffff,
			wantSend:   8192,
		},
		{
			// A Hello SendBufferSize below 8192 lowers the floor for the
			// server's receive buffer to 1024 bytes.
			name: "receive buffer 1024, at the floor of a small Hello send buffer",
			clientACK: func() *Acknowledge {
				ack := *DefaultClientACK
				ack.SendBufSize = 2000
				return &ack
			}(),
			ack: &Acknowledge{
				ReceiveBufSize: 1024,
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: true,
			wantRecv:   1024,
			wantSend:   0xffff,
		},
		{
			name: "receive buffer 1023, below the floor of a small Hello send buffer",
			clientACK: func() *Acknowledge {
				ack := *DefaultClientACK
				ack.SendBufSize = 2000
				return &ack
			}(),
			ack: &Acknowledge{
				ReceiveBufSize: 1023,
				SendBufSize:    0xffff,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: false,
			wantErr: "uacp: server receive buffer 1023 is below the 1024 byte minimum " +
				"OPC 10000-6 §7.1.2.4 requires for a Hello send buffer of 2000 bytes",
		},
		{
			// A Hello ReceiveBufferSize below 8192 lowers the floor for the
			// server's send buffer to 1024 bytes.
			name: "send buffer 1024, at the floor of a small Hello receive buffer",
			clientACK: func() *Acknowledge {
				ack := *DefaultClientACK
				ack.ReceiveBufSize = 2000
				return &ack
			}(),
			ack: &Acknowledge{
				ReceiveBufSize: 0xffff,
				SendBufSize:    1024,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: true,
			wantRecv:   minLargeAckBufSize, // clamped down to the spec minimum, not to the Hello's small offer
			wantSend:   1024,
		},
		{
			name: "send buffer 1023, below the floor of a small Hello receive buffer",
			clientACK: func() *Acknowledge {
				ack := *DefaultClientACK
				ack.ReceiveBufSize = 2000
				return &ack
			}(),
			ack: &Acknowledge{
				ReceiveBufSize: 0xffff,
				SendBufSize:    1023,
				MaxMessageSize: 2 * MB,
				MaxChunkCount:  DefaultMaxChunkCount,
			},
			wantDialOK: false,
			wantErr: "uacp: server send buffer 1023 is below the 1024 byte minimum " +
				"OPC 10000-6 §7.1.2.4 requires for a Hello receive buffer of 2000 bytes",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			addr, _ := fakeAckServer(t, tt.ack, nil)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			d := &Dialer{ClientACK: tt.clientACK}
			c, err := d.Dial(ctx, "opc.tcp://"+addr+"/foo/bar")
			if !tt.wantDialOK {
				require.Contains(t, err.Error(), tt.wantErr, "the handshake must refuse an Acknowledge below the spec's floor, with the floor's own error")
				require.Nil(t, c)
				return
			}
			require.NoError(t, err)
			defer c.Close()
			require.Equal(t, tt.wantRecv, c.ReceiveBufSize(), "the client kept the receive buffer it settled on")
			require.Equal(t, tt.wantSend, c.SendBufSize(), "the client kept the send buffer it settled on")
		})
	}
}
