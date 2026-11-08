//go:build conformance

// Package securitypolicy holds the gopcua security policy tests: one
// behaviour per file, with the clause it checks in the file's opening
// comment. This file holds the fixture they all run against; it is not
// a row.
//
// The fixture is a scripted TCP peer, built on net and not on gopcua's
// uacp server: it accepts one connection, reads the client's HEL, answers
// the ACK, reads the next frame - the OpenSecureChannel request the rows
// score - records its bytes, and closes the connection without ever
// answering it. The rows score what the opcua.Client's chosen
// SecurityPolicy hands the secure channel inside that frame. The peer
// uses gopcua's uacp types to frame the ACK and to read whole messages
// off the wire, which is not the library grading itself: the bytes the
// rows score are decrypted and counted by the standard library, and the
// expected numbers come from the clauses, never from gopcua.
//
// Every helper here returns plain values. A helper does not call
// t.Errorf or t.Fatalf; a row's test function turns a non-empty reason
// string into the failure or the decline, and it, not the fixture,
// registers the teardown with t.Cleanup. The teardown closes the
// listener and the accepted connection, so a Connect left blocked by
// the library under test is released when the test ends.
//
// The package name is securitypolicy, as the directory is; the
// production code it drives lives in github.com/gopcua/opcua and its
// uasc and uapolicy packages.
package securitypolicy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/uacp"
)

// handshake bounds the opening exchange: how long the peer waits for the
// client's frames, and the deadline on the context launchClient gives
// Connect. The bound is not a timing assumption: a correct client sends
// its HEL and its OPN in milliseconds, so the bound exists to let a slow
// machine finish, not to wait out a broken one; the teardown releases a
// client that never sends, and the context's deadline releases a Connect
// that stays blocked. README.md says this of the bound the way
// ../transport/README.md says it of unblock.
const handshake = 10 * time.Second

// peerAck is the Acknowledge the scripted peer answers the HEL with:
// protocol version 0, 65535-byte buffers in both directions, and no
// message-size or chunk-count limit.
var peerAck = &uacp.Acknowledge{
	Version:        0,
	ReceiveBufSize: 65535,
	SendBufSize:    65535,
}

// scriptedPeer is the scripted TCP peer: it accepts one connection, reads
// the HEL, answers the ACK, records the next frame - the OPN - and
// closes the connection without answering it.
type scriptedPeer struct {
	listener net.Listener

	mu        sync.Mutex
	conn      net.Conn
	closed    bool   // set by teardown; serve closes the connection itself when set
	offScript string // the message type of a frame that arrived where the script expected another

	opn chan []byte // the recorded OPN frame, delivered once
}

// startScriptedPeer brings the peer up on the loopback interface, on a
// free port the kernel chooses - never port 4840, which other suites
// bind. It returns the peer, a teardown for the listener and the
// accepted connection, and "" - or, when the peer could not be started,
// the reason the run cannot be scored.
func startScriptedPeer() (*scriptedPeer, func(), string) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, func() {}, fmt.Sprintf("listen on the loopback interface: %v", err)
	}
	p := &scriptedPeer{
		listener: l,
		opn:      make(chan []byte, 1),
	}
	go p.serve()
	teardown := func() {
		_ = l.Close()
		p.mu.Lock()
		defer p.mu.Unlock()
		p.closed = true
		if p.conn != nil {
			_ = p.conn.Close()
		}
	}
	return p, teardown, ""
}

// addr is the host:port the peer listens on.
func (p *scriptedPeer) addr() string {
	return p.listener.Addr().String()
}

