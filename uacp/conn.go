package uacp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/wmnsk/gopcua/ua"
	"github.com/wmnsk/gopcua/utils"
)

// connid stores the current connection id. updated with atomic.AddUint32
var connid uint32

// nextid returns the next connection id
func nextid() uint32 {
	return atomic.AddUint32(&connid, 1)
}

func Dial(ctx context.Context, endpoint string) (*Conn, error) {
	log.Printf("Connect to %s", endpoint)
	network, raddr, err := utils.ResolveEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	c, err := net.DialTCP(network, nil, raddr)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		id:         nextid(),
		c:          c,
		rcvBufSize: 0xffff,
		sndBufSize: 0xffff,
	}

	log.Printf("conn %d: start HEL/ACK handshake", conn.id)
	if err := conn.handshake(endpoint); err != nil {
		log.Printf("conn %d: HEL/ACK handshake failed: %s", conn.id, err)
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// Listener is a OPC UA Connection Protocol network listener.
type Listener struct {
	l          net.Listener
	endpoint   string
	rcvBufSize uint32
	sndBufSize uint32
}

// Listen acts like net.Listen for OPC UA Connection Protocol networks.
//
// Currently the endpoint can only be specified in "opc.tcp://<addr[:port]>/path" format.
//
// If the IP field of laddr is nil or an unspecified IP address, Listen listens
// on all available unicast and anycast IP addresses of the local system.
// If the Port field of laddr is 0, a port number is automatically chosen.
func Listen(endpoint string, rcvBufSize uint32) (*Listener, error) {
	network, laddr, err := utils.ResolveEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	l, err := net.Listen(network, laddr.String())
	if err != nil {
		return nil, err
	}
	return &Listener{
		l:          l,
		endpoint:   endpoint,
		rcvBufSize: rcvBufSize,
		sndBufSize: 0xffff,
	}, nil
}

// Accept accepts the next incoming call and returns the new connection.
//
// The first param ctx is to be passed to monitor(), which monitors and handles
// incoming messages automatically in another goroutine.
func (l *Listener) Accept(ctx context.Context) (*Conn, error) {
	c, err := l.l.Accept()
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		id:         nextid(),
		c:          c,
		rcvBufSize: 0xffff,
		sndBufSize: 0xffff,
	}
	if err := conn.srvhandshake(l.endpoint); err != nil {
		c.Close()
		return nil, err
	}
	return conn, nil
}

// Close closes the Listener.
func (l *Listener) Close() error {
	return l.l.Close()
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.l.Addr()
}

// Endpoint returns the listener's EndpointURL.
func (l *Listener) Endpoint() string {
	return l.endpoint
}

type Conn struct {
	id         uint32
	c          net.Conn
	rcvBufSize uint32
	sndBufSize uint32
}

func (a *Conn) ID() uint32 {
	return a.id
}

func (a *Conn) Close() error {
	log.Printf("conn %d: close", a.id)
	return a.c.Close()
}

func (a *Conn) Read(b []byte) (int, error) {
	return a.c.Read(b)
}

func (a *Conn) Write(b []byte) (int, error) {
	return a.c.Write(b)
}

func (a *Conn) SetDeadline(t time.Time) error {
	return a.c.SetDeadline(t)
}

func (a *Conn) SetReadDeadline(t time.Time) error {
	return a.c.SetReadDeadline(t)
}

func (a *Conn) SetWriteDeadline(t time.Time) error {
	return a.c.SetWriteDeadline(t)
}

func (a *Conn) LocalAddr() net.Addr {
	return a.c.LocalAddr()
}

func (a *Conn) RemoteAddr() net.Addr {
	return a.c.RemoteAddr()
}

func (a *Conn) handshake(endpoint string) error {
	m := &Hello{
		Version:        0,
		ReceiveBufSize: 0xffff,
		SendBufSize:    0xffff,
		MaxMessageSize: 0,
		MaxChunkCount:  0,
		EndPointURL:    endpoint,
	}

	if err := a.send("HELF", m); err != nil {
		return err
	}

	b, err := a.recv()
	if err != nil {
		return err
	}

	msgtyp := string(b[:4])
	if msgtyp != "ACKF" {
		return fmt.Errorf("got %s want ACK", msgtyp)
	}

	ack := new(Acknowledge)
	if _, err := ua.Decode(b[hdrlen:], ack); err != nil {
		return fmt.Errorf("decode ACK failed: %s", err)
	}

	if ack.Version != 0 {
		return fmt.Errorf("invalid version %d", ack.Version)
	}
	a.rcvBufSize = ack.ReceiveBufSize
	a.sndBufSize = ack.SendBufSize
	log.Printf("conn %d: recv ACK:%v", a.id, ack)
	return nil
}

