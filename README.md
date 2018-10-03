# gopcua

gopcua provides easy and painless handling of OPC UA Binary Protocol in pure Golang.

[![CircleCI](https://circleci.com/gh/wmnsk/gopcua.svg?style=svg)](https://circleci.com/gh/wmnsk/gopcua)
[![GoDoc](https://godoc.org/github.com/wmnsk/gopcua?status.svg)](https://godoc.org/github.com/wmnsk/gopcua)
[![Go Report Card](https://goreportcard.com/badge/github.com/wmnsk/gopcua)](https://goreportcard.com/report/github.com/wmnsk/gopcua)
[![GitHub](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/wmnsk/gopcua/blob/master/LICENSE)

## Table of Contents

- [gopcua](#gopcua)
  - [Table of Contents](#table-of-contents)
  - [Quickstart](#quickstart)
    - [Installing](#installing)
    - [Running Examples](#running-examples)
  - [Supported Features](#supported-features)
    - [Protocol Stack](#protocol-stack)
    - [Services](#services)
  - [Disclaimer](#disclaimer)
  - [Author](#author)
  - [License](#license)

## Quickstart

See example directory for sample codes.
Currently simple `client`, `server` implementation and `sender`, which lets user to manipulate any parts including connection setup sequence, are available.

### Installing

Simply use `go get`.

The following command will send `Hello`, `OpenSecureChannel`, `CreateSession`, `CloseSecureChannel` to the destination specified in command-line arguments.

```shell-session
go get -u github.com/pkg/errors
go get -u github.com/google/go-cmp
go get -u github.com/wmnsk/gopcua
```

### Running Examples

`client` opens the SecureChannel with the endpoint specified in command-line arguments.

If `--payload` is given, it sends any data on top of UASC headers.

```shell-session
cd examples/client
go run client.go --endpoint "opc.tcp://endpoint.example/gopcua/server" --payload <payload in hex stream format>
```

`server` listens and accepts the SecureChannel opening request from the client on specified network.

```shell-session
cd examples/client
go run server.go --endpoint "opc.tcp://endpoint.example/gopcua/server"
```

NOTE: Automatic session activation has not been implemented at this time.

## Supported Features

### Protocol Stack

| Categories     | Features                         | Supported | Notes |
| -------------- | -------------------------------- | --------- | ----- |
| Encoding       | OPC UA Binary                    | Yes       |       |
|                | OPC UA JSON                      |           |       |
|                | OPC UA XML                       |           |       |
| Transport      | UA-TCP UA-SC UA Binary           | Yes       |       |
|                | OPC UA HTTPS                     |           |       |
|                | SOAP-HTTP WS-SC UA Binary        |           |       |
|                | SOAP-HTTP WS-SC UA XML           |           |       |
|                | SOAP-HTTP WS-SC UA XML-UA Binary |           |       |
| Encryption     | None                             | Yes       |       |
|                | Basic128Rsa15                    |           |       |
|                | Basic256                         |           |       |
|                | Basic256Sha256                   |           |       |
| Authentication | Anonymous                        |           |       |
|                | User Name Password               |           |       |
|                | X509 Certificate                 |           |       |

### Services

| Service Set                 | Service                       | Supported | Notes        |
| --------------------------- | ----------------------------- | --------- | ------------ |
| Discovery Service Set       | FindServers                   | Yes       |              |
|                             | FindServersOnNetwork          | Yes       |              |
|                             | GetEndpoints                  | Yes       |              |
|                             | RegisterServer                |           |              |
|                             | RegisterServer2               |           |              |
| Secure Channel Service Set  | OpenSecureChannel             | Yes       |              |
|                             | CloseSecureChannel            | Yes       |              |
| Session Service Set         | CreateSession                 | Yes       |              |
|                             | CloseSession                  | Yes       |              |
|                             | ActivateSession               | Yes       |              |
|                             | Cancel                        | Yes       |              |
| Node Management Service Set | AddNodes                      |           |              |
|                             | AddReferences                 |           |              |
|                             | DeleteNodes                   |           |              |
|                             | DeleteReferences              |           |              |
| View Service Set            | Browse                        |           |              |
|                             | BrowseNext                    |           |              |
|                             | TranslateBrowsePathsToNodeIds |           |              |
|                             | RegisterNodes                 |           |              |
|                             | UnregisterNodes               |           |              |
| Query Service Set           | QueryFirst                    |           |              |
|                             | QueryNext                     |           |              |
| Attribute Service Set       | Read                          | Yes       |              |
|                             | Write                         | Yes       |              |
|                             | HistoryRead                   |           |              |
|                             | HistoryUpdate                 |           |              |
| Method Service Set          | Call                          |           |              |
| MonitoredItems Service Set  | CreateMonitoredItems          |           |              |
|                             | DeleteMonitoredItems          |           |              |
|                             | ModifyMonitoredItems          |           |              |
|                             | SetMonitoringMode             |           |              |
|                             | SetTriggering                 |           |              |
| Subscription Service Set    | CreateSubscription            | Partial   | Request-only |
|                             | ModifySubscription            |           |              |
|                             | SetPublishingMode             |           |              |
|                             | Publish                       |           |              |
|                             | Republish                     |           |              |
|                             | DeleteSubscriptions           |           |              |
|                             | TransferSubscriptions         |           |              |

_Tables here are generated by [Markdown Tables Generator](https://www.tablesgenerator.com/markdown_tables)_

## Disclaimer

This is still experimental project. Any part of the exported API may be changed before first release.

## Author

Yoshiyuki Kurauchi ([GitHub](https://github.com/wmnsk/) / [Twitter](https://twitter.com/wmnskdmms))

## License

[MIT](https://github.com/wmnsk/gopc-ua/blob/master/LICENSE)