// serve plays the script: accept one connection, read the HEL, answer the
// ACK, read the OPN, deliver it on the channel, and close. A frame whose
// message type is neither HELF nor OPNF is recorded in offScript instead:
// it is determinately the client's bytes, and awaitOpenSecureChannel
// reports it as gopcua's own behaviour. Any other failure - no
// connection, a client that goes away mid-exchange - leaves the channel
// empty, and the check declines. If teardown runs between the Accept and
// this goroutine taking the mutex, teardown never saw the connection, so
// serve closes it here.
func (p *scriptedPeer) serve() {
	c, err := p.listener.Accept()
	if err != nil {
		return // the listener was closed in teardown
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		_ = c.Close()
		return
	}
	p.conn = c
	p.mu.Unlock()

	uc, err := uacp.NewConn(c.(*net.TCPConn), peerAck)
	if err != nil {
		return // the peer cannot frame the connection; the check declines
	}

	hel, err := uc.Receive()
	if err != nil {
		return // the client went away; the check declines
	}
	if string(hel[:4]) != "HELF" {
		p.noteOffScript(hel[:4])
		return
	}

	if err := uc.Send("ACKF", peerAck); err != nil {
		return // the client went away; the check declines
	}

	frame, err := uc.Receive()
	if err != nil {
		return // the client went away; the check declines
	}
	if string(frame[:4]) != "OPNF" {
		p.noteOffScript(frame[:4])
		return
	}

	// The OPN is the row's evidence. The channel is buffered, so the
	// deliver never blocks; the close is what makes the client's Connect
	// return its expected error.
	p.opn <- frame
	_ = uc.Close()
}

// noteOffScript records the message type of a frame that arrived where
// the script expected a HEL or an OPN, once.
func (p *scriptedPeer) noteOffScript(msgType []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.offScript == "" {
		p.offScript = string(msgType)
	}
}

// awaitOpenSecureChannel is the setup check the rows share: it waits up
// to `within` for the peer to record the OPN frame the client sent, and
// returns the frame with "". Otherwise it returns the reason the run
// cannot be scored, and which side caused the failed check: a frame with
// a message type that is neither HELF nor OPNF can only be the client's
// bytes (true), and the row prints the reason with
// "gopcua's own behaviour made this run unscoreable: %s" and t.Errorf,
// then returns; everything else - no connection, no HEL, no OPN within
// `within` - can close on either side, and is a decline the row prints
// with "could not set the test up: %s".
func (p *scriptedPeer) awaitOpenSecureChannel(within time.Duration) ([]byte, string, bool) {
	select {
	case frame := <-p.opn:
		return frame, "", false
	case <-time.After(within):
	}
	p.mu.Lock()
	offScript := p.offScript
	p.mu.Unlock()
	if offScript != "" {
		return nil, fmt.Sprintf("a frame arrived with message type %q, "+
			"where the script expected HELF or OPNF", offScript), true
	}
	return nil, fmt.Sprintf("no OPN frame arrived within %v", within), false
}

// launchClient builds the opcua.Client under test and runs its Connect on
// its own goroutine, under a context bounded by handshake. The row passes
// the security policy options it scores; the harness adds
// AutoReconnect(false), so a failed connect is a returned error and not
// a retry loop. The peer never answers the OPN and closes the connection
// once it has recorded it, so Connect is expected to return an error;
// the error is delivered on the returned channel and no row asserts it.
// The channel is buffered, so the goroutine never blocks on a row that
// has stopped reading. It returns the client, the channel, and "" - or,
// when the client could not be built, the reason the run cannot be
// scored. A "build the client" reason can come from either side - a
// certificate the harness minted, or the library rejecting valid
// options - so per ../README.md the row declines on it, printing it
// with "could not set the test up".
func launchClient(endpoint string, opts ...opcua.Option) (*opcua.Client, <-chan error, string) {
	client, err := opcua.NewClient(endpoint,
		append([]opcua.Option{opcua.AutoReconnect(false)}, opts...)...)
	if err != nil {
		return nil, nil, fmt.Sprintf("build the client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), handshake)
	connectErr := make(chan error, 1)
	go func() {
		connectErr <- client.Connect(ctx)
		cancel()
	}()
	return client, connectErr, ""
}

// mintCert mints a self-signed 2048-bit RSA key and X.509 certificate in
// memory, for the common name it is given. Nothing is written to disk
// and no key material is committed: every run mints its own pair for the
// client and its own for the peer. It returns the key, the DER
// certificate, and "" - or, when the minting failed, the reason the run
// cannot be scored.
func mintCert(commonName string) (key *rsa.PrivateKey, cert []byte, why string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Sprintf("generate a 2048-bit RSA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
	}
	cert, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Sprintf("create the certificate: %v", err)
	}
	return key, cert, ""
}
