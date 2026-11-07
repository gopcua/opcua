//go:build conformance

// The package states one rule in seven places, and this is the executable
// statement of it. A session service that arrives on a connection before
// that connection's ActivateSession is faulted Bad_SessionNotActivated by
// answer() and the server does nothing with it, so no reading of the client
// may count it -- even though the same connection activates a moment later
// and is activated for the rest of the run.
//
// Every one of the seven builds its activated map inside the loop that reads
// it, so "activated" there means "had activated by the time this request
// arrived" and not "activated at some point". That ordering is the whole
// content of the rule, and nothing in the package asserted it. A reviewer
// proposing to lift the seven into one activatedConns(run) map[int]bool is
// proposing membership over the run, which answers a different question:
// six of the seven would start counting the faulted requests, and
// firstEarlyUse, whose entire job is to find one, would return false for
// every client that activates at all. The suite would stay green, because no
// client this fixture drives has ever pipelined.
//
// This is a test of the harness and not a row. The six review questions in
// README.md are for rows, and question 2 -- whether the verdict turns on a
// client fact -- is answered here by "no, deliberately": the records are
// written by hand and the subject is the package's own readings of them.
//
// Each case gives the same requests twice, once with the session service
// ahead of its connection's ActivateSession and once behind it, and requires
// the two to read differently. Requiring a difference is what makes this an
// ordering test: a predicate that ignored order, and a predicate that
// counted nothing at all, both fail it.

package reconnect

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gopcua/opcua/ua"
)

// The scenario every case is set in: one connection before the drop, which
// does everything in the right order, and the drop applied to it.
const activationRuleDropConn = 1
const activationRuleHeldSeq = 7

func activationRulePrologue() []requestRecord {
	return []requestRecord{
		{Conn: 1, Service: "CreateSessionRequest"},
		{Conn: 1, Service: "ActivateSessionRequest"},
		{Conn: 1, Service: "CreateSubscriptionRequest"},
		{Conn: 1, Service: "PublishRequest"},
	}
}

// activationRuleRun is the prologue followed by what the client did after the
// drop, which is the part each case varies.
func activationRuleRun(afterDrop ...requestRecord) runRecord {
	return runRecord{
		DropConn: activationRuleDropConn,
		HeldSeq:  activationRuleHeldSeq,
		Requests: append(activationRulePrologue(), afterDrop...),
	}
}

func republishRecord(conn int, seq uint32) requestRecord {
	return requestRecord{Conn: conn, Service: "RepublishRequest", RepublishSeq: seq}
}

// activationRulePolled is a run in which the server created a notification
// above the one it held, so that republishSetupFailed's other reason -- no
// gap for the client to notice -- is out of the way and the case reads only
// the poll.
func activationRulePolled(afterDrop ...requestRecord) runRecord {
	run := activationRuleRun(afterDrop...)
	run.LastSeq = activationRuleHeldSeq + 1
	return run
}

// activationRuleRecreate is a run in which the client replaced the disowned
// subscription on an activated connection 2 and then did afterRecreate.
// runRecord.RecreateAfter is set to the boundary the fixture would have set:
// the number of requests recorded when that CreateSubscription arrived.
func activationRuleRecreate(afterRecreate ...requestRecord) runRecord {
	replaced := []requestRecord{
		{Conn: 2, Service: "CreateSessionRequest"},
		{Conn: 2, Service: "ActivateSessionRequest"},
		republishRecord(2, activationRuleHeldSeq),
		{Conn: 2, Service: "CreateSubscriptionRequest"},
	}
	run := activationRuleRun(append(replaced, afterRecreate...)...)
	run.RecreateAfter = len(run.Requests) - len(afterRecreate)
	return run
}

