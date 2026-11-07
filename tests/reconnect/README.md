# Reconnect tests

The shared contract for these suites is in `../README.md`; this file holds
only what is specific to reconnect.

A row here drives the gopcua client through its public API against the
scripted server in `harness.go`. That server can drop the transport on
command - underneath a held Publish, so a notification is provably lost -
and record both what went over the wire and what the application's
callback received. One run reduces to the `runRecord` type, documented at
its declaration in `harness.go`.

Item 3's reading here is the scripted server, the default one or the row
that owns it. It is checkable: `grep -c "See harness.go"
tests/reconnect/*_test.go` prints 1 for every row and 0 for
`harness_activation_rule_test.go`.

One file is not a row. `harness_activation_rule_test.go` has no client in
it: it feeds `runRecord` values written by hand to the eight helpers that
filter on activation, and requires each to read a session service that
arrived before its connection's `ActivateSession` differently from the
same request arriving after. The eight state one rule in eight places,
the ordering is the whole content of it, and that file is the only thing
checking they agree. It also checks the assumption all eight rest on,
that the scripted server grants every activation it is asked for, by
putting the five answers that are not a grant to the fixture's own check
and requiring it to notice each one. It is not a placement criterion:
it reads helpers wherever they already live, some in `harness.go` and
some in row files. The six questions in `../README.md` are for rows and
do not apply to it.

That count is checkable, the way item 3 is. `grep -ro 'activated :=
map\[int\]bool{}' tests/reconnect/ | wc -l` prints one line per helper
that filters on activation, and `grep -o 'name: "[^" ]* [^"]*ActivateSession'
tests/reconnect/harness_activation_rule_test.go | cut -d'"' -f2 | cut
-d' ' -f1 | sort -u | wc -l` prints how many of them
`TestActivationRuleIsOrdered` has a case for. Both print 8; the second is
lower than the case count because one helper has two cases.

One convention the six questions assume here: a row with two kinds of
decline splits them, the one that is gopcua's own defect through the
`knownBug` arm and the one a conforming client earns as a `t.Fatalf` up
with the preconditions. The rule and its reasoning are on `setupFailed`
in `harness.go`.

Q6's reading here is another client's source: open62541 and Eclipse Milo
are the two stacks that have caught something in this package.

The suite exists because a diff of reconnect code does not tell you what
the client will do: that depends on how the server answers after the
drop, and servers differ.

What the package deliberately does not test, and the fixture capability
each gap waits on, is in `NOT-TESTED.md`.

## Constants closed by pull request 907, 2026-11-05

https://github.com/gopcua/opcua/pull/907 merged into main and fixed seven
of the eight behaviours that carried a constant. Each row below was
logging its `knownBug` line; each now passes, and its constant and every
arm that read it went out in the change that rebased this suite onto
that main.

| row | behaviour that was missing |
|---|---|
| `data_resumes_after_reconnect_test.go` | the client reconnected and re-activated the session but never received another notification |
| `data_resumes_on_recreated_subscription_test.go` | the client left the subscription as it was, so it never created the replacement or monitored anything on it |
| `missed_notification_recovered_when_transfer_refused_test.go` | the server still had the notification the client missed, and the client never got it back |
| `new_subscription_after_republish_returns_invalid_id_test.go` | the client never asked for what it missed, so it was never told the subscription id was invalid and never replaced the subscription |
| `publish_resumes_when_republish_loop_ends_test.go` | the client never asked for what it missed, so it had no loop of its own to end and nothing to resume from |
| `republish_after_reconnect_test.go` | the client never asked for the notification it missed |
| `republish_asks_for_each_missed_number_test.go` | the client never asked for what it missed, so there was no loop to walk |

One constant survives:
`no_acknowledgements_when_nothing_is_available_test.go`, whose behaviour
907 did not fix.

## Rows that score a pull request

A row can score a change that exists only as a pull request. It carries no
`knownBug` constant, because the behaviour is on main and it is the pull
request that breaks it, so it is listed here and not in the table above.

| row | what it scores |
|---|---|
| `acknowledgements_resume_after_one_is_rejected_test.go` | https://github.com/gopcua/opcua/pull/899, which latches on one Bad_SequenceNumberUnknown and never acknowledges again |
