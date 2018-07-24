package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/wmnsk/gopcua/uacp"
)

func main() {
	var (
		ip     = flag.String("ip", "127.0.0.1", "Destination IP Address")
		port   = flag.String("port", "11111", "Destination Port Number")
		sndBuf = flag.Int("sndbuf", 65535, "SendBufferSize")
		rcvBuf = flag.Int("rcvbuf", 65535, "ReceiveBufferSize")
		maxMsg = flag.Int("maxmsg", 0, "MaxMessageSize")
		url    = flag.String("url", "opc.tcp://deadbeef.example/foo/bar", "OPC UA Endpoint URL")
		reason = flag.String("reason", "Something went wrong", "Error reason")
	)
	flag.Parse()

	var payload [][]byte
	// Setup Hello
	hello, err := uacp.NewHello(0, uint32(*sndBuf), uint32(*rcvBuf), uint32(*maxMsg), *url).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Hello: %s", err)
	}
	payload = append(payload, hello)

	// Setup Acknowledge
	ack, err := uacp.NewAcknowledge(0, uint32(*sndBuf), uint32(*rcvBuf), uint32(*maxMsg)).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Acknowledge: %s", err)
	}
	payload = append(payload, ack)

	// Setup Error
	e, err := uacp.NewError(uacp.BadSecureChannelClosed, *reason).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Error: %s", err)
	}
	payload = append(payload, e)

	raddr, err := net.ResolveTCPAddr("tcp", *ip+":"+*port)
	if err != nil {
		log.Fatalf("Failed to resolve TCP Address: %s", err)
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatalf("Failed to open TCP connection: %s", err)
	}
	defer conn.Close()

	for _, x := range payload {
		if _, err := conn.Write(x); err != nil {
			log.Fatalf("Failed to write message: %s", err)
		}
		log.Printf("Successfully sent message: %x", x)

		time.Sleep(1 * time.Second)
	}
}
