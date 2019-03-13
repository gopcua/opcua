package opcua

import (
	"context"
	"fmt"
	"time"

	uad "github.com/gopcua/opcua/datatypes"
	uas "github.com/gopcua/opcua/services"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

// Client is a high-level client for an OPC/UA server.
// It establishes a secure channel and a session.
type Client struct {
	Addr string

	config  *uasc.Config
	sechan  *uasc.SecureChannel
	session *uasc.Session
}

func NewClient(addr string, cfg *uasc.Config) *Client {
	return &Client{Addr: addr, config: cfg}
}

// Open connects to the server and establishes a secure channel
// and a session.
func (c *Client) Open() error {
	ctx := context.Background()
	conn, err := uacp.Dial(ctx, c.Addr)
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

	// todo(fs): this should probably be configurable.
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

// Close closes the session, the secure channel and the network
// connection to the server.
func (c *Client) Close() error {
	if c.session != nil {
		c.session.Close()
	}
	return c.sechan.Close()
}

// Node returns a node object which accesses its attributes
// through this client connection.
func (c *Client) Node(id *uad.NodeID) *Node {
	return &Node{ID: id, c: c}
}

// Read executes a synchronous read request.
func (c *Client) Read(req *uas.ReadRequest) (*uas.ReadResponse, error) {
	var res *uas.ReadResponse
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*uas.ReadResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return res, err
}

// Browse executes a synchronous browse request.
func (c *Client) Browse(req *uas.BrowseRequest) (*uas.BrowseResponse, error) {
	var res *uas.BrowseResponse
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*uas.BrowseResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return res, err
}

// todo(fs): this is not done yet since we need to be able to register
// todo(fs): monitored items.
type Subscription struct {
	res *uas.CreateSubscriptionResponse
}

// todo(fs): return subscription object with channel
func (c *Client) Subscribe(intv time.Duration) (*Subscription, error) {
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
	return &Subscription{res}, err
}
