# OPCUA

opcua is a native Go implementation of the OPC/UA Binary Protocol.

[![CircleCI](https://circleci.com/gh/gopcua/opcua.svg?style=shield)](https://circleci.com/gh/gopcua/opcua)
[![GoDoc](https://godoc.org/github.com/gopcua/opcua?status.svg)](https://godoc.org/github.com/gopcua/opcua)
[![GolangCI](https://golangci.com/badges/github.com/gopcua/opcua.svg)](https://golangci.com/r/github.com/gopcua/opcua)
[![GitHub](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/gopcua/opcua/blob/master/LICENSE)

## Quickstart

```sh
go get -u github.com/gopcua/opcua
go run examples/datetime/datetime.go -endpoint opc.tcp://localhost:4840
```

### Current State (20 Mar 2019)

Our goal is to make this the native Go library for OPC/UA. 

This code is not ready for production but we intend to get it there.

We are testing the code against real-world PLCs and other OPC/UA
implementations but this needs to be more formalized. The goal is to have the
examples working with real PLCs. Please let us know if they don't.

We are working on the library and some things are working but others are not. 

Here is what currently works:

 * client connection handshake, create secure channel and session
 * async request/response dispatching on the secure channel
 * support for chunking when receiving (not sending)
 * all structures and enums are generated from official OPC Foundation defintions
 * basic `uasc` listener available but no server implementation
 * start of a high-level Client implementation. See `client.go` and 
   `examples/datetime` for a usage example.
 * decent tests of the binary protocol codec

Here is what is not yet working:

 * `ERR` messages are not yet bubbled up to the caller (not hard but need to do it)
 * service calls need to check `ServiceStatus` and bubble that error up (also not hard)
 * no security protocol support. @dwhutchinson provided the crypto code but it needs to be
   integrated into the network layer. 
 * no high-level server implementation, address space, etc.

## Your Help is Appreciated

If you are looking for ways to contribute you can

 * test the high-level client against real OPC/UA servers
 * add functions to the client or tell us which functions you need for `gopcua` to be useful
 * work on the security layer, server and other components
 * and last but not least, file issues, review code and write/update documentation

Also, if the library is already useful please spread the word as a motivation.

## Supported Features

The current focus is on the OPC UA Binary protocol over TCP. No other protocols are supported at this point.

### Protocol Stack

| Categories     | Features                         | Supported | Notes |
|----------------|----------------------------------|-----------|-------|
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

The current set of supported services is only for the high-level client.

| Service Set                 | Service                       | Supported | Notes        |
|-----------------------------|-------------------------------|-----------|--------------|
| Discovery Service Set       | FindServers                   |           |              |
|                             | FindServersOnNetwork          |           |              |
|                             | GetEndpoints                  |           |              |
|                             | RegisterServer                |           |              |
|                             | RegisterServer2               |           |              |
| Secure Channel Service Set  | OpenSecureChannel             | Yes       |              |
|                             | CloseSecureChannel            | Yes       |              |
| Session Service Set         | CreateSession                 | Yes       |              |
|                             | CloseSession                  | Yes       |              |
|                             | ActivateSession               | Yes       |              |
|                             | Cancel                        |           |              |
| Node Management Service Set | AddNodes                      |           |              |
|                             | AddReferences                 |           |              |
|                             | DeleteNodes                   |           |              |
|                             | DeleteReferences              |           |              |
| View Service Set            | Browse                        | Started   |              |
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
| Subscription Service Set    | CreateSubscription            |           |              |
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

The [gopcua](https://github.com/gopcua) team.

## License

[MIT](https://github.com/gopcua/opcua/blob/master/LICENSE)
