package uasc

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/wmnsk/gopcua/ua"
	"github.com/wmnsk/gopcua/uacp"
)

func Dial(endpoint string) (*Conn, error) {
	log.Printf("Connect to %s", endpoint)
	addr := strings.TrimPrefix(endpoint, "opc.tcp://")
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		id:  1,
		c:   c,
		ack: &uacp.Acknowledge{ReceiveBufSize: 0xffff, SendBufSize: 0xffff},
	}

	log.Printf("conn %d: start HEL/ACK handshake", conn.id)
	if err := conn.handshake(endpoint); err != nil {
		log.Printf("conn %d: HEL/ACK handshake failed: %s", conn.id, err)
		conn.Close()
		return nil, err
	}
	return conn, nil
}

type Conn struct {
	id  int
	c   net.Conn
	ack *uacp.Acknowledge
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
	m := &uacp.Hello{
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

	ack := new(uacp.Acknowledge)
	if err := ua.Decode(b, ack); err != nil {
		return fmt.Errorf("decode ACK failed: %s", err)
	}

	if ack.Version != 0 {
		return fmt.Errorf("invalid version %d", ack.Version)
	}
	a.ack = ack
	log.Printf("conn %d: recv ACK:%v", a.id, ack)
	return nil
}

// recv receives a message from the stream and returns it without the header.
func (a *Conn) recv() ([]byte, error) {
	const hdrlen = 8

	hdr := make([]byte, hdrlen)
	_, err := io.ReadFull(a.c, hdr)
	if err != nil {
		return nil, fmt.Errorf("hdr read faled: %s", err)
	}

	var h uacp.Header
	if err := ua.Decode(hdr, &h); err != nil {
		return nil, fmt.Errorf("hdr decode failed: %s", err)
	}

	if h.MessageSize > a.ack.ReceiveBufSize {
		return nil, fmt.Errorf("packet too large: %d > %d bytes", h.MessageSize, a.ack.ReceiveBufSize)
	}

	b := make([]byte, h.MessageSize-hdrlen)
	if _, err := io.ReadFull(a.c, b); err != nil {
		return nil, fmt.Errorf("read msg failed: %s", err)
	}

	log.Printf("conn %d: recv %s%c with %d bytes", a.id, h.MessageType, h.ChunkType, len(b))
	return b, nil
}

func (a *Conn) send(typ string, msg interface{}) error {
	if len(typ) != 4 {
		return fmt.Errorf("invalid msg type: %s", typ)
	}

	body, err := ua.Encode(msg)
	if err != nil {
		return fmt.Errorf("encode msg failed: %s", err)
	}

	h := uacp.Header{
		MessageType: typ[:3],
		ChunkType:   typ[3],
		MessageSize: uint32(len(body) + 8),
	}

	if h.MessageSize > a.ack.SendBufSize {
		return fmt.Errorf("send packet too large: %d > %d bytes", h.MessageSize, a.ack.SendBufSize)
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
