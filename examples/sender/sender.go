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

	var payload [][]byte
	// Setup Hello
	hello, err := uacp.NewHello(0, uint32(*sndBuf), uint32(*rcvBuf), uint32(*maxMsg), *url).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Hello: %s", err)
	}
	payload = append(payload, hello)

	// Setup OpenSecureChannel
	o := services.NewOpenSecureChannelRequest(
		time.Now(),
		0, 1, 0, 0, "",
		0, services.ReqTypeIssue, services.SecModeNone, 6000000, nil,
	)
	o.RequestHeader.SetDiagAll()

	opn, err := o.Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize OpenSecureChannelRequest: %s", err)
	}

	seqHdr, err := uasc.NewSequenceHeader(
		1, 1, opn,
	).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize SequenceHeader: %s", err)
	}

	asyHdr, err := uasc.NewAsymmetricSecurityHeader(
		*uri, "", "", seqHdr,
	).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize AsymmetricHeader: %s", err)
	}

	hdr, err := uasc.NewHeader(
		"OPN", "F", 0, asyHdr,
	).Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize MessageHeader: %s", err)
	}

	payload = append(payload, hdr)

	//x, _ := hex.DecodeString("4f504e4684000000000000002f000000687474703a2f2f6f7063666f756e646174696f6e2e6f72672f55412f5365637572697479506f6c696379234e6f6e65000000000000000001000000010000000100be010000803553e02401000001000000ff030000ffffffff00000000000000000000000000000001000000ffffffff808d5b00")
	//payload = append(payload, x)

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
