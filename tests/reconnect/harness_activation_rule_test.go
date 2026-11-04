// The package states one rule in five places, and this is the executable
// statement of it. A session service that arrives on a connection before
// that connection's ActivateSession is faulted Bad_SessionNotActivated by
// answer() and the server does nothing with it, so no reading of the client
// may count it -- even though the same connection activates a moment later
// and is activated for the rest of the run.
//
// Every one of the five builds its activated map inside the loop that reads
// it, so "activated" there means "had activated by the time this request
// arrived" and not "activated at some point". That ordering is the whole
// content of the rule, and nothing in the package asserted it. A reviewer
// proposing to lift the five into one activatedConns(run) map[int]bool is
// proposing membership over the run, which answers a different question:
// four of the five would start counting the faulted requests, and
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
	"testing"
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
