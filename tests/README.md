# Test suites

Each file in a package here is one Go test for one behaviour in the OPC UA
space, named after that behaviour, holding one `func Test`. Each such
file is called a row. Its opening comment covers these, in this order:

1. the behaviour and the clause quoted, with the conformance unit if any;
2. the neighbouring rows by filename and what separates them;
3. what the row runs against - for a row driven by a scripted server, that
   server, the default one or the row that owns it; for a row that drives
   the library directly, the shared helpers - closing with
   `See harness.go.`;
4. what the row asserts and what it deliberately does not;
5. the reference stacks and any exception, where stacks were read.

Item 3 is checkable: `grep -c "See harness.go" tests/<package>/*_test.go`
prints 1 for every row, and a file that is not a row prints 0 and says
why in its package's own README. The item is deliberately wider than the
scripted server it started as, and it still forbids something concrete at
that width: a row that names no fixture at all, and a row that runs
against something other than the package default and does not say which.

Only a test function calls `t.Errorf` or `t.Fatalf`; a helper returns plain
values. A helper two rows call lives in `harness.go`, as does one
`harness.go` itself calls, whatever the row count; a helper one row calls
lives with that row.

A decline is a `t.Fatalf`, so the test is red like any other failure. The
word marks why it is red - the run could not be scored - and not a third
outcome between passing and failing.

## Tests for behaviour that does not exist yet

A test for a behaviour gopcua does not have yet carries a `knownBug`
constant, declared inside the test function. Run
`go test ./tests/<package>/ -v`:

- The test runs and its failure prints as a `known bug:` line naming the
  behaviour. The build stays green.
- When your change fixes the behaviour, the test passes, and that pass is
  reported as a hard error telling you to delete the `knownBug` constant.
  The branches of the verdict switch are called arms.
- Delete the constant, **every arm that reads it, and anything only those
  arms used**, or the file stops compiling. Do it in the same commit as
  the fix, and say in the commit message what the fix was.

Who that is depends on where these suites are. While they are unmerged,
the person who deletes a constant is whoever runs the suite by hand
against a candidate fix, and the deletion lands with that fix as we
merge it. Where a fix reaches main ahead of a
suite, the deletion lands instead in the change that rebases the suite
onto it. Once the suites are merged, the contributor whose fix turns a
row red deletes the constant in their own pull request, and that is how
the entry is closed.

Do not delete a `knownBug` constant because the test started passing for a
reason you cannot explain.

A row with no `knownBug` constant says in a clause that gopcua passes it,
the way a row says which reference stacks were read, so that silence there
means nobody checked rather than nothing to report.

A constant costs the row a second thing while it stands: the row cannot
tell *this library is broken* from *this library cannot be measured at
all*. Where the missing behaviour is an absence and the arm keys on that
absence, an implementation that never reaches the stimulus lands in the
same green arm as one that reached it and did the wrong thing. Key the arm
on a presence where the behaviour allows one, and where it does not, say
so in the file.

## Reviewing a new test: six questions

A test that passes while checking nothing is worse than no test, because
it is trusted. Tooling cannot see all of it here: the behaviour a
`knownBug` constant masks is logged rather than failed, so a mutation
score counts that row as passing whatever the library does about that
behaviour. The arms that fire beside the constant are ordinary failures
and are visible.

So a reviewer asks these six, each of which has caught a real defect in
one of these packages.

1. **Can the failing or declining branch ever run?** Trace it. One test
   here had a FAIL condition that could never hold, so it passed every
   client and failed none. Every automated check we had called it sound.
2. **Which facts come from the code under test, and which from the
   fixture?** Whether the test passes has to turn on a fact about the
   code under test; a row whose verdict turns on the fixture's own state
   tests the fixture, and scores nothing the library does.
3. **Would a correct implementation reach this code?** Ask it of
   anything you delete, and answer from gopcua's source rather than from
   the specification. Two server arms were deleted as dead; a correct
   client reaches both, and one was the test's only route to passing.
4. **Does the test compile against unfixed code?** If it fails only
   because a symbol the fix introduces is missing, it tests a name, not a
   behaviour: it stays red for a wrong fix and goes green for a regression
   that keeps the name.
5. **Did we construct the condition, or hope for it?** Construct the
   loss, and construct the chance to notice it. The setup check
   `republish_after_reconnect_test.go` owns passed a run where the client
   reconnected and then sent no Publish at all: nothing arrived after the
   gap, so it was never asked what it is scored on.
6. **Would a known-good implementation fail this?** Read one other
   implementation's source and check the test against it - another
   client's source for a row that drives the client, another stack's codec
   for a row that drives an encoder or decoder. Reading all of
   UA-.NETStandard, node-opcua, open62541 and Eclipse Milo is a
   maintainer's job, not a condition of the pull request. If a known-good
   implementation would fail the test, the test is the likely mistake,
   and there are two different reasons. Either it checks how the
   implementation works rather than what it achieves, and you rewrite it
   to assert the outcome. Or every library leaves this to the
   application, and then it is not a test every implementation should
   pass: keep it only if you say so in the file, name the stacks it
   fails, and show it still fails an implementation that fakes the
   outcome instead of earning it. An exception is declared in the header
   of the file that takes it, not here; a row checked with no failing
   stack says so in a clause, so that silence means nobody checked. Put
   what you find in the pull request.

## Optional tools

None of these is a gate. While the behaviour is missing, our authoring
tools can edit a throwaway copy of the package and check that the test
notices: hide the library's evidence and require the test to fail -
*checked coverage* - or change what the evidence says and require the
test to flip between pass and fail. They live outside this repository
because hooks compiled into a test would add a second code path to the
code that has to be trustworthy.

Once a fix turns a test green, ordinary mutation testing works on it:
break the code under test and confirm the test fails again. Scope it to
the package, because these tests take seconds each and a whole-module run
costs hours. Plant a deliberate bug first and check the tool reports it
before trusting a clean result.
