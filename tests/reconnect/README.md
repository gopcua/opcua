# Reconnect tests

Each file here is one Go test for one client behaviour in the OPC UA
reconnect space, named after that behaviour. The specification clause it
checks is in the file's opening comment.

A test drives the gopcua client through its public API against the scripted
server in `harness.go`. That server can drop the transport on command —
underneath a held Publish, so a notification is provably lost — and record
both what went over the wire and what the application's callback received. What one run reduces to is the `record`
type, documented at its declaration in `harness.go`.

A helper in a test file reads that `record` and returns plain values; only
the test function itself calls `t.Errorf` or `t.Fatalf`.

The suite exists because a diff of reconnect code does not tell you what the
client will do: that depends on how the server answers after the drop, and
servers differ.

## Tests that fail on purpose

A test for a behaviour gopcua does not have yet carries a `knownBug`
constant, declared inside the test function. Run `go test ./tests/reconnect/ -v`:

- The test runs and its failure prints as a `known bug:` line naming the
  behaviour. The build stays green.
- When your change fixes the behaviour, the test passes, and that pass is
  reported as a hard error telling you to delete the `knownBug` constant.
- Delete it in the same commit as the fix, and say in the commit message
  what the fix was.

Do not delete a `knownBug` constant because the test started passing for a
reason you cannot explain.

## Reviewing a new test: six questions

A test that passes while checking nothing is worse than no test, because it
is trusted. Tooling cannot catch that here: every mutation testing tool in
every language starts by requiring the suite to pass, and a test that fails
on purpose breaks that premise. So a reviewer asks these six, each of which
has caught a real defect in this package.

1. **Can the failing branch ever run?** Trace it. One test here had a FAIL
   condition that could never hold, so it passed every client and failed
   none. Every automated check we had called it sound.

2. **Which facts come from the client, and which from our fake server?** Whether
   the test passes has to turn on a client fact; an assertion about the
   harness's own state tests the fixture. We shipped a test that passed or
   failed on a field only the fake server writes.

3. **Would a correct client reach this code?** Ask it of anything you delete,
   and answer from gopcua's source rather than from the specification. We
   deleted two server arms as dead; a correct client reaches both, and one
   was the test's only route to passing.

4. **Does the test compile against unfixed code?** If it fails only because a
   symbol the fix introduces is missing, it tests a name, not a behaviour: it
   stays red for a wrong fix and goes green for a regression that keeps the
   name. Ten open pull requests ship a test in that state.

5. **Did we construct the condition, or hope for it?** An obligation that
   binds only once something is lost needs that loss constructed. If the
   client has to be unlucky for the test to mean anything, the result is a
   coin flip.

6. **Would a known-good client fail this?** Check the test against Eclipse
   Milo, UA-.NETStandard, node-opcua and open62541, reading their source. If
   one would fail it the test is the likely mistake, and there are two
   different reasons. Either it checks how a client works rather than what it
   achieves, and you rewrite it to assert the outcome. Or every library leaves
   this to the application, and then it is not a test every client should
   pass: keep it only if you say so in the file, name the stacks it fails, and
   show it still fails a client that fakes the outcome instead of earning it.
   We deleted one that a blind rebuild turned green without the client ever
   asking the server what it had lost. The standing exception is
   `republish_after_reconnect_test.go`: open62541 has no client-side Republish
   at all, and we think the test is right. Put what you find in the pull
   request.

## Optional tools

None of these is a gate. While a test is red, our authoring tools can edit a
throwaway copy of the package and check that the test notices: hide the
client's evidence and require the test to fail — a known technique, *checked
coverage* (Schuler & Zeller, 2011) — or change what the evidence says and
require the test to flip between pass and fail. They live outside this
repository because hooks compiled into a test would add a second code path to
the code that has to be trustworthy.

Once a fix turns a test green, ordinary mutation testing works on it: break
the reconnect code and confirm the test fails again. Scope it to the package,
because these tests take seconds each and a whole-module run costs hours.
Plant a deliberate bug first and check the tool reports it before trusting a
clean result.
