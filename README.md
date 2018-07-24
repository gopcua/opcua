# gopcua

[![CircleCI](https://circleci.com/gh/wmnsk/gopcua.svg?style=svg)](https://circleci.com/gh/wmnsk/gopcua)

gopcua provides easy and painless encoding/decoding of OPC UA protocol in pure Golang.

**THIS IS STILL EXPERIMENTAL PROJECT, ANY IMPLEMENTATION MAY CHANGE DRASTICALLY IN FUTURE**


## Example

```go
    // Create Hello (Version, SendBufSize, ReceiveBufSize, MaxMessageSize, EndPointURL)
    hello := connection.NewHello(0, 10, 20, 1024, "opc.tcp://endpoint.example/foo/bar")

    // Serialize to write on TCP connection
    helloBytes, err := hello.Serialize()
    if err != nil {
        log.Fatalf("Failed to serialize Hello: %s", err)
    }

    // Setup TCP connection
    raddr, err := net.ResolveTCPAddr("tcp", "10.0.0.1:11111")
    if err != nil {
        log.Fatalf("Failed to resolve TCP Address: %s", err)
    }

    conn, err := net.DialTCP("tcp", nil, raddr)
    if err != nil {
        log.Fatalf("Failed to open TCP connection: %s", err)
    }
    defer conn.Close()

    // Write on TCP connection once per 3 sec.
    for {
        if _, err := conn.Write(helloBytes); err != nil {
            log.Fatalf("Failed to write Hello: %s", err)
        }
        log.Printf("Successfully sent Hello: %s", hello.String())

        time.Sleep(3 * time.Second)
    }
```

## Author

Yoshiyuki Kurauchi ([GitHub](https://github.com/wmnsk/) / [Twitter](https://twitter.com/wmnskdmms))

[![BMC](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://buymeacoff.ee/yoshk)

## License

[MIT](https://github.com/wmnsk/gopc-ua/blob/master/LICENSE)
