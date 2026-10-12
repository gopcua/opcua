# tests/spec

These specs run the gopcua client against a scripted OPC UA server. Each
part of the standard has its own directory, and each directory holds one file
per clause, named after the clause's title.

The whole suite skips unless the environment variable `SPECTEST=1` is set, so
a plain `go test ./...` stays fast and the skip is visible in every package's
output. Run the suite with:

    SPECTEST=1 go test -race -timeout 30m ./tests/spec/...

The suite fails on `main`. No spec is skipped because it is expected to fail,
and nothing marks a failure as expected: read the failure output to decide
what produced it. A fault that cannot apply to a scenario skips its case,
visibly.

Select specs with `ginkgo.focus`, a regular expression matched against a
spec's path: the chain of its container texts down to its own text. A
matrix case's path is its clause, its scenario, its fault and its check.
A hand-written spec's path is whatever chain of containers its spec
declares. One case, the SessionSurvives scenario under RequestLost/Publish:

    SPECTEST=1 go test -race -timeout 30m ./tests/spec/part4/ -ginkgo.focus='SessionSurvives RequestLost/Publish'

One check the matrix plans for every fault:

    SPECTEST=1 go test -race -timeout 30m ./tests/spec/part4/ -ginkgo.focus='KeepsSubscriptionID'

The part4 binary also reads `SPECTEST_FOCUS`, whose value becomes the focus
when no `-ginkgo.focus` flag was passed.

A rule is a `matrix.Rule` value in the clause file of the clause it cites.
The facts a rule's Check reads are the clause package's `ruleFacts` type,
declared in `part4/helpers_test.go`. An invariant is a `matrix.Invariant`
value in `part4/invariants_test.go`. The runner's self-spec, which registers
stand-in scenarios and runs them the way a clause file does, runs in
`internal/matrix`. No code under `internal/` asserts what a clause demands:
`internal/` holds the harness, the faults and the matrix runner, and what a
clause demands belongs in its clause file under `part4/`.

A failure message starting with `spectest:` reports a fault in the harness
itself. Any other failure is a check on client behaviour that did not hold.

To script a server answer, call `ScriptedServer.WaitHeldPublish()`, which
returns a `HeldPublish` to answer; use `Subscription` for retained
notifications and Republish faults. To drop the connection at a chosen
moment, use `Relay.Cut` or `Relay.CutAt`.
`part4/reestablishing_connections_test.go` shows a worked example.

A clause file can answer a service the scripted server leaves to the in-tree
server's default handlers. Pass
`harness.WithHandler(id.BrowseRequest_Encoding_DefaultBinary, handle)` to
`harness.New`, and `handle` answers every Browse request instead of the
default handler. A handler for a service the scripted server already answers
fails the test as a harness fault. Any service can be a fault target:
`harness.MessageOf(&ua.BrowseRequest{})` returns the `message.Message` that
`Relay.CutAt` and `Relay.DelayAt` take.
