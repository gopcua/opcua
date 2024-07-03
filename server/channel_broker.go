package server

import (
	"context"
	"crypto/rsa"
	"io"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

type channelBroker struct {
	endpoints map[string]*ua.EndpointDescription

	wg sync.WaitGroup

	// mu protects concurrent modification of s, secureChannelID, and secureTokenID
	mu sync.RWMutex
	// s is a slice of all SecureChannels watched by the channelBroker
	s map[uint32]*uasc.SecureChannel

	// Next Secure Channel ID to issue to a client
	secureChannelID uint32

	// Next Token ID to issue to a client
	secureTokenID uint32

	// msgChan is the common channel that all messages from all channels
	// get funneled into for handling
	msgChan chan *uasc.MessageBody
	logger  Logger
}

func newChannelBroker(logger Logger) *channelBroker {
	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	return &channelBroker{
		endpoints:       make(map[string]*ua.EndpointDescription),
		s:               make(map[uint32]*uasc.SecureChannel),
		msgChan:         make(chan *uasc.MessageBody),
		secureChannelID: uint32(rng.Int31()),
		secureTokenID:   uint32(rng.Int31()),
		logger:          logger,
	}
}

// RegisterConn connects a new UACP connection to the channel broker's list
// of connections and starts waiting for data on it.  Data is pushed onto the broker's
// Response channel
// Blocks until the context is done, the connection closes, or a critical error
func (c *channelBroker) RegisterConn(ctx context.Context, conn *uacp.Conn, localCert []byte, localKey *rsa.PrivateKey) error {
	cfg := defaultChannelConfig()
	cfg.Certificate = localCert
	cfg.LocalKey = localKey

	c.mu.Lock()
	c.secureChannelID++
	c.secureTokenID++
	secureChannelID := c.secureChannelID
	secureTokenID := c.secureTokenID
	sequenceNumber := uint32(mrand.Int31n(1023) + 1)
	c.mu.Unlock()

	errch := make(chan error, 1)
	sc, err := uasc.NewServerSecureChannel(
		"", // todo(fs): this is most likely wrong
		conn,
		cfg,
		errch,
		secureChannelID,
		sequenceNumber,
		secureTokenID,
	)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Error creating secure channel for new connection: %s", err)
		}
		return err
	}

	c.mu.Lock()
	c.s[secureChannelID] = sc
	if c.logger != nil {
		c.logger.Info("Registered new channel (id %d) now at %d channels", secureChannelID, len(c.s))
	}
	c.mu.Unlock()
	c.wg.Add(1)
outer:
	for {
		select {
		case <-ctx.Done():
			// todo(fs): return error?
			break outer

		default:
			msg := sc.Receive(ctx)
			if msg.Err == io.EOF {
				if c.logger != nil {
					c.logger.Warn("Secure Channel %d closed", secureChannelID)
				}
				break outer
			}
			// todo(fs): honor ctx
			c.msgChan <- msg
		}
	}

	c.mu.Lock()
	delete(c.s, secureChannelID)
	c.mu.Unlock()
	c.wg.Done()

	return nil
}

// Close gracefully closes all secure channels
// todo(fs): use ctx
func (c *channelBroker) Close() error {
	var err error
	c.mu.Lock()
	for _, s := range c.s {
		s.Close()
	}
	c.mu.Unlock()

	// Wait for all goroutines to finish or timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second): // todo(fs): magic number
		if c.logger != nil {
			c.logger.Error("CloseAll: timed out waiting for channels to exit")
		}
	}

	return err
}

func (c *channelBroker) ReadMessage(ctx context.Context) *uasc.MessageBody {
	select {
	case <-ctx.Done():
		return nil
	case msg := <-c.msgChan:
		return msg
	}
}
