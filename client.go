package gopcua

import (
	"fmt"
	"time"

	uad "github.com/wmnsk/gopcua/datatypes"
	uas "github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uasc"
)

type Client struct {
	Addr string

	config  *uasc.Config
	sechan  *uasc.SecureChannel
	session *uasc.Session
}

func NewClient(addr string, cfg *uasc.Config) *Client {
	return &Client{Addr: addr, config: cfg}
}

func (c *Client) Open() error {
	conn, err := uasc.Dial(c.Addr)
	if err != nil {
		return err
	}
	sechan := uasc.NewSecureChannel(conn, nil)
	if err := sechan.Open(); err != nil {
		conn.Close()
		return err
	}
	sechan.EndpointURL = c.Addr
	c.sechan = sechan

	sessionCfg := uasc.NewClientSessionConfig(
		[]string{"en-US"},
		uad.NewAnonymousIdentityToken("open62541-anonymous-policy"),
	)

	session := uasc.NewSession(sechan, sessionCfg)
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

func (c *Client) Read(id *uad.NodeID) (*uad.Variant, error) {
	req := &uas.ReadRequest{
		MaxAge:             0,
		TimestampsToReturn: 0,
		NodesToRead: []*uad.ReadValueID{
			&uad.ReadValueID{
				NodeID:       id,
				AttributeID:  uad.IntegerIDValue,
				DataEncoding: &uad.QualifiedName{},
			},
		},
	}

	var res *uad.Variant
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*uas.ReadResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		if len(r.Results) > 0 {
			res = r.Results[0].Value
		}
		return nil
	})
	return res, err
}

// todo(fs): return subscription object with channel
func (c *Client) Subscribe(intv time.Duration) (*uas.CreateSubscriptionResponse, error) {
	req := &uas.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(intv / time.Millisecond),
		RequestedLifetimeCount:      60,
		RequestedMaxKeepAliveCount:  20,
		PublishingEnabled:           true,
	}

	var res *uas.CreateSubscriptionResponse
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*uas.CreateSubscriptionResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return res, err
}
