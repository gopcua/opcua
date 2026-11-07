# Transport tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to the transport package.

A row here drives **gopcua's transport dial** — `uacp.Dialer.Dial`
(uacp/conn.go:69) — against a **scripted silent peer**: a plain TCP peer
built on `net`, not gopcua's uacp server. The peer accepts one connection,
reads and records that a HEL message header arrived, and never answers.
The rows score what `Dial` does while the peer withholds its Acknowledge,
and which address of a name it dials; gopcua's uacp server would answer,
and the silence is the stimulus, so a scripted peer is what the behaviours
need. The fixture is small and lives entirely in `harness.go`; item 3's
reading here is that fixture, and it is checkable the way it is in the
other packages: `grep -c "See harness.go" tests/transport/*_test.go`
prints 1 for every row and 0 for `harness.go`.

## Why no OPC UA clause binds these behaviours

The nearest clause for all three rows is OPC 10000-6 §7.1.3, which
describes the handshake the rows interrupt. What it says is descriptive:
it fixes the order — the client's Hello, the server's Acknowledge, the
OpenSecureChannel after — and it says a client may be handed a domain
name. No sentence in it is a shall about when a client gives up waiting,
or about what a client does with the several addresses a name resolves
to. So each row says plainly that no OPC UA clause binds its behaviour,
and names the Go contract it scores instead: `context.Context`'s
cancellation and deadline for the two handshake rows, `net.Dialer`'s
documented behaviour of trying each address a name resolves to for the
address row. The Go doc line is quoted verbatim in the row's header, from
`go env GOROOT`.

## The package's machinery

- **A wall clock scores a blocked dial.** A `Dial` that never returns
  cannot be observed except by outliving a bound, so the handshake rows
  give the call one (`unblock` in `harness.go`) and score the return, not
  the block. The bound is not a timing assumption: a library that honours
  the context unblocks at once — the deadline path sets the connection's
  read deadline from the context (uacp/conn.go:236-237) and a teardown
  close ends the read with `io.EOF` — so the bound exists to let a fixed
  library finish, not to wait out a broken one. It also bounds the setup
  checks' waits for the HEL. Every listener and connection is closed in
  `t.Cleanup`, so a `Dial` left blocked on main is released when the test
  ends.

- **`localhost` gives the address row its two addresses.** On the host
  this suite was authored on, `localhost` resolves to two addresses,
  `::1` and `127.0.0.1` (measured with `getaddrinfo`), and the row
  resolves the name itself with `net.Resolver.LookupIPAddr` — the same
  call `ResolveEndpoint` makes (uacp/endpoint.go:40) — so the row and the
  library agree on the order. The row listens only on the **last**
  address and dials `opc.tcp://localhost:<port>`; on main the HEL never
  arrives there, because `ResolveEndpoint` keeps only `addrs[0]`
  (uacp/endpoint.go:50) and `Dialer.Dial` dials that one address
  (uacp/conn.go:83). The row **declines** when `localhost` resolves to a
  single address — there is no next address, so the behaviour cannot be
  constructed — and when anything accepts a connection on the first
  address at that port, because a HEL at the last address would then not
  be evidence about the first address's refusal.

## The knownBug ledger

Two constants stand, one per row, declared inside their test functions;
both are written with the bodies. They are expected to be closed by the
same fix: honouring the context's cancellation in `Handshake` closes
`handshake_gives_up_when_its_context_is_cancelled_test.go`, and iterating
the resolved addresses closes
`dial_reaches_a_host_name_on_its_next_address_test.go`. No constant
stands for the deadline row: gopcua passes it as it stands, and it exists
to guard the deadline path against a change made to the cancellation
path — the obvious fix for the cancellation row is to select on
`ctx.Done()` beside the read, and a change that broke the deadline while
fixing the cancellation would pass that row and fail this one.
