# tests/spec

These specs run the gopcua client against a scripted OPC UA server. The
directory holds one file per spec section, named by its clause number so the
files sort in the standard's order, and each spec is labelled with every
clause it cites.

Run the suite with `go test -race ./tests/spec/...`. The default run skips
every spec labelled `known-defect`, a spec measured to fail on `main`, so it
passes on `main`. To run every spec, including the known defects:

    go test -race -timeout 30m ./tests/spec/part4/ -ginkgo.label-filter=''

Under the `ginkgo` CLI an empty `--label-filter` never reaches the test
binary, so the known defects stay skipped (measured with ginkgo v2.33.0); set
the environment variable to a filter that matches everything:

    SPECTEST_LABEL_FILTER='!known-defect || known-defect' go run github.com/onsi/ginkgo/v2/ginkgo@v2.33.0 ./tests/spec/part4

A spec's labels name what it cites. `P4-6.7` cites Part 4 clause 6.7
(https://reference.opcfoundation.org/Core/Part4/v105/docs/<clause>).
`should` marks a recommendation and `interop` tolerates server behaviour the
standard does not cover. `issue-N` and `pr-N` cite GitHub. `racy` marks a
spec whose outcome depends on a goroutine interleaving the harness cannot
force.

CI runs the known-defect specs in their own job. It fails when a
known-defect spec passes, unless the spec is also labelled `racy`, which
only prints a warning; a fix that makes a known-defect spec pass removes
that label.

A failure message starting with `spectest:` reports a fault in the harness
itself. Any other failure reports client behaviour.

To script a server answer, call `ScriptedServer.WaitHeldPublish()`, which
returns a `HeldPublish` to answer; use `Subscription` for retained
notifications and Republish faults. To drop the connection at a chosen
moment, use `Relay.Cut` or `Relay.CutAt`.
`06_07_reestablishing_connections_test.go` shows a worked example.
