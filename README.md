# gopcua

gopcua provides easy and painless handling of OPC UA Binary Protocol in pure Golang.

[![CircleCI](https://circleci.com/gh/wmnsk/gopcua.svg?style=svg)](https://circleci.com/gh/wmnsk/gopcua)
[![GoDoc](https://godoc.org/github.com/wmnsk/gopcua?status.svg)](https://godoc.org/github.com/wmnsk/gopcua)
[![Go Report Card](https://goreportcard.com/badge/github.com/wmnsk/gopcua)](https://goreportcard.com/report/github.com/wmnsk/gopcua)
[![GitHub](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/wmnsk/gopcua/blob/master/LICENSE)


## Quickstart

See example directory for sample codes.
Currently simple `client`, `server` implementation and `sender`, which lets user to manipulate any parts including connection setup sequence, are available.

### Requirements

gopcua is implemented in pure Golang. So just `go get` the following packages.

_Vendoring is planned to be implemented after Go 1.12 release._

[github.com/pkg/errors](https://github.com/pkg/errors)  
[github.com/google/go-cmp](https://github.com/google/go-cmp)  
[github.com/wmnsk/gopcua](https://github.com/wmnsk/gopcua)  

### Running Examples

[`client`](./examples/client) opens the SecureChannel with the endpoint specified in command-line arguments.

If `--payload` is given, it sends any data on top of UASC headers.

```shell-session
cd examples/client
go run client.go --endpoint "opc.tcp://endpoint.example/gopcua/server" --payload <payload in hex stream format>
```

[`server`](./examples/server) listens and accepts the SecureChannel opening request from the client on specified network.

```shell-session
cd examples/client
go run server.go --endpoint "opc.tcp://endpoint.example/gopcua/server"
```

## Help Wanted!

We believe our idea to implement the OPC-UA protocol in Golang can contribute to fostering the industry, and we are trying to make this project production-ready.
However, due to the lack of resources the progress is not quite good actually. So, we want your help.

### by writing codes

As listed in GitHub [issues](https://github.com/wmnsk/gopcua/issues) and [projects](https://github.com/wmnsk/gopcua/projects/2), we still have a lot of things to be considered/implemented.
Resolving the issues listed by writing your code would help really much.

### by reporting issues

Please feel free to open an issue to report anything you face when using the package.

### by reviewing

Please don't hesitate to post your thoughts on some issues and/or pull requests. It also helps us a lot.

### by sharing

The number of stars or shares on Twitter(or anything else) is a great motivation for us to work on the project continuously.

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

Yoshiyuki Kurauchi ([Twitter](https://twitter.com/wmnskdmms) / [LinkedIn](https://www.linkedin.com/in/yoshiyuki-kurauchi/))

## License

[MIT](https://github.com/wmnsk/gopc-ua/blob/master/LICENSE)
