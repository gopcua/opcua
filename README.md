# gopcua

[![CircleCI](https://circleci.com/gh/wmnsk/gopcua.svg?style=svg)](https://circleci.com/gh/wmnsk/gopcua)
[![Go Report Card](https://goreportcard.com/badge/github.com/wmnsk/gopcua)](https://goreportcard.com/report/github.com/wmnsk/gopcua)

[![GoDoc](https://godoc.org/github.com/wmnsk/gopcua?status.svg)](https://godoc.org/github.com/wmnsk/gopcua)
[![GitHub](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/wmnsk/gopcua/blob/master/LICENSE)

gopcua provides easy and painless encoding/decoding of OPC UA Binary Protocol in pure Golang.

## Disclaimer

THIS IS STILL EXPERIMENTAL PROJECT, ANY IMPLEMENTATION INCLUDING EXPORTED APIs MAY CHANGE DRASTICALLY IN THE FUTURE.

## Quickstart

See example directory for sample codes.

### Run

The following command will send `Hello`, `OpenSecureChannel`, `CreateSession`, `CloseSecureChannel` to the destination specified in command-line arguments.

```shell-session
git clone git@github.com:wmnsk/gopcua.git
cd examples/sender
go run sender.go --ip <dst IP> --port <dst Port> --url "opc.tcp://endpoint.example/gopcua/server"
```

## Roadmap

(To be written more precisely...)

- [ ] Protocol definitions
  - [x] OPC UA Connection Protocol
    - [x] Interface to handle all messages
    - [x] Header
    - [x] Hello
    - [x] Acknowledge
    - [x] Error
    - [x] Reverse Hello
  - [ ] OPC UA Secure Conversation
    - [x] Message header
    - [x] Asymmetric algorithm Security header
    - [x] Symmetric algorithm Security header
    - [x] Sequence header
    - [ ] Message footer
  - [ ] Basic Service Implementation
    - [x] Interface to handle all services
    - [x] Open Secure Channel Request / Response
    - [x] Close Secure Channel Request / Response
    - [x] Get Endpoints Request / Response
    - [x] Create Session Request / Response
    - [ ] Close Session Request / Response
    - [ ] Activate Session Request / Response
  - [ ] Additional Service Implementation
    - [ ] XXX...
    - [ ] XXX...
    - [ ] XXX...
- [ ] State Machine
  - [ ] Implement `net.Conn` (`Dial()`, `Listen()`, `Accept()`)
  - [ ] XXX...
- [ ] Others
  - [ ] Documentation (improve GoDoc, README)
  - [x] Integrated way to handle common errors
  - [ ] XXX...

## Author

Yoshiyuki Kurauchi ([GitHub](https://github.com/wmnsk/) / [Twitter](https://twitter.com/wmnskdmms))

## License

[MIT](https://github.com/wmnsk/gopc-ua/blob/master/LICENSE)
