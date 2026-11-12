# What this package does not test

Every OPC 10000-4 §6.7 behaviour the harness can reach has a row. The rest of
the clause's client statements are either scored by a row, or — for "shall
re-establish the connection by creating a new SecureChannel and activating the
Session" — enforced as a precondition by `reconnectSetupFailed` in seven rows,
with no row taking it as its subject, or blocked on a fixture capability
`harness.go` does not have. The table below is the blocked ones. This file
exists so a reader can tell them from a behaviour nobody thought of.

| capability the harness lacks | blocks |
|---|---|
| a fixture that fails ActivateSession | the other half of "The Client shall only create a new Session if ActivateSession fails"; "TransferSubscriptions is used to transfer the Subscription to the new Session" and "If TransferSubscriptions fails, the Client needs to create a new Subscription" |
| a secured endpoint — security policy, application instance certificate, user token | "the Client shall use the same security policy, application instance certificate and the same user credential"; "If the Client ApplicationInstanceCertificate is updated, the Client shall create a new Session" and shall try to transfer its Subscriptions to it (which also needs mid-run certificate rotation) |
| secure-channel-level `Bad_CertificateInvalid` plus a GetEndpoints arm | "In case of security failures, the Client shall use the GetEndpoints Service" |
| a Call arm plus node and method ids in `requestRecord` | the Client "should use the Method ResendData or should read the values of all data MonitoredItems" |
| an application-side value cache | the Client "should make sure that it does not overwrite newer data in the Client with older values provided through Republish" |
| a configurable monitored node plus a ServerStatus node | "A Client should subscribe for the State Variable in the ServerStatus to detect shutdown" |
| a server-initiated connection | reverse connect |
| a run that goes silent without closing the transport | "The preferred mechanism for a Client to monitor the connection status is through the keep-alive of the Subscription" |

Some of these need more than the table says.

**The failing-activation fixture is not another `runOptions` field.** `answer`
grants every ActivateSession unconditionally, and `recordActivationFault` marks
a whole run unscorable if the fixture ever answers one with anything but a clean
grant — the assumption the seven activation-filtering helpers rest on, and the
one `harness_activation_rule_test.go` exists to hold. That capability is a
rework of the package's own self-check, which is what makes it the most
expensive item here.

**ResendData has a reachable stimulus and an unmeasurable response.**
`runOptions{}` already answers every Republish `Bad_MessageNotAvailable`, which
is exactly the clause's trigger. But `requestRecord` carries `Conn`, `Service`,
`RepublishSeq` and `Acks` and no node or method id, so a Call cannot be
identified as ResendData, and a read of the monitored items cannot be told from
gopcua's own namespace-array read, which is what the `ReadRequest` arm answers.
It is nearer to reachable than the rest.

**The keep-alive item is the library's, unlike the two below.** `publishTimeout` in
`subscription.go` already computes the interval a keep-alive is expected within, and `publish` in
`client_sub.go` answers the timeout that produces with *ignore and continue the loop*, so the
signal is built and discarded inside gopcua; none of it is the application's to choose. What the
harness lacks is a run that stops answering without closing the transport. The fixture already
withholds Publish responses, so the silence is there; what is not is a scenario that stalls instead
of dropping, and a `runRecord` whose readings are not all anchored on `DropAt` and `DropConn`.

**Two of these are mis-addressed, not merely blocked.** What to monitor is
the application's choice, not the library's, and gopcua keeps no value cache. So
the ServerStatus and overwrite-ordering items would not be client-library
conformance rows even once the capability existed — the case question 6 of the
README describes, where every library leaves the behaviour to the application.

## A gap that is not a missing capability

The table is read as "capability the harness lacks → what it blocks", and this
one has no capability to name, so it is written out here instead of as a row.

In the gopcua client, pause and resume are two signals read by the same
`select`, so Go picks between them at random: a resume picked first is
discarded as redundant, and the pause behind it parks the publish loop with
nothing left to wake it. This suite cannot test that, and no fixture
capability would change it. The defect is a scheduling race between goroutines
inside the client, and this suite is a black-box harness that drives the client
through its public API against a scripted server. Putting it in the table would
promise a row that will never be written.

The defect is tracked as https://github.com/gopcua/opcua/issues/895, and
nothing in this tree covers it. A test that can see it has to race pause
against resume many times inside one run: a different instrument from this
one, not a larger fixture for it.
