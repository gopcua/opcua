package opcua

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua/keyring"
	"github.com/gopcua/opcua/ua"
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

	if c.config == nil {
		c.config = uasc.NewClientConfigSecurityNone(3600000)
	}

	sechan := uasc.NewSecureChannel(conn, c.config)
	if err := sechan.Open(); err != nil {
		conn.Close()
		return err
	}
	sechan.EndpointURL = c.Addr
	c.sechan = sechan

	// todo(dh): Should probably be able to use a different cert than the secchan cert
	var appURI string
	if c.config.LocalThumbprint != nil {
		cert, err := keyring.Certificate(c.config.LocalThumbprint)
		if err == nil && len(cert.URIs) > 0 {
			appURI = cert.URIs[0].String()
		}
	}

	var tokenID *ua.AnonymousIdentityToken
	for _, t := range c.config.ServerEndpoint.UserIdentityTokens {
		// todo(dh): Allow more than anonymous authentication eventually
		if t.TokenType == ua.UserTokenTypeAnonymous {
			tokenID = &ua.AnonymousIdentityToken{PolicyID: t.PolicyID}
			break
		}
	}
	if tokenID == nil {
		tokenID = &ua.AnonymousIdentityToken{PolicyID: "Anonymous"}
	}

	// todo(fs): this should probably be configurable.
	sessionCfg := uasc.NewClientSessionConfig(
		appURI,
		[]string{"en-US"},
		tokenID,
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
func (c *Client) Node(id *ua.NodeID) *Node {
	return &Node{ID: id, c: c}
}

// Read executes a synchronous read request.
//
// By default, the function requests the value of the nodes
// in the default encoding of the server.
func (c *Client) Read(req *ua.ReadRequest) (*ua.ReadResponse, error) {
	// clone the request and the ReadValueIDs to set defaults without
	// manipulating them in-place.
	rvs := make([]*ua.ReadValueID, len(req.NodesToRead))
	for i, rv := range req.NodesToRead {
		rc := &ua.ReadValueID{}
		*rc = *rv
		if rc.AttributeID == 0 {
			rc.AttributeID = ua.AttributeIDValue
		}
		if rc.DataEncoding == nil {
			rc.DataEncoding = &ua.QualifiedName{}
		}
		rvs[i] = rc
	}
	req = &ua.ReadRequest{
		MaxAge:             req.MaxAge,
		TimestampsToReturn: req.TimestampsToReturn,
		NodesToRead:        rvs,
	}

	var res *ua.ReadResponse
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*ua.ReadResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return res, err
}

// Write executes a synchronous write request.
func (c *Client) Write(req *ua.WriteRequest) (res *ua.WriteResponse, err error) {
	err = c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*ua.WriteResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return
}

// Browse executes a synchronous browse request.
func (c *Client) Browse(req *ua.BrowseRequest) (*ua.BrowseResponse, error) {
	var res *ua.BrowseResponse
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*ua.BrowseResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return res, err
}

// GetEndpoints reads all available endpoints from the server at endpoint
func GetEndpoints(endpoint string) ([]*ua.EndpointDescription, error) {
	var res *ua.GetEndpointsResponse

	ctx := context.Background()

	c := NewClient(endpoint, nil)
	conn, err := uacp.Dial(ctx, c.Addr)
	if err != nil {
		return nil, err
	}
	sechan := uasc.NewSecureChannel(conn, nil)
	if err := sechan.Open(); err != nil {
		conn.Close()
		return nil, err
	}
	sechan.EndpointURL = c.Addr
	c.sechan = sechan

	reqHeader := &ua.RequestHeader{
		AuthenticationToken: ua.NewTwoByteNodeID(0),
		Timestamp:           time.Now(),
		AdditionalHeader:    ua.NewExtensionObject(nil),
	}

	req := &ua.GetEndpointsRequest{
		RequestHeader: reqHeader,
		EndpointURL:   endpoint,
	}

	err = c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*ua.GetEndpointsResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})

	c.Close()
	return res.Endpoints, err
}

// todo(fs): this is not done yet since we need to be able to register
// todo(fs): monitored items.
type Subscription struct {
	res *ua.CreateSubscriptionResponse
}

// todo(fs): return subscription object with channel
func (c *Client) Subscribe(intv time.Duration) (*Subscription, error) {
	req := &ua.CreateSubscriptionRequest{
		RequestedPublishingInterval: float64(intv / time.Millisecond),
		RequestedLifetimeCount:      60,
		RequestedMaxKeepAliveCount:  20,
		PublishingEnabled:           true,
	}

	var res *ua.CreateSubscriptionResponse
	err := c.sechan.Send(req, func(v interface{}) error {
		r, ok := v.(*ua.CreateSubscriptionResponse)
		if !ok {
			return fmt.Errorf("invalid response: %T", v)
		}
		res = r
		return nil
	})
	return &Subscription{res}, err
}
