package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uacp"
	"github.com/wmnsk/gopcua/uasc"
)

func main() {
	var (
		ip     = flag.String("ip", "127.0.0.1", "Destination IP Address")
		port   = flag.String("port", "11111", "Destination Port Number")
		sndBuf = flag.Int("sndbuf", 65535, "SendBufferSize")
		rcvBuf = flag.Int("rcvbuf", 65535, "ReceiveBufferSize")
		maxMsg = flag.Int("maxmsg", 0, "MaxMessageSize")
		url    = flag.String("url", "opc.tcp://deadbeef.example/foo/bar", "OPC UA Endpoint URL")
		uri    = flag.String("uri", "http://opcfoundation.org/UA/SecurityPolicy#None", "OPC UA Secure Policy URI")
	)
	flag.Parse()

	// Setup Hello
	hello, err := uacp.NewHello(0, uint32(*sndBuf), uint32(*rcvBuf), uint32(*maxMsg), *url).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Hello: %s", err)
	}

	// Setup OpenSecureChannel
	cfg := &uasc.Config{
		SecureChannelID:   1,
		SecurityPolicyURI: *uri,
		RequestID:         1,
	}

	o := services.NewOpenSecureChannelRequest(
		time.Now(), 0, 1, 0, 0, "", 0,
		services.ReqTypeIssue, services.SecModeNone,
		6000000, nil,
	)
	o.SetDiagAll()
	opn, err := uasc.New(o, cfg).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize OpenSecureChannel: %s", err)
	}

	// Prepare TCP connection
	raddr, err := net.ResolveTCPAddr("tcp", *ip+":"+*port)
	if err != nil {
		log.Fatalf("Failed to resolve TCP Address: %s", err)
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatalf("Failed to open TCP connection: %s", err)
	}
	defer conn.Close()

	// Send Hello and wait for Acknowledge to come
	if _, err := conn.Write(hello); err != nil {
		log.Fatalf("Failed to write Hello: %s", err)
	}
	log.Printf("Successfully sent Hello: %x", hello)

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		log.Fatalf("Failed to read from conn: %s", err)
	}

	cp, err := uacp.Decode(buf[:n])
	if err != nil {
		log.Fatalf("Something went wrong: %s", err)
	}

	switch cp.MessageTypeValue() {
	case uacp.MessageTypeAcknowledge:
		log.Printf("Received Acknowlege: %s", cp)

		// Send OpenSecureChannelRequest and wait for Resposne to come
		if _, err := conn.Write(opn); err != nil {
			log.Fatalf("Failed to write OpenSecureChannel: %s", err)
		}
		log.Printf("Successfully sent OpenSecureChannel: %x", opn)

		m, err := conn.Read(buf)
		if err != nil {
			log.Fatalf("Failed to read from conn: %s", err)
		}

		sc, err := uasc.Decode(buf[:m])
		if err != nil {
			log.Fatalf("Something went wrong: %s", err)
		}
		log.Printf("Received: %s", sc)
	case uacp.MessageTypeError:
		log.Fatalf("Received Error, closing: %s", cp)
	default:
		log.Fatalf("Received unexpected message, closing: %s", cp)
	}
}
