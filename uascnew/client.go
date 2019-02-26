package uascnew

import (
	"fmt"

	ua "github.com/wmnsk/gopcua/datatypes"
	uas "github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uasc"
)

type Client struct {
	Addr   string
	Config *uasc.Config

	sechan  *SecureChannel
	session *Session
}

func NewClient(addr string, cfg *uasc.Config) *Client {
	return &Client{Addr: addr, Config: cfg}
}

func (c *Client) Connect() error {
	conn, err := Dial(c.Addr)
	if err != nil {
		return err
	}
	sechan := NewSecureChannel(conn, nil)
	if err := sechan.Open(); err != nil {
		conn.Close()
		return err
	}
	sechan.endpointURL = c.Addr
	c.sechan = sechan

	sessionCfg := uasc.NewClientSessionConfig(
		[]string{"en-US"},
		ua.NewAnonymousIdentityToken("open62541-anonymous-policy"),
	)

	session := NewSession(sechan, sessionCfg)
	if err := session.Open(); err != nil {
		sechan.Close()
		return err
	}
	c.session = session
	return nil
}

func (c *Client) Close() error {
	if c.session != nil {
		c.session.Close()
	}
	return c.sechan.Close()
}

func (c *Client) Read(id string) (*ua.Variant, error) {
	nid, err := ua.NewNodeID(id)
	if err != nil {
		return nil, err
	}

	h := NewReadAsync(c.sechan)
	if err := h.Send(nid); err != nil {
		return nil, err
	}
	return h.Recv()
}

// ReadAsync implements an async ReadRequest/Response for a single
// node value.
type ReadAsync struct {
	sechan *SecureChannel
	ch     chan asyncResponse
}

func NewReadAsync(sechan *SecureChannel) *ReadAsync {
	return &ReadAsync{sechan, make(chan asyncResponse)}
}

func (r *ReadAsync) Send(id *ua.NodeID) error {
	req := &uas.ReadRequest{
		MaxAge:             0,
		TimestampsToReturn: 0,
		NodesToRead: []*ua.ReadValueID{
			&ua.ReadValueID{
				NodeID:       id,
				AttributeID:  ua.IntegerIDValue,
				DataEncoding: &ua.QualifiedName{},
			},
		},
	}
	return r.sechan.send(req, r.ch)
}

func (r *ReadAsync) Recv() (*ua.Variant, error) {
	resp := <-r.ch
	if resp.err != nil {
		return nil, resp.err
	}
	res, ok := resp.v.(*uas.ReadResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response: %T", resp.v)
	}
	if len(res.Results) > 0 {
		return res.Results[0].Value, nil
	}
	return nil, nil
}
