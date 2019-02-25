package uascnew

import (
	"testing"
	"time"
)

func TestDial(t *testing.T) {
	conn, err := Dial("opc.tcp://localhost:4840")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	defer conn.Close()
}

func TestSecureChannel(t *testing.T) {
	conn, err := Dial("opc.tcp://localhost:4840")
	if err != nil {
		t.Fatal(err)
	}
	s := NewSecureChannel(conn, nil)
	if err := s.Open(); err != nil {
		t.Fatal(err)
	}
	defer s.Close()

}

func TestClientRead(t *testing.T) {
	c := NewClient("opc.tcp://localhost:4840", nil)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	v, err := c.Read("ns=0;i=2258")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("timex: %v", v)
}