func TestActivationRuleIsOrdered(t *testing.T) {
	cases := []struct {
		// name is the reading under test.
		name string
		// early sends the session service ahead of its connection's
		// ActivateSession; late sends the same requests behind it.
		early, late runRecord
		// read reduces the reading's result to something comparable, so one
		// table can hold predicates with different signatures.
		read func(runRecord) string
		// want is what each record has to read. They must differ.
		wantEarly, wantLate string
	}{
		{
			name: "republishesAfterDrop does not count a Republish that preceded its ActivateSession",
			early: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
			),
			late: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
			),
			read: func(run runRecord) string {
				asked, wrong, total := republishesAfterDrop(run)
				return fmt.Sprintf("asked=%d wrongSeqs=%v total=%d", asked, wrong, total)
			},
			// total is the same in both on purpose: it is the denominator the
			// rows print and is not filtered, so this pins that too.
			wantEarly: "asked=0 wrongSeqs=[] total=3",
			wantLate:  "asked=1 wrongSeqs=[] total=3",
		},
		{
			name: "republishSeqsAfterDrop leaves out a Republish that preceded its ActivateSession",
			early: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
			),
			late: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
			),
			read: func(run runRecord) string {
				return fmt.Sprintf("%v", republishSeqsAfterDrop(run))
			},
			wantEarly: "[]",
			wantLate:  "[7]",
		},
		{
			name: "republishSetupFailed does not take a Publish that preceded its ActivateSession as a poll",
			early: activationRulePolled(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "PublishRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
			),
			late: activationRulePolled(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				requestRecord{Conn: 2, Service: "PublishRequest"},
			),
			read: republishSetupFailed,
			wantEarly: "the client never polled again after the drop on a connection it had " +
				"activated, so it was never in a position to notice the loss",
			wantLate: "",
		},
		{
			name: "subscriptionAfterInvalidID does not count a CreateSubscription that preceded its ActivateSession",
			early: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				requestRecord{Conn: 3, Service: "CreateSubscriptionRequest"},
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
			),
			late: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
				requestRecord{Conn: 3, Service: "CreateSubscriptionRequest"},
			),
			read: func(run runRecord) string {
				lastAsk, created, sent := subscriptionAfterInvalidID(run)
				return fmt.Sprintf("lastAsk=%d created=%d sent=%d", lastAsk, created, sent)
			},
			// sent is three in both, and holds the faulted CreateSubscription
			// in the early one: it is the denominator the row prints, which
			// is why the row says "none of them a CreateSubscription the
			// server acted on" rather than "none of them a CreateSubscription".
			wantEarly: "lastAsk=7 created=0 sent=3",
			wantLate:  "lastAsk=7 created=1 sent=3",
		},
		{
			name: "subscriptionAfterInvalidID does not take a Republish that preceded its ActivateSession as the last ask",
			early: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				republishRecord(3, 9),
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
			),
			late: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
				republishRecord(3, 9),
			),
			read: func(run runRecord) string {
				lastAsk, created, sent := subscriptionAfterInvalidID(run)
				return fmt.Sprintf("lastAsk=%d created=%d sent=%d", lastAsk, created, sent)
			},
			wantEarly: "lastAsk=7 created=0 sent=3",
			wantLate:  "lastAsk=9 created=0 sent=0",
		},
		{
			name: "afterLastRepublish does not end the loop on a Republish that preceded its ActivateSession",
			early: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 2, Service: "PublishRequest"},
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				republishRecord(3, 9),
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
				requestRecord{Conn: 3, Service: "PublishRequest"},
			),
			late: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				republishRecord(2, activationRuleHeldSeq),
				requestRecord{Conn: 2, Service: "PublishRequest"},
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
				republishRecord(3, 9),
				requestRecord{Conn: 3, Service: "PublishRequest"},
			),
			read: func(run runRecord) string {
				lastAsk, published, sent, asked := afterLastRepublish(run)
				return fmt.Sprintf("lastAsk=%d published=%d sent=%d asked=%v", lastAsk, published, sent, asked)
			},
			wantEarly: "lastAsk=7 published=1 sent=1 asked=true",
			wantLate:  "lastAsk=9 published=1 sent=1 asked=true",
		},
		{
			name: "newSubscriptionAfterDrop does not count a Publish that preceded its ActivateSession",
			early: activationRuleRecreate(
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				// The replacement's CreateMonitoredItems is the fixture's own
				// next step, and newSubscriptionAfterDrop counts no Publish
				// before it; both arms carry it ahead of the Publishes, so the
				// only thing that varies is Publish against ActivateSession.
				requestRecord{Conn: 3, Service: "CreateMonitoredItemsRequest"},
				requestRecord{Conn: 3, Service: "PublishRequest"},
				requestRecord{Conn: 3, Service: "PublishRequest"},
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
			),
			late: activationRuleRecreate(
				requestRecord{Conn: 3, Service: "CreateSessionRequest"},
				requestRecord{Conn: 3, Service: "ActivateSessionRequest"},
				requestRecord{Conn: 3, Service: "CreateMonitoredItemsRequest"},
				requestRecord{Conn: 3, Service: "PublishRequest"},
				requestRecord{Conn: 3, Service: "PublishRequest"},
			),
			read: func(run runRecord) string {
				created, published := newSubscriptionAfterDrop(run)
				return fmt.Sprintf("created=%v published=%d", created, published)
			},
			// created is true in both: the replacement itself is behind its
			// own connection's ActivateSession either way, and what varies is
			// only whether the server could answer the Publish that followed.
			wantEarly: "created=true published=0",
			wantLate:  "created=true published=2",
		},
		{
			name: "firstEarlyUse finds the request that preceded its ActivateSession",
			early: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "PublishRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
			),
			late: activationRuleRun(
				requestRecord{Conn: 2, Service: "CreateSessionRequest"},
				requestRecord{Conn: 2, Service: "ActivateSessionRequest"},
				requestRecord{Conn: 2, Service: "PublishRequest"},
			),
			read: func(run runRecord) string {
				early, found := firstEarlyUse(run)
				return fmt.Sprintf("found=%v service=%q conn=%d", found, early.Service, early.Conn)
			},
			wantEarly: `found=true service="PublishRequest" conn=2`,
			wantLate:  `found=false service="" conn=0`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// A case whose two records read the same cannot tell an ordered
			// reading from an unordered one, and would pass a reading that
			// counted nothing at all.
			if c.wantEarly == c.wantLate {
				t.Fatalf("this case cannot tell the two records apart: both are expected to read %q, "+
					"so a reading that ignored the order of requests would pass it", c.wantEarly)
			}
			if got := c.read(c.early); got != c.wantEarly {
				t.Errorf("with the request ahead of its connection's ActivateSession, where the server "+
					"faults it Bad_SessionNotActivated and does nothing with it, the reading is %s; "+
					"it has to be %s, which is what it reads when the request comes after", got, c.wantEarly)
			}
			if got := c.read(c.late); got != c.wantLate {
				t.Errorf("with the request behind its connection's ActivateSession, where the server "+
					"acts on it, the reading is %s; it has to be %s", got, c.wantLate)
			}
		})
	}
}

