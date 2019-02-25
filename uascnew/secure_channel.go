package uascnew

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"reflect"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uasc"
)

type SecureChannel struct {
	c           *Conn
	cfg         *uasc.Config
	reqHeader   *services.RequestHeader
	resHeader   *services.ResponseHeader
	endpointURL string
}

func NewSecureChannel(c *Conn, cfg *uasc.Config) *SecureChannel {
	if cfg == nil {
		cfg = uasc.NewClientConfigSecurityNone(3333, 3600000)
	}

	// always reset the secure channel id
	cfg.SecureChannelID = 0

	reqHeader := &services.RequestHeader{
		AuthenticationToken: datatypes.NewTwoByteNodeID(0),
		Timestamp:           time.Now(),
		TimeoutHint:         0xffff,
		AdditionalHeader:    services.NewNullAdditionalHeader(),
	}

	return &SecureChannel{
		c:         c,
		cfg:       cfg,
		reqHeader: reqHeader,
	}
}

func (s *SecureChannel) Open() error {
	if err := s.sendOpenSecureChannelRequest(); err != nil {
		return err
	}
	return s.handleOpenSecureChannelResponse()
}

func (s *SecureChannel) Close() error {
	if err := s.sendCloseSecureChannelRequest(); err != nil {
		log.Print("failed to send close secure channel request")
	}
	return s.c.Close()
}

func (s *SecureChannel) sendOpenSecureChannelRequest() error {
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
	return s.send(svc)
}

func (s *SecureChannel) handleOpenSecureChannelResponse() error {
	svc, err := s.recv()
	if err != nil {
		return err
	}
	resp, ok := svc.(*services.OpenSecureChannelResponse)
	if !ok {
		return fmt.Errorf("got %T, want OpenSecureChannelResponse", svc)
	}
	s.cfg.SecurityTokenID = resp.SecurityToken.TokenID
	return nil
}

// CloseSecureChannelRequest sends CloseSecureChannelRequest on top of UASC to SecureChannel.
func (s *SecureChannel) sendCloseSecureChannelRequest() error {
	svc := &services.CloseSecureChannelRequest{
		SecureChannelID: s.cfg.SecureChannelID,
	}
	return s.send(svc)
}

func (s *SecureChannel) send(svc services.Service) (err error) {
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
	m := uasc.NewMessage(svc, s.cfg)
	b, err := m.Encode()
	if err != nil {
		return err
	}

	// send the message
	_, err = s.c.Write(b)
	return err
}

func (s *SecureChannel) recv() (services.Service, error) {
	const hdrlen = 12

	hdr := make([]byte, hdrlen)
	_, err := io.ReadFull(s.c, hdr)
	if err != nil {
		return nil, fmt.Errorf("sechan: read header failed: %s", err)
	}

	h := new(uasc.Header)
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

	m := new(uasc.Message)
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
