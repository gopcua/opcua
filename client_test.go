package opcua

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/pascaldekloe/goe/verify"
)

func TestClient_Send_DoesNotPanicWhenDisconnected(t *testing.T) {
	c := NewClient("opc.tcp://example.com:4840")
	err := c.Send(&ua.ReadRequest{}, func(i interface{}) error {
		return nil
	})
	verify.Values(t, "", err, ua.StatusBadServerNotConnected)
}

func TestClient_NewClientCanBeClosed(t *testing.T) {
	c := NewClient("opc.tcp://example.com:4840")
	c.Close()
}
