# Reconnect tests

Each file here is one Go test for one client behaviour in the OPC UA
reconnect space, named after that behaviour. Its opening comment covers
these, in this order:

1. the behaviour and the clause quoted, with the conformance unit if any;
2. the neighbouring rows by filename and what separates them;
3. the scripted server, the default one or the row that owns it, closing
   with `See harness.go.`;
4. what the row asserts and what it deliberately does not;
5. the reference stacks and any exception, where stacks were read.

Item 3 is checkable: `grep -c "See harness.go" tests/reconnect/*_test.go`
prints 1 for every row and 0 for `harness_activation_rule_test.go`.

A test drives the gopcua client through its public API against the scripted
server in `harness.go`. That server can drop the transport on command —
underneath a held Publish, so a notification is provably lost — and record
both what went over the wire and what the application's callback received.
One run reduces to the `runRecord` type, documented at its declaration in
`harness.go`.

Only a test function calls `t.Errorf` or `t.Fatalf`. A helper reads the
`runRecord` and returns plain values. A helper two rows call lives in
`harness.go`, as does one `harness.go` itself calls, whatever the row count;
a helper one row calls lives with that row. `harness_activation_rule_test.go`
is not a placement criterion: it reads helpers wherever they already live,
some in `harness.go` and some in row files.

One file is not a row. `harness_activation_rule_test.go` has no client in it:
it feeds `runRecord` values written by hand to the eight helpers that filter
on activation, and requires each to read a session service that arrived
before its connection's `ActivateSession` differently from the same request
arriving after. The eight state one rule in eight places, the ordering is
the whole content of it, and that file is the only thing checking they
agree. It also checks the assumption all eight rest on, that the scripted
server grants every activation it is asked for, by putting the five answers
that are not a grant to the fixture's own check and requiring it to notice
each one. The six questions below are for rows and do not apply to it.

That count is checkable, the way item 3 above is. `grep -ro 'activated :=
map\[int\]bool{}' tests/reconnect/ | wc -l` prints one line per helper that
filters on activation, and `grep -o 'name: "[^" ]* [^"]*ActivateSession'
tests/reconnect/harness_activation_rule_test.go | cut -d'"' -f2 | cut -d' '
-f1 | sort -u | wc -l` prints how many of them `TestActivationRuleIsOrdered`
has a case for. Both print 8; the second is lower than the case count
because one helper has two cases.

The suite exists because a diff of reconnect code does not tell you what the
client will do: that depends on how the server answers after the drop, and
servers differ.

What the package deliberately does not test, and the fixture capability each
gap waits on, is in `NOT-TESTED.md`.

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
as we merge it. Where a fix reaches main ahead of this suite, the deletion
lands instead in the change that rebases the suite onto it. Once the
suite is merged, the contributor whose fix turns a row red deletes the
constant in their own pull request, and that is how the entry is closed.

Do not delete a `knownBug` constant because the test started passing for a
reason you cannot explain.

A row with no `knownBug` constant says in a clause that gopcua passes it, the
way a row says which reference stacks were read, so that silence there means
nobody checked rather than nothing to report.

A constant costs the row a second thing while it stands: the row cannot tell
*this client is broken* from *this client cannot be measured at all*. Where
the missing behaviour is an absence and the arm keys on that absence, a
client that never reaches the stimulus lands in the same green arm as one
that reached it and did the wrong thing. Key the arm on a presence where the
behaviour allows one, and where it does not, say so in the file.

### Constants closed by pull request 907, 2026-11-05

https://github.com/gopcua/opcua/pull/907 merged into main and fixed seven of
the eight behaviours that carried a constant. Each row below was logging its
`knownBug` line; each now passes, and its constant and the two arms that read
it went out in the change that rebased this suite onto that main.

| row | behaviour that was missing |
|---|---|
| `data_resumes_after_reconnect_test.go` | the client reconnected and re-activated the session but never received another notification |
| `data_resumes_on_recreated_subscription_test.go` | the client left the subscription as it was, so it never created the replacement or monitored anything on it |
| `missed_notification_recovered_when_transfer_refused_test.go` | the server still had the notification the client missed, and the client never got it back |
| `new_subscription_after_republish_returns_invalid_id_test.go` | the client never asked for what it missed, so it was never told the subscription id was invalid and never replaced the subscription |
| `publish_resumes_when_republish_loop_ends_test.go` | the client never asked for what it missed, so it had no loop of its own to end and nothing to resume from |
| `republish_after_reconnect_test.go` | the client never asked for the notification it missed |
| `republish_asks_for_each_missed_number_test.go` | the client never asked for what it missed, so there was no loop to walk |

One constant survives: `no_acknowledgements_when_nothing_is_available_test.go`,
whose behaviour 907 did not fix.

## Rows that score a pull request

A row can score a change that exists only as a pull request. It carries no
`knownBug` constant, because the behaviour is on main and it is the pull
request that breaks it, so it is listed here and not in the table above.

| row | what it scores |
|---|---|
| `acknowledgements_resume_after_one_is_rejected_test.go` | https://github.com/gopcua/opcua/pull/899, which latches on one Bad_SequenceNumberUnknown and never acknowledges again |

## Reviewing a new test: six questions

A test that passes while checking nothing is worse than no test, because it
is trusted. Tooling cannot see all of it here: the behaviour a `knownBug`
constant masks is logged rather than failed, so a mutation score counts that
row as passing whatever the client does about that behaviour. The arms that
fire beside the constant are ordinary failures and are visible.

A decline is a `t.Fatalf`, so the test is red like any other failure. The
word marks why it is red — the run could not be scored — and not a third
outcome between passing and failing.

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
   An exception is declared in the header of the file that takes it, not
   here; a row checked with no failing stack says so in a clause, so that
   silence means nobody checked. Put what you find in the pull request.

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
