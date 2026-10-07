# tests/spec

These specs run the gopcua client against a scripted OPC UA server. The
directory holds one file per spec section, named by its clause number so the
files sort in the standard's order, and each spec is labelled with every
clause it cites.

The whole suite skips unless the environment variable `SPECTEST=1` is set, so
a plain `go test ./...` stays fast and the skip is visible in every package's
output. Run the suite with:

    SPECTEST=1 go test -race -timeout 30m ./tests/spec/...

The default run skips every spec labelled `known-defect`, a spec measured to
fail on `main`, so it passes on `main`. To run every spec, including the known
defects:

    SPECTEST=1 go test -race -timeout 30m ./tests/spec/part4/ -ginkgo.label-filter=''

Until gopcua/opcua#897 merges, that run fails under `-race`. Spec C1 calls
`Close` while the client is redialling, which reproduces the data race of
issue #883, and the race detector then fails the test binary.

Under the `ginkgo` CLI an empty `--label-filter` never reaches the test
binary, so the known defects stay skipped (measured with ginkgo v2.33.0); set
the environment variable to a filter that matches everything:

    SPECTEST=1 SPECTEST_LABEL_FILTER='!known-defect || known-defect' go run github.com/onsi/ginkgo/v2/ginkgo@v2.33.0 ./tests/spec/part4

A spec's labels name what it cites. `P4-6.7` cites Part 4 clause 6.7
(https://reference.opcfoundation.org/Core/Part4/v105/docs/<clause>).
`should` marks a recommendation and `interop` tolerates server behaviour the
standard does not cover. `issue-N` and `pr-N` cite GitHub.

A known-defect label is a prediction that its spec fails on `main`. A later
PR gates CI on it: a run that makes a known-defect spec pass fails the gate,
so a fix removes the label.

A failure message starting with `spectest:` reports a fault in the harness
itself. Any other failure reports client behaviour.

To script a server answer, call `ScriptedServer.WaitHeldPublish()`, which
returns a `HeldPublish` to answer; use `Subscription` for retained
notifications and Republish faults. To drop the connection at a chosen
moment, use `Relay.Cut` or `Relay.CutAt`.
`06_07_reestablishing_connections_test.go` shows a worked example.
