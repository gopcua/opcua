# Server tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to the server package.

A row here drives the **in-tree gopcua server** — production code, not a
scripted server — through its public API, with a real gopcua client
connected over the wire, because the behaviours scored so far are
properties of the server's own call graph: the re-entrancy of its
notification path and the locking around its namespaces. A scripted
server would score the script. Item 3's reading here is therefore the
shared fixture in `harness.go`, which brings up the server, the
MapNamespace and the client together; it is checkable the way it is in
the other packages: `grep -c "See harness.go" tests/server/*_test.go`
prints 1 for every row and 0 for `harness.go`. `race_on.go` and
`race_off.go` are not rows either: they hold the build-tagged
`raceDetector` constant the race row reads, and a build constraint
applies to a whole file, so the constant cannot live with the row.
Where a row needs the server to misbehave in a way the in-tree server
cannot, that is a fixture gap and is named here or in a NOT-TESTED.md,
not worked around.

The package name is `server`, as the directory is; the production package
`github.com/gopcua/opcua/server` is a different import path.

This package has two pieces of machinery of its own:

- **A wall clock scores a wedge.** A hung goroutine cannot be observed
  in-process except by outliving a deadline, so the write-half rows give
  the call one (`respond` in `harness.go`) and score the deadline, not
  the hang. The deadline is not a timing assumption: `SetValue` writes
  one map entry and pushes one notification onto a buffered channel.
  One caution for the author wiring the bodies: a wedged server is slow
  to tear down as well - measured, a row that failed at its 5s deadline
  took a further ~30s in cleanup, because `srv.Close` waits on the very
  locks the wedge holds - so the deadline's failure line is the row's
  verdict and the cleanup lag after it is not a second failure.
