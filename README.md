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

We support the current and previous major Go release.
See below for a list of [Tested Platforms](#tested-platforms) and [Supported Features](#supported-features).

[![GitHub](https://github.com/gopcua/opcua/workflows/gopuca/badge.svg)](https://github.com/gopcua/opcua/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gopcua/opcua.svg)](https://pkg.go.dev/github.com/gopcua/opcua)
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/gopcua/opcua/blob/master/LICENSE)
[![Version](https://img.shields.io/github/tag/gopcua/opcua.svg?color=blue&label=version)](https://github.com/gopcua/opcua/releases)

## v0.5.x BREAKING CHANGES

* `v0.5.0` released on 14 Aug 2023: all `Client` methods must have a context
* `v0.5.1` released on 22 Aug 2023: the `NewClient` function returns an error

In `v0.3.0` on 21 Jan 2022 release we added `WithContext` variants for all methods
to avoid a breaking change. All existing methods without a context had a disclaimer
that with `v0.5.0` their signature would change to include the context
and that the `WithContext` method would be removed. 

We missed to update the `NewClient` function in `v0.5.0` which was fixed
in `v0.5.1`.

Please update your code and let us know if there are any issues!

Thank you!

Your GOPCUA Team

## Quickstart

```sh
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

<p align="left">
   <a href="https://umh.docs.umh.app">
      <img alt="united manufacturing hub" width="10%" src="logo/united-manufacturing-hub.jpg">
   </a>
</p>

### Projects using gopcua

`gopcua` is not only utilized in production environments, but it also serves as a critical component in other larger projects. Here are some projects that rely on `gopcua` for their functionality:

- [Telegraf](https://github.com/influxdata/telegraf): This plugin-driven server agent is used for collecting and sending metrics. It leverages `gopcua` to extract data from OPC-UA servers and insert it into InfluxDB. Telegraf supports both polling and subscribing methods for data acquisition.
- [benthos-umh](https://github.com/united-manufacturing-hub/benthos-umh): This project is built upon the [benthos](https://github.com/benthosdev/benthos) stream-processing framework. It utilizes `gopcua` to extract data from OPC-UA servers and forwards the information to MQTT or Kafka brokers. benthos-umh currently supports polling for data collection.

## Disclaimer

We are still actively working on this project and the APIs will change.

However, you can safely assume that we are aiming to make the APIs as
stable as possible since the code is in use in several large scale
production environments.

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
| B&R Automation PC 3100                                  | v0.3.x            | production  | ACS          |
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

Here is the current set of supported services. For low-level access use the client `Send` function directly.

| Service Set                 | Service                       | Client | Notes        |
|-----------------------------|-------------------------------|--------|--------------|
| Discovery Service Set       | FindServers                   | Yes    |              |
|                             | FindServersOnNetwork          | Yes    |              |
|                             | GetEndpoints                  | Yes    |              |
|                             | RegisterServer                |        |              |
|                             | RegisterServer2               |        |              |
| Secure Channel Service Set  | OpenSecureChannel             | Yes    |              |
|                             | CloseSecureChannel            | Yes    |              |
| Session Service Set         | CreateSession                 | Yes    |              |
|                             | CloseSession                  | Yes    |              |
|                             | ActivateSession               | Yes    |              |
|                             | Cancel                        |        |              |
| Node Management Service Set | AddNodes                      |        |              |
|                             | AddReferences                 |        |              |
|                             | DeleteNodes                   |        |              |
|                             | DeleteReferences              |        |              |
| View Service Set            | Browse                        | Yes    |              |
|                             | BrowseNext                    | Yes    |              |
|                             | TranslateBrowsePathsToNodeIds |        |              |
|                             | RegisterNodes                 | Yes    |              |
|                             | UnregisterNodes               | Yes    |              |
| Query Service Set           | QueryFirst                    |        |              |
|                             | QueryNext                     |        |              |
| Attribute Service Set       | Read                          | Yes    |              |
|                             | Write                         | Yes    |              |
|                             | HistoryRead                   | Yes    |              |
|                             | HistoryUpdate                 |        |              |
| Method Service Set          | Call                          | Yes    |              |
| MonitoredItems Service Set  | CreateMonitoredItems          | Yes    |              |
|                             | DeleteMonitoredItems          | Yes    |              |
|                             | ModifyMonitoredItems          | Yes    |              |
|                             | SetMonitoringMode             |        |              |
|                             | SetTriggering                 |        |              |
| Subscription Service Set    | CreateSubscription            | Yes    |              |
|                             | ModifySubscription            |        |              |
|                             | SetPublishingMode             |        |              |
|                             | Publish                       | Yes    |              |
|                             | Republish                     |        |              |
|                             | DeleteSubscriptions           | Yes    |              |
|                             | TransferSubscriptions         |        |              |

## Authors

The [Gopcua Team](https://github.com/gopcua/opcua/graphs/contributors).

If you need to get in touch with us directly you may find us on [Keybase.io](https://keybase.io)
but try to create an issue first.

## License

[MIT](https://github.com/gopcua/opcua/blob/master/LICENSE)
