package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/wmnsk/gopc-ua/connection"
)

func main() {
	var (
		ip   = flag.String("ip", "127.0.0.1", "Destination IP Address.")
		port = flag.String("port", "11111", "Destination Port Number.")
		url  = flag.String("url", "opc.tcp://deadbeef.example/foo/bar", "OPC UA Endpoint URL.")
	)
	flag.Parse()
	hello := connection.NewHello(0, 10, 20, 1024, *url)
	helloBytes, err := hello.Serialize()
	if err != nil {
		log.Fatalf("Failed to serialize Hello: %s", err)
	}

	raddr, err := net.ResolveTCPAddr("tcp", *ip+":"+*port)
	if err != nil {
		log.Fatalf("Failed to resolve TCP Address: %s", err)
	}

	conn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatalf("Failed to open TCP connection: %s", err)
	}
	defer conn.Close()

	for {
		if _, err := conn.Write(helloBytes); err != nil {
			log.Fatalf("Failed to write Hello: %s", err)
		}
		log.Printf("Successfully sent Hello: %s", hello.String())

		time.Sleep(3 * time.Second)
	}
}
