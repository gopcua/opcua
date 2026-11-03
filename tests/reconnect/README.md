# Reconnect tests

Each file here is one Go test for one client behaviour in the OPC UA
reconnect space, named after that behaviour. The specification clause it
checks is in the file's opening comment.

A test drives the gopcua client through its public API against the scripted
server in `harness.go`. That server can drop the transport on command —
underneath a held Publish, so a notification is provably lost — and record
both what went over the wire and what the application's callback received.
One run reduces to the `runRecord` type, documented at its declaration in
`harness.go`.

Only a test function calls `t.Errorf` or `t.Fatalf`. A helper reads the
`runRecord` and returns plain values. A helper two rows call lives in
`harness.go`; a helper one row calls lives with that row.

The suite exists because a diff of reconnect code does not tell you what the
client will do: that depends on how the server answers after the drop, and
servers differ.

## Tests for behaviour that does not exist yet

A test for a behaviour gopcua does not have yet carries a `knownBug`
constant, declared inside the test function. Run `go test ./tests/reconnect/ -v`:

- The test runs and its failure prints as a `known bug:` line naming the
  behaviour. The build stays green.
- When your change fixes the behaviour, the test passes, and that pass is
  reported as a hard error telling you to delete the `knownBug` constant.
- Delete the constant **and the two switch arms that read it**, or the file
  stops compiling. Do it in the same commit as the fix, and say in the commit
  message what the fix was.

Who that is depends on where this suite is. While it is unmerged — which is
where it is today — the person who deletes a constant is whoever runs the
suite by hand against a candidate fix, and the deletion lands with that fix
as we merge it. Once the suite is merged, the contributor whose fix turns a
row red deletes the constant in their own pull request, and that is how the
entry is closed.

Do not delete a `knownBug` constant because the test started passing for a
reason you cannot explain.

A constant costs the row a second thing while it stands: the row cannot tell
*this client is broken* from *this client cannot be measured at all*. Where
the missing behaviour is an absence and the arm keys on that absence, a
client that never reaches the stimulus lands in the same green arm as one
that reached it and did the wrong thing. Key the arm on a presence where the
behaviour allows one, and where it does not, say so in the file.

## Reviewing a new test: six questions

A test that passes while checking nothing is worse than no test, because it
is trusted. Tooling cannot see all of it here: the behaviour a `knownBug`
constant masks is logged rather than failed, so a mutation score counts that
row as passing whatever the client does about that behaviour. The arms that
fire beside the constant are ordinary failures and are visible.

One convention the questions assume: a row with two kinds of decline splits
them, the one that is gopcua's own defect through the `knownBug` arm and the
one a conforming client earns as a `t.Fatalf` up with the preconditions. The
rule and its reasoning are on `setupFailed` in `harness.go`.

So a reviewer asks these six, each of which has caught a real defect in this
package.

1. **Can the failing or declining branch ever run?** Trace it. One test here had a FAIL
   condition that could never hold, so it passed every client and failed
   none. Every automated check we had called it sound.

2. **Which facts come from the client, and which from our fake server?** Whether
   the test passes has to turn on a client fact; an assertion about the
   harness's own state tests the fixture.

3. **Would a correct client reach this code?** Ask it of anything you delete,
   and answer from gopcua's source rather than from the specification. We
   deleted two server arms as dead; a correct client reaches both, and one
   was the test's only route to passing.

4. **Does the test compile against unfixed code?** If it fails only because a
   symbol the fix introduces is missing, it tests a name, not a behaviour: it
   stays red for a wrong fix and goes green for a regression that keeps the
   name.

5. **Did we construct the condition, or hope for it?** Construct the loss, and
   construct the chance to notice it. The Republish row's own setup check
   passed a run where the client reconnected and then sent no Publish at all:
   nothing arrived after the gap, so it was never asked what it is scored on.

6. **Would a known-good client fail this?** Read one other stack's source and
   check the test against it; open62541 and Eclipse Milo are the two that have
   caught something here. Reading all of UA-.NETStandard, node-opcua and the
   other two is a maintainer's job, not a condition of the pull request. If a
   known-good client would fail the test the test is the likely mistake, and there are two
   different reasons. Either it checks how a client works rather than what it
   achieves, and you rewrite it to assert the outcome. Or every library leaves
   this to the application, and then it is not a test every client should
   pass: keep it only if you say so in the file, name the stacks it fails, and
   show it still fails a client that fakes the outcome instead of earning it.
   An exception is declared in the header of the file that takes it, not here.
   Put what you find in the pull request.

## Optional tools

None of these is a gate. While the behaviour is missing, our authoring tools
can edit a throwaway copy of the package and check that the test notices:
hide the client's evidence and require the test to fail — *checked coverage*
— or change what the evidence says and require the test to flip between pass
and fail. They live outside this
repository because hooks compiled into a test would add a second code path to
the code that has to be trustworthy.

Once a fix turns a test green, ordinary mutation testing works on it: break
the reconnect code and confirm the test fails again. Scope it to the package,
because these tests take seconds each and a whole-module run costs hours.
Plant a deliberate bug first and check the tool reports it before trusting a
clean result.
