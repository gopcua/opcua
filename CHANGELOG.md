# Changelog

## v0.5.3 (07 Dec 2023)

* Fix unchecked type assertion in Subscription Stats (#693)
* setSession to nil in recreateSession action to avoid unnecessary CloseSession (#700)
* StatusBadSessionNotActivated in updateNamespaces call during recreateSession action while reconnecting (#673)

## v0.5.2 (18 Oct 2023)

* feat(encode): print written hex on debugCodec flag (#685)
* fix: ReferenceNodes usage with mask set (#683)
* Empty policyURI fallback on SecureChannel SecurityPolicyURI (#669)
* feat: add support for AuthPrivateKey (#681)
* Fixed panic if h.MessageSize < hdrlen bytes. (#692)
* Problem with using ReferencedNodes (#682)
* Running examples/browse.go returns EOF error (#550)
* Empty session policyURI (#668)
* Failed to open a secure channel with AuthCertificate and different certificates (#671)

## v0.5.1 (22 Aug 2023)

* refactor: make NewClient return an error (#674)
* feat: add support for FindServers and FindServersOnNetwork (#675)
* Readme: adjust Services section (#676)
* Update github actions (#677)

## v0.5.0 (14 Aug 2023)

* Drop WithContext methods and require all methods to have a context (#554)

## v0.4.1 (14 Aug 2023)

* Update the schema to v1.05.02-2022-11-01 and regenerate code (#589)
* fix: handle extra padding if key length > 2048 (#648)
* Add B&R Automation PC 3100 to the list of equipments (#663)
* uasc: return an error for invalid uri/mode combinations with None (#664)
* go1.21 and python3.11 (for testing)

## v0.4.0 (13 Jun 2023)

* Bugfix: Close session properly if activation fails (#657)
* v0.4.0 preparation (#662)

## v0.3.15 (25 May 2023)

* Panic in secure_channel.go (#640)

## v0.3.14 (22 May 2023)

* Remove 'if err == nil' anti-pattern (#652)
* Improve error handling (#653)
* Add United Manufacturing Hub as user (#647)

## v0.3.13 (23 Mar 2023)

* go1.20 (#645)
* Add missing HistoryRead methods (#586)

## v0.3.12 (22 Mar 2023)

* set SecureChannel nil in Close() method (#596)
* Revise error message (#643)
* dependabot: bump golang.org/x/crypto (#644)
* If no subscriptions -> monitor infinite loop of reconnections (#597)
* skip StatusBadNoSubscription in monitor loop (#599)
* Trigger resumeSubscriptions only if there are subscriptions (#641)

## v0.3.11 (1 Feb 2023)

* Decoder fails to decode type which converts to time.Time (#633)

## v0.3.10 (25 Jan 2023)

* drop io/ioutil (#627)
* uacp: honor the context deadline during the handshake (#629)

## v0.3.9 (12 Jan 2023)

* Ignore empty filename in RemoteCertificateFile (#626)

## v0.3.8 (08 Dec 2022)

* Fix nil subscription stats to return error (#602)
* `log.Fatal` called when a certificate fails to load (#616)
* Bump go version to 1.19

## v0.3.7 (05 Oct 2022)

* Stop uasc token expiration timer. Resource leak (#608)

## v0.3.6 (29 Sep 2022)

* Relax node id parser (#607)

## v0.3.5 (15 Jun 2022)

* Change encryption URI for aes128Sha256RsaOaep to w3.org (#585)

## v0.3.4 (6 May 2022)

* ua: do not panic if the same extension object is registered multiple times (#579)
* use errors.Is and errors.As (#578)
* ua: log unknown extension object type id (#576)

## v0.3.3 (8 Apr 2022)

* Don't panic on close (#562)
* Set minimum Go version to go1.17 (#573)
* Refactor the use of the `subMux` lock (#572)

## v0.3.2 (14 Mar 2022)

* Add support for arrays (#564)

## v0.3.1 (27 Jan 2022)

* Refresh cached namespaces on reconnect (#552)
* Add more `WithContext(ctx)` methods and use context in more places (#555)

## v0.3.0 (21 Jan 2022)

* Add `WithContext(ctx)` variants to all methods of `Client` and `Node` and migrate existing methods
  to use `context.Background()`. The existing methods without a context are deprecated and starting
  with v0.5.0 we will drop the `WithContext(ctx)` prefix and all `Client` and `Node` methods will
  require a `context`. (#541, #542, #548, #549)

## v0.2.7 (18 Jan 2022)

* Add a `FindNamespace` method to `Client`. (#546)

## v0.2.6 (4 Jan 2022)

* Fix invalid session id regression introduced with v0.2.4 (#539)
