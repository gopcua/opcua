package uasc

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
)

type Response struct {
	V   interface{}
	Err error
}

type SecureChannel struct {
	EndpointURL string

	c         *Conn
	cfg       *Config
	reqHeader *services.RequestHeader

	// mu guards handler which contains the response channels
	// for the outstanding requests. The key is the request
	// handle which is part of the Request and Response headers.
	mu      sync.Mutex
	handler map[uint32]chan Response

	// ctx controls the request loop and terminates it when
	// cancelled.
	ctx    context.Context
	cancel func()
}

func NewSecureChannel(c *Conn, cfg *Config) *SecureChannel {
	if cfg == nil {
		cfg = NewClientConfigSecurityNone(3333, 3600000)
	}

	// always reset the secure channel id
	cfg.SecureChannelID = 0

	reqHeader := &services.RequestHeader{
		AuthenticationToken: datatypes.NewTwoByteNodeID(0),
		Timestamp:           time.Now(),
		TimeoutHint:         0xffff,
		AdditionalHeader:    services.NewNullAdditionalHeader(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &SecureChannel{
		c:         c,
		cfg:       cfg,
		ctx:       ctx,
		cancel:    cancel,
		reqHeader: reqHeader,
		handler:   make(map[uint32]chan Response),
	}
}

func (s *SecureChannel) Open() error {
	return s.openSecureChannel()
}

func (s *SecureChannel) Close() error {
	s.cancel()
	if err := s.closeSecureChannel(); err != nil {
		log.Print("failed to send close secure channel request")
	}
	return s.c.Close()
}

func (s *SecureChannel) LocalEndpoint() string {
	return s.EndpointURL
}

func (s *SecureChannel) openSecureChannel() error {
	// todo(fs): do we need to set the nonce if the security policy is None?
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	svc := &services.OpenSecureChannelRequest{
		ClientProtocolVersion:    0,
		SecurityTokenRequestType: services.ReqTypeIssue,
		MessageSecurityMode:      s.cfg.SecurityMode,
		ClientNonce:              nonce,
		RequestedLifetime:        s.cfg.Lifetime,
	}
	return s.Send(svc, func(v interface{}) error {
		resp, ok := v.(*services.OpenSecureChannelResponse)
		if !ok {
			return fmt.Errorf("got %T, want OpenSecureChannelResponse", svc)
		}
		s.cfg.SecurityTokenID = resp.SecurityToken.TokenID
		return nil
	})
}

// closeSecureChannel sends CloseSecureChannelRequest on top of UASC to SecureChannel.
func (s *SecureChannel) closeSecureChannel() error {
	svc := &services.CloseSecureChannelRequest{
		SecureChannelID: s.cfg.SecureChannelID,
	}
	return s.Send(svc, nil)
}

// Send sends the service request and calls h with the response object.
func (s *SecureChannel) Send(svc services.Service, h func(interface{}) error) error {
	ch, err := s.SendAsync(svc)
	if err != nil {
		return err
	}
	// todo(fs): handle timeout
	resp := <-ch
	if resp.Err != nil {
		return resp.Err
	}
	return h(resp.V)
}

// SendAsync sends the service request and returns a channel which will receive the
// response when it arrives.
func (s *SecureChannel) SendAsync(svc services.Service) (resp chan Response, err error) {
	// use reflection to set typeid and request header
	// the send method will update the request header fields
	// and reset them if the call failed.
	//
	// this is a workaround to avoid changing the existing datastructures
	// and break existing code. We should send type id and request header
	// separately here and remove these fields from the structs.:w
	typeID := datatypes.NewFourByteExpandedNodeID(0, svc.ServiceType())
	val := reflect.ValueOf(svc)
	val.Elem().Field(0).Set(reflect.ValueOf(typeID))
	val.Elem().Field(1).Set(reflect.ValueOf(s.reqHeader))

	// update counters and reset them on error
	s.cfg.SequenceNumber++
	s.reqHeader.RequestHandle++
	s.reqHeader.Timestamp = time.Now()
	defer func() {
		if err != nil {
			s.cfg.SequenceNumber--
			s.reqHeader.RequestHandle--
		}
	}()

	// todo(fs): should we drop the headers from Message and generate them here?
	// encode the message
	m := NewMessage(svc, s.cfg)
	b, err := m.Encode()
	if err != nil {
		return nil, err
	}

	// send the message
	if _, err := s.c.Write(b); err != nil {
		return nil, err
	}

	// register the handler
	resp = make(chan Response)
	s.mu.Lock()
	s.handler[s.reqHeader.RequestHandle] = resp
	s.mu.Unlock()
	return resp, nil
}

func (s *SecureChannel) recv() (services.Service, error) {
	const hdrlen = 12

	hdr := make([]byte, hdrlen)
	_, err := io.ReadFull(s.c, hdr)
	if err != nil {
		return nil, fmt.Errorf("sechan: read header failed: %s", err)
	}

	h := new(Header)
	if _, err := h.Decode(hdr); err != nil {
		return nil, fmt.Errorf("sechan: decode header failed: %s", err)
	}

	if s.cfg.SecureChannelID > 0 && s.cfg.SecureChannelID != h.SecureChannelID {
		return nil, fmt.Errorf("sechan: secure channel id mismatch: got 0x%04x, want 0x%04x", h.SecureChannelID, s.cfg.SecureChannelID)
	}

	b := make([]byte, h.MessageSize-hdrlen)
	if _, err := io.ReadFull(s.c, b); err != nil {
		return nil, fmt.Errorf("sechan: read message failed")
	}

	// todo(fs): handle ERR messages
	m := new(Message)
	if _, err := m.Decode(append(hdr, b...)); err != nil {
		return nil, fmt.Errorf("sechan: decode message failed: %s", err)
	}
	log.Printf("conn %d: rcvd %d bytes with %T", -1, len(hdr)+len(b), m.Service)

	if s.cfg.SecureChannelID == 0 {
		s.cfg.SecureChannelID = h.SecureChannelID
		log.Printf("conn %d: set secure channel id to %04x", -1, s.cfg.SecureChannelID)
	}

	return m.Service, nil
}

// monitor starts the request loop.
func (s *SecureChannel) monitor() {
	go s.run(s.ctx)
}

// run executes the request loop. It receives messages from the
// secure channel, decodes them and then forwards them to the
// registered response channel, if there is one. Otherwise, the
// message is dropped.
func (s *SecureChannel) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
			svc, err := s.recv()

			// since we are not decoding the ResponseHeader separately
			// we need to drop every message that has an error since we
			// cannot get to the RequestHandle in the ResponseHeader.
			// To fix this we must a) decode the ResponseHeader separately
			// and subsequently remove it and the TypeID from all service
			// structs and tests. We also need to add a deadline to all
			// handlers and check them periodically to time them out.
			if err != nil {
				log.Print(err)
				continue
			}

			// the response header is always the second field
			val := reflect.ValueOf(svc)
			resp := val.Elem().Field(1).Interface().(*services.ResponseHeader)

			// check if we have a pending request handler for this response.
			s.mu.Lock()
			ch := s.handler[resp.RequestHandle]
			delete(s.handler, resp.RequestHandle)
			s.mu.Unlock()

			// no handler -> next response
			if ch == nil {
				log.Printf("%T: %s", svc, err)
				continue
			}

			// send response to caller
			go func() {
				ch <- Response{svc, err}
				close(ch)
			}()
		}
	}
}
