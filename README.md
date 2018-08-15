# gopcua

[![CircleCI](https://circleci.com/gh/wmnsk/gopcua.svg?style=svg)](https://circleci.com/gh/wmnsk/gopcua)

gopcua provides easy and painless encoding/decoding of OPC UA protocol in pure Golang.

**THIS IS STILL EXPERIMENTAL PROJECT, ANY IMPLEMENTATION MAY CHANGE DRASTICALLY IN FUTURE**


## Quickstart

See example directory for sample codes.

### Run

```shell-session
$ git clone git@github.com:wmnsk/gopcua.git
$ cd examples/sender
$ go run sender.go --ip <dst IP> --port <dst Port> --url "opc.tcp://endpoint.example/gopcua/server"
```

## Roadmap

(To be written more precisely...)

- [ ] Protocol definitions
    - [ ] OPC UA Connection Protocol
        - [x] Interface to handle all messages
        - [x] Header
        - [x] Hello
        - [x] Acknowledge
        - [x] Error
        - [ ] Reverse Hello
    - [ ] OPC UA Secure Conversation
        - [ ] Interface to handle all messages
        - [x] Message Header
        - [x] Asymmetric algorithm Security header
        - [x] Symmetric algorithm Security header
        - [x] Sequence Header
        - [ ] Message footer
        - [ ] Message Implementation
            - [ ] Open Secure Channel Request / Response
            - [ ] Close Secure Channel Request / Response
            - [ ] Get Endpoint Request / Response
            - [ ] XXX...
- [ ] State Machine
    - [ ] Implement `net.Conn`
    - [ ] XXX...
- [ ] Others
    - [ ] Documentation(GoDoc, README)
    - [ ] Integrated way to handle common errors


## Author

Yoshiyuki Kurauchi ([GitHub](https://github.com/wmnsk/) / [Twitter](https://twitter.com/wmnskdmms))

[![BMC](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://buymeacoff.ee/yoshk)

## License

[MIT](https://github.com/wmnsk/gopc-ua/blob/master/LICENSE)