func (a *Conn) srvhandshake(endpoint string) error {
	b, err := a.recv()
	if err != nil {
		a.sendError(BadTCPInternalError)
		return err
	}

	// HEL or RHE?
	msgtyp := string(b[:4])
	msg := b[hdrlen:]
	switch msgtyp {
	case "HELF":
		hel := new(Hello)
		if _, err := ua.Decode(msg, hel); err != nil {
			a.sendError(BadTCPInternalError)
			return err
		}
		if hel.EndPointURL != endpoint {
			a.sendError(BadTCPEndpointURLInvalid)
			return fmt.Errorf("invalid endpoint url %s", hel.EndPointURL)
		}
		ack := &Acknowledge{
			Version:        0,
			ReceiveBufSize: a.rcvBufSize,
			SendBufSize:    a.sndBufSize,
			MaxMessageSize: 0, // what is a sensible default?
			MaxChunkCount:  1000,
		}
		if err := a.send("ACKF", ack); err != nil {
			a.sendError(BadTCPInternalError)
			return err
		}
		return nil

	case "RHEF":
		rhe := new(ReverseHello)
		if _, err := ua.Decode(msg, rhe); err != nil {
			a.sendError(BadTCPInternalError)
			return err
		}
		if rhe.EndPointURL != endpoint {
			a.sendError(BadTCPEndpointURLInvalid)
			return fmt.Errorf("invalid endpoint url %s", rhe.EndPointURL)
		}
		log.Printf("conn %d: connecting to %s", a.id, rhe.ServerURI)
		a.c.Close()
		c, err := Dial(context.Background(), rhe.ServerURI)
		if err != nil {
			return err
		}
		a.c = c
		return nil

	default:
		a.sendError(BadTCPInternalError)
		return fmt.Errorf("invalid handshake packet %q", msgtyp)
	}
}

func (a *Conn) sendError(code uint32) error {
	return a.send("ERRF", &Error{Error: code})
}

// hdrlen is the size of the uacp header
const hdrlen = 8

// recv receives a message from the stream and returns it without the header.
func (a *Conn) recv() ([]byte, error) {
	hdr := make([]byte, hdrlen)
	_, err := io.ReadFull(a.c, hdr)
	if err != nil {
		return nil, fmt.Errorf("hdr read faled: %s", err)
	}

	var h Header
	if _, err := ua.Decode(hdr, &h); err != nil {
		return nil, fmt.Errorf("hdr decode failed: %s", err)
	}

	if h.MessageSize > a.rcvBufSize {
		return nil, fmt.Errorf("packet too large: %d > %d bytes", h.MessageSize, a.rcvBufSize)
	}

	b := make([]byte, h.MessageSize-hdrlen)
	if _, err := io.ReadFull(a.c, b); err != nil {
		return nil, fmt.Errorf("read msg failed: %s", err)
	}

	log.Printf("conn %d: recv %s%c with %d bytes", a.id, h.MessageType, h.ChunkType, len(b))
	return append(hdr, b...), nil
}

func (a *Conn) send(typ string, msg interface{}) error {
	if len(typ) != 4 {
		return fmt.Errorf("invalid msg type: %s", typ)
	}

	body, err := ua.Encode(msg)
	if err != nil {
		return fmt.Errorf("encode msg failed: %s", err)
	}

	h := Header{
		MessageType: typ[:3],
		ChunkType:   typ[3],
		MessageSize: uint32(len(body) + 8),
	}

	if h.MessageSize > a.sndBufSize {
		return fmt.Errorf("send packet too large: %d > %d bytes", h.MessageSize, a.sndBufSize)
	}

	hdr, err := h.Encode()
	if err != nil {
		return fmt.Errorf("encode hdr failed: %s", err)
	}

	b := append(hdr, body...)
	if _, err := a.c.Write(b); err != nil {
		return fmt.Errorf("write failed: %s", err)
	}
	log.Printf("conn %d: sent %s with %d bytes", a.id, typ, len(b))

	return nil
}
