package conn

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
	if err := c.sendReadRequest(nid); err != nil {
		return nil, err
	}
	resp, err := c.handleReadResponse()
	if err != nil {
		return nil, err
	}
	if len(resp.Results) > 0 {
		return resp.Results[0].Value, nil
	}
	return nil, nil
}

func (c *Client) sendReadRequest(id *ua.NodeID) error {
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
	// todo(fs): shouldn't we send this via the session?
	return c.sechan.send(req)
}

func (c *Client) handleReadResponse() (*uas.ReadResponse, error) {
	svc, err := c.sechan.recv()
	if err != nil {
		return nil, err
	}
	resp, ok := svc.(*uas.ReadResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response. Got %T, want CreateSessionResponse", svc)
	}
	return resp, nil
}