- **A race is scored in a child process.** The Go race detector fails
  the test binary itself when it reports — measured: a run of the race
  row failed with `WARNING: DATA RACE` from the runtime, with none of
  the row's own arms firing — so no `knownBug` arm inside the row can
  intercept a race. The race row
  therefore runs its window in a child of the test binary and reads the
  child's output and exit. A child whose output holds `WARNING: DATA
  RACE` exits **nonzero** - the testing framework fails the very test
  that raced - so the exit code is material only when the output holds
  no report. Keying on the exit code alone makes the row decline every
  run. The known-bug arm keys on the detector's report naming the row's
  Attribute frame, a presence a fixed library does not show, and the
  deletion arm keys on a clean child that completed at least ten reads
  inside a two-second window - the floor is what stops an empty run
  passing, so it needs a number and not just a name. A
  plain `go test` without `-race` cannot score the row at all, and the
  row declines in that case; `make test` runs `-race ./...`.

## Where a row's evidence comes from here

`../README.md`'s rule speaks of bytes: a row's verdict rests on bytes from
outside gopcua, because a value that gopcua encodes and gopcua decodes
back has only asked gopcua whether it agrees with itself. **As written,
the rule does not apply to the rows in this package, because none of them
scores bytes.** A deadlock is a control-flow property and a data race is a
memory-model property; no row here compares a wire encoding to anything.

Its principle applies with full force, and for a reason the encoder rows
do not have: the client and the server in every row here are **both
gopcua**. A row whose verdict turns on the server's answer reaching the
client's callback is a round trip in server clothing — a symmetric bug on
both sides of the wire is invisible to it. The equivalent of "bytes from
outside gopcua" for a server row is **an instrument outside gopcua's two
halves**: the race detector's happens-before analysis, the runtime's
fatal map abort, a wall clock on a goroutine the library does not know
about, or another stack's client. The three rows stand on the first three
of those. What would not be evidence: anything scored by asking the
gopcua client whether it liked what the gopcua server sent.

Two consequences, one per kind of assertion a server row can make:

- **The verdict may rest on the instrument; the trimmings may not.** The
  race row's monotonic-sequence check (a read that moves backwards is a
  torn observation) compares two values that both crossed the wire
  through gopcua codecs, so a symmetric codec drift cancels in it. It is
  there to keep the window non-vacuous, not to score the wire, and its
  header says so.
- **Where a server row does assert a value that crossed the wire, the
  original rule applies again in its original form.** The write-half
  rows' notification check compares the written literal with what the
  subscription received: the literal is from outside gopcua, but the
  received value went through both of its codecs, so the check scores
  *delivery*, not bytes. `../README.md` permits a round trip when another
  row holds the literal bytes for the same behaviour and this one names
  it. No row anywhere holds the literal bytes for an Int32 inside a
  Variant — the encoding package's rows cover DateTime, the nullable
  built-ins and Variant array-length policing — so there is nothing to
  name, and the write-half rows say so in their headers instead of
  pretending. What would close it: an encoding row in
  `../encoding/` holding those bytes, which the write-half rows would
  then name; or driving the same write with a second stack's client.

## What a reference stack means here

For a row that drives the server, the Q6 reading of "another
implementation" is **another stack's server source**: open62541's or
UA-.NETStandard's handling of the same behaviour, read to check the row
against a known-good implementation. Where a row's verdict rests on a
*client's* interpretation of what the server sent — any row whose
evidence is "the client received" — the reading is another stack's
*client*, because that is the half whose judgement is being trusted.
None has been read for the three rows here; their headers say so, so
silence means nobody checked rather than nothing to report.

## The knownBug ledger, 2026-11-06

One constant stands per row below, declared inside its test function.

| row | state |
|---|---|
| `read_while_writing_is_race_free_test.go` | `knownBug` stands: `MapNamespace.Attribute` reads `ns.Data` without the namespace lock (`server/namespace_map.go:195 and :256`) while `SetValue` writes it under the lock. Measured 2026-11-05: `WARNING: DATA RACE` under `go test -count=1 -race -run TestReadWhileWritingIsRaceFree ./tests/server/`. |
| `setvalue_on_a_subscribed_tag_returns_test.go` | no constant; gopcua passes it as it stands (measured 2026-11-06, 1.41s under `-race`). It exists to catch the regression the fix for the row above invites: with an RLock added to `Attribute` — the obvious fix — `SetValue` wedges (measured, 6.35s, "did not return within 5s"). |
| `setattribute_on_a_subscribed_tag_returns_test.go` | no constant; gopcua passes it as it stands (measured 2026-11-06, 1.45s under `-race`). Same guard, at the call site a client Write lands on. |

The constant in `read_while_writing_is_race_free_test.go` is expected to
be closed by a change to `Attribute`'s locking. The two write-half rows
are the guard that such a change does not trade the race for a deadlock:
measured, an RLock in `Attribute` turns both write call sites into
self-deadlocks, because the notification path re-enters `Attribute` on
the calling goroutine while the caller holds the namespace lock
(`server/monitored_item_service.go:130`).

## What a cold author still needs

- **An encoding row holding the literal bytes of an Int32 inside a
  Variant**, in `../encoding/`, if the write-half rows' notification
  checks are ever to be more than delivery checks — see "Where a row's
  evidence comes from here".
- **Verification of the OPC 10000-3 §5.6.2 8.58 citation** ("In general
  Atomicity is expected of OPC UA read and write operations"), quoted in
  all three headers as the specification's nearest clause: it is not
  checkable in this tree, and the first author with the Part 3 text in
  front of them should confirm or correct it.
- **Q6 reads**: at least one other server's source read against each
  row, per the six questions in `../README.md`. None has been read.
- **A wire recorder around the in-tree server**, if a future row's
  verdict must rest on bytes the server actually sent. The recorder in
  `tests/reconnect/harness.go` is welded into that package's scripted
  server's accept loop; nothing in this tree can watch the in-tree
  server's transport. Nothing here needs it yet — the three rows score
  no bytes — so it is a named gap, not built.
