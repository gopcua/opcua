<p align="center">
   <img width="25%" src="https://raw.githubusercontent.com/gopcua/opcua/master/gopher.png">
</p>

<p align="center">
  Artwork by <a href="https://twitter.com/ashleymcnamara">Ashley McNamara</a><br/>
  Inspired by <a href="http://reneefrench.blogspot.co.uk/">Renee French</a><br/>
  Taken from <a href="https://gopherize.me">https://gopherize.me</a> by <a href="https://twitter.com/matryer">Mat Ryer</a>
</p>

<h1 align="center">OPC/UA</h1>

A native Go implementation of the OPC/UA Binary Protocol.

You need go1.13 or higher. We test with the current and previous Go version.
See below for a list of [Tested Platforms](#tested-platforms) and [Supported Features](#supported-features).

[![GitHub](https://github.com/gopcua/opcua/workflows/gopuca/badge.svg)](https://github.com/gopcua/opcua/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gopcua/opcua.svg)](https://pkg.go.dev/github.com/gopcua/opcua)
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/gopcua/opcua/blob/master/LICENSE)
[![Version](https://img.shields.io/github/tag/gopcua/opcua.svg?color=blue&label=version)](https://github.com/gopcua/opcua/releases)

## Note

`v0.2.4` and `v0.2.5` are broken and should not be used. Please upgrade to `v0.2.6` or later.
See [#538](https://github.com/gopcua/opcua/issues/538) for details.

## Quickstart

```sh
# make sure you have go1.17 or higher

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

## Sponsors

The `gopcua` project is sponsored by the following organizations by supporting the active committers to the project:

<table border="0">
   <tr valign="middle">
      <td width="33%">
        <a href="https://northvolt.com/">
          <img alt="Northvolt" src="https://raw.githubusercontent.com/gopcua/opcua/main/logo/northvolt.png">
        </a>
      </td>
      <td width="34%">
        <a href="https://www.evosoft.com/">
          <img alt="evosoft" src="https://raw.githubusercontent.com/gopcua/opcua/main/logo/evosoft.png">
        </a>
      </td>
      <td width="33%">
        <a href="https://www.intelecy.com/">
          <img alt="Intelecy AS" src="https://raw.githubusercontent.com/gopcua/opcua/main/logo/intelecy.png">
        </a>
      </td>
   </tr>
</table>

### Users

We would also like to list organizations which use `gopcua` in production. Please open a PR to include your logo below.
<p align="left">
   <a href="https://strateos.com">
      <img alt="strtaeos" width="10%" src="https://avatars1.githubusercontent.com/u/50255519?s=400&u=3c18028de0bd1a28b604d34d6b239d7a593a7e49&v=4">
   </a>
</p>

## Disclaimer

We are still actively working on this project and the APIs will change.

We have started to tag the code to support go modules and reproducible builds
but there is still no guarantee of API stability.

However, you can safely assume that we are aiming to make the APIs as
stable as possible. :)

The [Current State](https://github.com/gopcua/opcua/wiki/Current-State) was moved
to the [Wiki](https://github.com/gopcua/opcua/wiki).

## Your Help is Appreciated

If you are looking for ways to contribute you can

 * test the high-level client against real OPC/UA servers
 * add functions to the client or tell us which functions you need for `gopcua` to be useful
 * work on the security layer, server and other components
 * and last but not least, file issues, review code and write/update documentation

Also, if the library is already useful please spread the word as a motivation.

## Tested Platforms

`gopcua` is run in production by several companies and with different equipment.
The table below is an incomplete list of where and how `gopcua` is used to provide
some guidance on the level of testing.

We would be happy if you can add your equipment to the list. Just open a PR :)

| Device                                                  | gopcua version    | Environment | By           |
|---------------------------------------------------------|-------------------|-------------|--------------|
| Siemens S7-1500                                         | v0.1.x..latest    | production  | Northvolt    |
| Beckhoff C6015-0010,C6030-0060 on OPC/UA server 4.3.x   | v0.1.x..latest    | production  | Northvolt    |
| Kepware 6.x                                             | v0.1.x..latest    | production  | Northvolt    |
| Kepware 6.x                                             | v0.1.x, v0.2.x    | production  | Intelecy     |
| Cogent DataHub 9.x                                      | v0.1.x, v0.2.x    | production  | Intelecy     |
| ABB Ability EdgeInsight 1.8.X                           | v0.1.x, v0.2.x    | production  | Intelecy     |
| GE Digital Historian 2022 HDA Server                    | v0.3.x            | production  | Intelecy     |
| InfluxDB Telegraf plugin                                | v0.3.x            | ?           | Community    |

## Supported Features

The current focus is on the OPC UA Binary protocol over TCP. No other protocols are supported at this point.

| Categories     | Features                         | Supported | Notes       |
|----------------|----------------------------------|-----------|-------------|
| Encoding       | OPC UA Binary                    | Yes       |             |
|                | OPC UA JSON                      |           | not planned |
|                | OPC UA XML                       |           | not planned |
| Transport      | UA-TCP UA-SC UA Binary           | Yes       |             |
|                | OPC UA HTTPS                     |           | not planned |
|                | SOAP-HTTP WS-SC UA Binary        |           | not planned |
|                | SOAP-HTTP WS-SC UA XML           |           | not planned |
|                | SOAP-HTTP WS-SC UA XML-UA Binary |           | not planned |
| Encryption     | None                             | Yes       |             |
|                | Basic128Rsa15                    | Yes       |             |
|                | Basic256                         | Yes       |             |
|                | Basic256Sha256                   | Yes       |             |
| Authentication | Anonymous                        | Yes       |             |
|                | User Name Password               | Yes       |             |
|                | X509 Certificate                 | Yes       |             |

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
| View Service Set            | Browse                        | Yes       |              |
|                             | BrowseNext                    | Yes       |              |
|                             | TranslateBrowsePathsToNodeIds |           |              |
|                             | RegisterNodes                 | Yes       |              |
|                             | UnregisterNodes               | Yes       |              |
| Query Service Set           | QueryFirst                    |           |              |
|                             | QueryNext                     |           |              |
| Attribute Service Set       | Read                          | Yes       |              |
|                             | Write                         | Yes       |              |
|                             | HistoryRead                   | Yes       |              |
|                             | HistoryUpdate                 |           |              |
| Method Service Set          | Call                          | Yes       |              |
| MonitoredItems Service Set  | CreateMonitoredItems          | Yes       |              |
|                             | DeleteMonitoredItems          | Yes       |              |
|                             | ModifyMonitoredItems          | Yes       |              |
|                             | SetMonitoringMode             |           |              |
|                             | SetTriggering                 |           |              |
| Subscription Service Set    | CreateSubscription            | Yes       |              |
|                             | ModifySubscription            |           |              |
|                             | SetPublishingMode             |           |              |
|                             | Publish                       | Yes       |              |
|                             | Republish                     |           |              |
|                             | DeleteSubscriptions           | Yes       |              |
|                             | TransferSubscriptions         |           |              |

## Authors

The [Gopcua Team](https://github.com/gopcua/opcua/graphs/contributors).

If you need to get in touch with us directly you may find us on [Keybase.io](https://keybase.io)
but try to create an issue first.

## License

[MIT](https://github.com/gopcua/opcua/blob/master/LICENSE)