// The rule above is sound only while the scripted server grants every
// activation it is asked for, and recordActivationFault is what notices if it
// ever stops. These cases are the five ways an answer stops being a grant,
// and the three answers that are grants and must be left alone.
//
// The last of the three matters more than it looks: answer() returns nil for
// a held Publish, which is how this fixture constructs the loss, on every run
// the package makes. A guard that read nil as a refusal without first asking
// what the request was would decline all eleven rows on every run.
func TestFixtureNoticesAnActivationItDidNotGrant(t *testing.T) {
	const handle = 1
	activate := &ua.ActivateSessionRequest{RequestHeader: &ua.RequestHeader{RequestHandle: handle}}
	publish := &ua.PublishRequest{RequestHeader: &ua.RequestHeader{RequestHandle: handle}}

	// granted is the answer answer() really gives an ActivateSessionRequest.
	// It is asked of answer() rather than written out so that a change to
	// answer()'s grant reaches this case: a refusal there would be recorded
	// as a reason and fail the empty want below. What the case pins is
	// recordActivationFault, which can no longer be changed to flag the
	// answer the fixture gives. An ActivateSessionRequest does not use the
	// Session, so answer()'s activation gate is skipped and nothing here
	// needs a listener or a live connection.
	granted := (&server{rec: &recorder{}}).answer(1, activate)

	for _, c := range []struct {
		name string
		req  ua.Request
		resp ua.Response
		// want is the reason the fixture has to record, or "" if the answer
		// was a grant and nothing may be recorded.
		want string
	}{
		{
			name: "a ServiceFault",
			req:  activate,
			resp: newServiceFault(handle, ua.StatusBadIdentityTokenRejected),
			want: "faulted it with",
		},
		{
			name: "a non-OK ServiceResult",
			req:  activate,
			resp: &ua.ActivateSessionResponse{
				ResponseHeader: newResponseHeader(handle, ua.StatusBadIdentityTokenRejected),
				Results:        []ua.StatusCode{},
			},
			want: "answered it",
		},
		{
			name: "a rejected identity token under an OK ServiceResult",
			req:  activate,
			resp: &ua.ActivateSessionResponse{
				ResponseHeader: newResponseHeader(handle, ua.StatusOK),
				Results:        []ua.StatusCode{ua.StatusBadIdentityTokenRejected},
			},
			want: "accepted it and rejected identity token 0 with",
		},
		{
			name: "no answer at all",
			req:  activate,
			resp: nil,
			want: "never answered it",
		},
		{
			// No arm of answer() returns this for an ActivateSessionRequest.
			// Neither does any of the four above, and pinning them is what
			// this file is for: the catch-all is what keeps an arm added
			// tomorrow from reading as a grant.
			name: "an answer of some other type",
			req:  activate,
			resp: &ua.PublishResponse{ResponseHeader: newResponseHeader(handle, ua.StatusOK)},
			want: "answered it with a *ua.PublishResponse",
		},
		{
			name: "the answer the fixture really gives",
			req:  activate,
			resp: granted,
			want: "",
		},
		{
			name: "a grant whose Results entry is OK",
			req:  activate,
			resp: &ua.ActivateSessionResponse{
				ResponseHeader: newResponseHeader(handle, ua.StatusOK),
				Results:        []ua.StatusCode{ua.StatusOK},
			},
			want: "",
		},
		{
			name: "no answer to a held Publish, which every run of this fixture produces",
			req:  publish,
			resp: nil,
			want: "",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := &recorder{}
			o.recordActivationFault(c.req, c.resp)
			got := o.activationFaulted
			switch {
			case c.want == "" && got != "":
				t.Errorf("the fixture declined a run over an answer that grants the activation: %s", got)
			case c.want != "" && got == "":
				t.Errorf("the fixture let this answer stand as a grant, so every activation filter in "+
					"the package would go on reading an activation the server did not give; it has to "+
					"record a reason naming %q", c.want)
			case c.want != "" && !strings.Contains(got, c.want):
				t.Errorf("the fixture recorded %q, which does not name %q, so the reason a row prints "+
					"would not say which answer the server gave", got, c.want)
			}
		})
	}
}

// The reason is the first one, so a run whose later connections answered
// normally still reports the refusal that happened.
func TestFixtureKeepsTheFirstActivationItDidNotGrant(t *testing.T) {
	const handle = 1
	activate := &ua.ActivateSessionRequest{RequestHeader: &ua.RequestHeader{RequestHandle: handle}}

	o := &recorder{}
	o.recordActivationFault(activate, newServiceFault(handle, ua.StatusBadIdentityTokenRejected))
	first := o.activationFaulted
	o.recordActivationFault(activate, &ua.ActivateSessionResponse{
		ResponseHeader: newResponseHeader(handle, ua.StatusBadInternalError),
		Results:        []ua.StatusCode{},
	})
	o.recordActivationFault(activate, &ua.ActivateSessionResponse{
		ResponseHeader: newResponseHeader(handle, ua.StatusOK),
		Results:        []ua.StatusCode{},
	})

	if first == "" {
		t.Fatalf("the first refusal was not recorded at all")
	}
	if o.activationFaulted != first {
		t.Errorf("the reason changed after the first refusal, to %q; it has to stay %q, or a run "+
			"reports whichever refusal came last instead of the one that invalidated it",
			o.activationFaulted, first)
	}
}
