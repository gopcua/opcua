<p align="center">
   <img width="50%" src="https://raw.githubusercontent.com/gopcua/opcua/master/gopher.png">
</p>

<p align="center">
  Artwork by <a href="https://twitter.com/ashleymcnamara">Ashley McNamara</a> -
  Inspired by <a href="http://reneefrench.blogspot.co.uk/">Renee French</a> -
  Taken from <a href="https://gopherize.me">https://gopherize.me</a> by <a href="https://twitter.com/matryer">Mat Ryer</a>
</p>

<h1 align="center">OPCUA</h1>

A native Go implementation of the OPC/UA Binary Protocol.

You need go1.11 or higher. We test with the current and previous Go version.

[![CircleCI](https://circleci.com/gh/gopcua/opcua.svg?style=shield)](https://circleci.com/gh/gopcua/opcua)
[![GoDoc](https://godoc.org/github.com/gopcua/opcua?status.svg)](https://godoc.org/github.com/gopcua/opcua)
[![GolangCI](https://golangci.com/badges/github.com/gopcua/opcua.svg)](https://golangci.com/r/github.com/gopcua/opcua)
[![GitHub](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/gopcua/opcua/blob/master/LICENSE)

## Quickstart

```sh
# make sure you have go1.11 or higher

# install library
go get -u github.com/gopcua/opcua

# get current date and time 'ns=0;i=2258'
go run examples/datetime/datetime.go -endpoint opc.tcp://localhost:4840

# read the server version
go run examples/read/read.go -endpoint opc.tcp://localhost:4840 -node 'ns=0;i=2261'

# get the current date time using different security and authentication modes
go run examples/crypto/*.go -endpoint opc.tcp://localhost:4840 -cert path/to/cert.pem -key path/to/key.pem -sec-policy Basic256 -sec-mode SignAndEncrypt

# checkout examples/ for more examples...
```

## Disclaimer

We are still actively working on this project and the APIs will change.

However, you can safely assume that we are aiming to make the APIs as
stable as possible. :)

### Current State (19 Apr 2019)

 * `ERR` messages are bubbled up to the caller
 * security protocol support. See https://github.com/gopcua/opcua/wiki/Supported-Devices
   to get a list of devices/applications we have tested various crypto and authentication
   methods. Please add your own. See `examples/crypto` for an example.
 * Debug messages are now disabled by default.

We are getting closer to use this for our first production use cases. Subscription support
is certainly high on the list for the client as is a Server implementation. Let us know what
is missing right now to make this library useful for you and we can focus on this first.

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

## Authors

The [Gopcua Team](https://github.com/gopcua/opcua/graphs/contributors).

## Supported Features

The current focus is on the OPC UA Binary protocol over TCP. No other protocols are supported at this point.

| Categories     | Features                         | Supported | Notes |
|----------------|----------------------------------|-----------|-------|
| Encoding       | OPC UA Binary                    | Yes       |       |
|                | OPC UA JSON                      |           | not planned |
|                | OPC UA XML                       |           | not planned |
| Transport      | UA-TCP UA-SC UA Binary           | Yes       |       |
|                | OPC UA HTTPS                     |           | not planned |
|                | SOAP-HTTP WS-SC UA Binary        |           | not planned |
|                | SOAP-HTTP WS-SC UA XML           |           | not planned |
|                | SOAP-HTTP WS-SC UA XML-UA Binary |           | not planned |
| Encryption     | None                             | Yes       |       |
|                | Basic128Rsa15                    | Yes       |       |
|                | Basic256                         | Yes       |       |
|                | Basic256Sha256                   | Yes       |       |
| Authentication | Anonymous                        | Yes       |       |
|                | User Name Password               | Yes       |       |
|                | X509 Certificate                 | Yes       |       |

### Services

The current set of supported services is only for the high-level client.

| Service Set                 | Service                       | Supported | Notes        |
|-----------------------------|-------------------------------|-----------|--------------|
| Discovery Service Set       | FindServers                   |           |              |
|                             | FindServersOnNetwork          |           |              |
|                             | GetEndpoints                  | Yes       |              |
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
|                             | BrowseNext                    | Started   |              |
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

## License

[MIT](https://github.com/gopcua/opcua/blob/master/LICENSE)
