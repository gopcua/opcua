// Package reconnect holds the gopcua reconnect tests: one behaviour per file,
// with the clause it checks in the file's opening comment. This file is the
// fixture they all run against.
//
// The fixture is a scripted OPC UA server. It drops the transport on command,
// can refuse every connection after the drop, can refuse to transfer the
// subscription, and records what it decoded off the wire and what the
// client's callback received.
//
// The lost notification is constructed. The server answers the stream's first
// Publish and holds every later one: the notification gets a sequence number
// but is never sent. Once a Publish is held, the scenario resets the connection
// it is outstanding on, so the held notification cannot have reached the
// client. Nothing tells the session its transport is gone.
//
// Exactly one notification is lost in the stream, and the scenario checks it:
// a client that keeps one Publish outstanding leaves one sequence number
// created and withheld, which is the one availableSeqsLocked offers back and
// the one the rows name. A client that keeps more outstanding would lose more
// than the rows describe, and the run is reported as a setup failure instead.
//
// A run can ask for more than that one to be recoverable. runOptions.FurtherLost
// notifications are created as the drop is applied and go straight into the
// server's retransmission queue: responses that were sent and never received
// ("When the connection is lost, Publish responses may have been sent but not
// received by the Client", OPC 10000-4 §6.7). They are not part of the
// stream and do not touch the one-lost check, which is frozen first. They
// exist so that a Republish loop has a range to walk rather than a single
// number; the default is none.
//
// The sequence counter is global and never resets, so a subscription the
// client recreates after the drop does not start at 1: it looks like a gap to
// a client that detects loss by sequence gaps.
package reconnect

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

// publishPacing is how long the scripted server waits before answering a
// Publish. Without it the client would re-publish as fast as it is answered
// and flood the request log.
const publishPacing = 200 * time.Millisecond

// settleDefault is how long a run goes on after the drop when the test sets no
// Settle.
const settleDefault = 5 * time.Second

// SubID is the subscription id the scripted server hands out; the scenario
// has one subscription.
const SubID uint32 = 42

// monitoredNode is the node the scenario monitors. The scripted server never
// reads it; it manufactures notifications instead.
var monitoredNode = ua.NewStringNodeID(1, "reconnect.value")

// requestRecord is one service request the scripted server decoded off the
// wire. Conn is the connection generation it arrived on, counting from 1.
// runRecord.DropConn says which generation was dropped. RepublishSeq is the
// sequence number a RepublishRequest asked for and is meaningless on any
// other service.
//
// Acks is what a PublishRequest acknowledged, in the order the client wrote
// them, and is meaningless on any other service. It is empty both for a
// Publish that acknowledged nothing and for every request that is not a
// Publish, so a reader that needs to tell those apart checks Service first.
type requestRecord struct {
	Conn         int
	Service      string
	RepublishSeq uint32
	Acks         []ack
}

// ack is one SubscriptionAcknowledgement the client put in a Publish.
type ack struct {
	SubID, Seq uint32
}

// clone copies the acknowledgement slice, so a test holding a snapshot cannot
// reach the recorder's live state.
func (r requestRecord) clone() requestRecord {
	r.Acks = append([]ack(nil), r.Acks...)
	return r
}

// deliveryRecord is one notification the client's application callback
// received, carrying the sequence number the server encoded into the value.
type deliveryRecord struct {
	Seq uint32
	At  time.Time
}

// recorder collects what both ends saw while the run is live. Safe for
// concurrent use: the server serves each connection on its own goroutine and
// the drop watcher runs beside them.
type recorder struct {
	mu       sync.Mutex
	requests []requestRecord
	accepts  []time.Time // when each connection was accepted, refused ones included

	// dropStarted is set just before the connection is closed, so a client
	// that redials the instant it sees the reset is already refused.
	dropStarted bool
	// dropApplied is set once the close has succeeded; the scenario waits for
	// it. dropAt is that instant: a connection accepted before it is not a
	// reconnection attempt.
	dropApplied bool
	dropAt      time.Time

	lastSeq       uint32   // highest notification sequence created so far
	seqAtDrop     uint32   // lastSeq when the drop was applied
	heldSeq       uint32   // sequence of the first held Publish; 0 = none
	queued        []uint32 // further sequence numbers kept for retransmission; see the package comment
	heldConn      int      // connection the held Publish is outstanding on; 0 = none
	firstAnswered bool     // the stream's first Publish has been answered, ever

	// activatedConns holds the connections ActivateSession has arrived on. It
	// is per connection because a reconnect must activate the session again.
	activatedConns map[int]bool
	// activationFaulted is the fixture checking itself, and is empty while
	// the check holds. Every activation filter in this package reads "an
	// ActivateSessionRequest was recorded" as "the Session is activated on
	// that connection", which is true only because answer() returns success
	// for every one of them. A server scripted to refuse re-activation --
	// which reaching §5.14.7.1 would need, as
	// no_acknowledgements_when_nothing_is_available_test.go notes -- would
	// turn all of those filters into no-ops at once, and every row would go
	// on scoring clients against a Session the server had taken away. This
	// holds the reason the rows decline instead.
	activationFaulted string
	// anyMonitored is state of the whole fixture, unlike activatedConns: the
	// scenario has one subscription. A client that recreates the subscription
	// and then fails to recreate its monitored items gets keep-alives for the
	// rest of the run, and the row that notices reports missing notifications
	// rather than missing items.
	anyMonitored bool
	// seqAtRecreate is lastSeq at the moment the first CreateSubscription
	// after the drop arrived; zero while none has. Everything above it was
	// created for the replacement subscription, so it is the line between data
	// the client was still being given on the old one and data it earned by
	// recreating.
	seqAtRecreate uint32
	// recreateAfter is len(requests) at that same moment, so the requests
	// from it on are everything the client sent after that CreateSubscription
	// went over the wire. Zero while no such request has arrived, and that
	// sentinel cannot be mistaken for a real value: the request is recorded
	// before the arm that sets this runs, so the count is at least one
	// whenever the arm has run. The boundary is all it is: serve() records a
	// request before answer() reaches that arm, and another connection's
	// goroutine can append in between, so requests[recreateAfter-1] is not
	// reliably the CreateSubscription itself.
	recreateAfter int

	// deliveries is what the application's own notification callback
	// received. It belongs here and not beside the callback because the
	// record a test reads has to hold it and the request log as one value,
	// and one lock is the only way two slices written by different goroutines
	// can be read at one instant.
	deliveries []deliveryRecord
}

// recordDelivery is the application callback's one mutation.
func (o *recorder) recordDelivery(seq uint32) {
	o.mu.Lock()
	o.deliveries = append(o.deliveries, deliveryRecord{Seq: seq, At: time.Now()})
	o.mu.Unlock()
}

func (o *recorder) deliveryCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.deliveries)
}

// recordRequest takes the decoded request, so the call site does not need to
// know which services carry something worth keeping.
func (o *recorder) recordRequest(connID int, req ua.Request) {
	rr := requestRecord{Conn: connID, Service: serviceName(req)}
	switch r := req.(type) {
	case *ua.PublishRequest:
		for _, a := range r.SubscriptionAcknowledgements {
			rr.Acks = append(rr.Acks, ack{SubID: a.SubscriptionID, Seq: a.SequenceNumber})
		}
	case *ua.RepublishRequest:
		rr.RepublishSeq = r.RetransmitSequenceNumber
	}
	o.mu.Lock()
	if rr.Service == "ActivateSessionRequest" {
		if o.activatedConns == nil {
			o.activatedConns = map[int]bool{}
		}
		o.activatedConns[connID] = true
	}
	o.requests = append(o.requests, rr)
	o.mu.Unlock()
}

// recordActivationFault keeps the first answer the scripted server gave an
// ActivateSessionRequest that was not a success, and ignores every other
// request. serve() calls it at the one point that holds both the request and
// the response it produced, so the usesSession gate at the top of answer(),
// the catch-all at the bottom and any arm added between them are covered
// without a call site of their own.
//
// It records rather than failing the test: this runs on the connection's own
// goroutine, and reaching for *testing.T from there races the test goroutine.
// setupFailed is what turns the record into a declined row.
//
// Five things stop an answer being a success. A ServiceFault and a non-OK
// ServiceResult are both visible to anyone grepping for newServiceFault. A
// Results entry that is not OK under an OK ServiceResult is the server
// accepting the request and rejecting the identity token in it, which
// refuses the activation without producing a fault anywhere, so a guard that
// looked only for faults would let that one through. No answer at all
// leaves the request recorded and the Session unactivated, which is the
// failure this whole check is about, reached by the one route that writes no
// status down anywhere. And an answer of any other type is the catch-all:
// nothing in answer() produces one today, and an arm that did would read as
// a grant without it.
//
// The nil it sees is the nil serve() sees, and it is only ever read for an
// ActivateSessionRequest: answer() returns nil for a held Publish on every
// run this package makes, which is the fixture's own mechanism and not a
// fault.
func (o *recorder) recordActivationFault(req ua.Request, resp ua.Response) {
	if _, isActivation := req.(*ua.ActivateSessionRequest); !isActivation {
		return
	}
	var why string
	switch act := resp.(type) {
	case nil:
		why = "never answered it"
	case *ua.ServiceFault:
		why = fmt.Sprintf("faulted it with %s", responseStatus(act))
	case *ua.ActivateSessionResponse:
		switch {
		case responseStatus(act) != ua.StatusOK:
			why = fmt.Sprintf("answered it %s", responseStatus(act))
		default:
			for i, st := range act.Results {
				if st != ua.StatusOK {
					why = fmt.Sprintf("accepted it and rejected identity token %d with %s", i, st)
					break
				}
			}
		}
	default:
		why = fmt.Sprintf("answered it with a %T", act)
	}
	if why == "" {
		return
	}
	o.mu.Lock()
	if o.activationFaulted == "" {
		o.activationFaulted = "the scripted server " + why + ", so the Session was not " +
			"activated on that connection. Every check in this package that filters on " +
			"activation reads the request and not the answer, and all of them are now " +
			"reading an activation that did not happen, so nothing this run recorded can " +
			"be scored"
	}
	o.mu.Unlock()
}

func (o *recorder) recordConn() {
	o.mu.Lock()
	o.accepts = append(o.accepts, time.Now())
	o.mu.Unlock()
}

func (o *recorder) isDropStarted() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.dropStarted
}

func (o *recorder) recordDropStarted() {
	o.mu.Lock()
	o.dropStarted = true
	o.mu.Unlock()
}

// recordDropApplied freezes the sequence counter alongside the flag: the
// dropped connection serves nothing more, so this is how many notifications
// the server had created when the loss was made. The counter is frozen before
// the further lost notifications are created, so they are not counted as
// notifications the stream lost.
func (o *recorder) recordDropApplied(furtherLost int) {
	o.mu.Lock()
	o.dropApplied = true
	o.dropAt = time.Now()
	o.seqAtDrop = o.lastSeq
	for i := 0; i < furtherLost; i++ {
		o.lastSeq++
		o.queued = append(o.queued, o.lastSeq)
	}
	o.mu.Unlock()
}

func (o *recorder) dropAppliedAt() time.Time {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.dropAt
}

func (o *recorder) isDropApplied() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.dropApplied
}

func (o *recorder) seqCreatedAtDrop() uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.seqAtDrop
}

func (o *recorder) isActivated(connID int) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.activatedConns[connID]
}

// recordItemsMonitored is the CreateMonitoredItems arm's one mutation, and
// the only thing that turns the fixture's keep-alives back into data.
func (o *recorder) recordItemsMonitored() {
	o.mu.Lock()
	o.anyMonitored = true
	o.mu.Unlock()
}

// recordNewSubscription is the CreateSubscription arm's one mutation: a fresh
// subscription has nothing monitored on it yet, and the first one created
// after the drop fixes both the sequence number everything the replacement
// carries is above and the point in the request log the replacement starts
// at. All three are set together under the one lock. A Publish answered
// between the first two would be counted on the wrong side of that line, and
// a reader that could find the third without the second -- or the second
// without the third -- would have two answers to "did the client replace the
// subscription" and no reason for them to agree.
func (o *recorder) recordNewSubscription() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.anyMonitored = false
	// recreateAfter is the flag rather than seqAtRecreate, because
	// seqAtRecreate is a sequence number and nothing here forbids it being
	// zero, while a recorded request count cannot be.
	if o.dropApplied && o.recreateAfter == 0 {
		o.seqAtRecreate = o.lastSeq
		o.recreateAfter = len(o.requests)
	}
}

func (o *recorder) isAnyMonitored() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.anyMonitored
}

// peekNextSeq is the sequence number the next notification will carry.
// Reading it does not use it up, so a keep-alive can carry it.
func (o *recorder) peekNextSeq() uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lastSeq + 1
}

func (o *recorder) heldPublishSeq() uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.heldSeq
}

// availableSeqsLocked is what the server says it can still resend, read with
// the recorder's lock already held. An empty list claims it keeps no
// retransmission queue at all (OPC 10000-4 §5.14.1.1), so while a
// notification is held back this list names it -- except under
// NoRetransmissionQueue, where availableLocked throws the list away and makes
// that claim on purpose. Every Publish answer and every TransferResult
// carries it, and the two agree except under AdvertisesSentSequence, where a
// data PublishResponse carries its own sequence number as well; see
// availableWith. Only under RefusesTransfer does a Republish for that number
// actually get it back; the default server advertises it and then answers
// Bad_MessageNotAvailable, which is how republish_after_reconnect_test.go
// observes a client asking without being handed the answer.
// Everything from the first withheld notification to the last one queued as
// the drop was applied is in the list; anything the server creates after
// that is outside it. The list is contiguous because heldSeq and the
// FurtherLost numbers recordDropApplied queues behind it are consecutive, so
// a Republish loop walking it never meets a hole -- which would look to a
// test exactly like a client that stopped incrementing too early. A client
// with several Publishes outstanding does not put one there either:
// runScenario's one-lost check declines such a run before a row scores it,
// and the withheld numbers past the first are in neither heldSeq nor queued,
// so the list would come up short rather than holey.
func (o *recorder) availableSeqsLocked() []uint32 {
	if o.heldSeq == 0 {
		return []uint32{}
	}
	last := o.heldSeq
	for _, q := range o.queued {
		if q > last {
			last = q
		}
	}
	seqs := make([]uint32, 0, last-o.heldSeq+1)
	for seq := o.heldSeq; seq <= last; seq++ {
		seqs = append(seqs, seq)
	}
	return seqs
}

func (o *recorder) heldPublishConn() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.heldConn
}

// publishStep answers the stream's first Publish and holds every later one.
// After the drop it answers again with fresh data, so a client that notices a
// gap only when the next notification arrives has one to notice it in.
// Every call uses up a sequence number, held or not.
func (o *recorder) publishStep(connID int) (seq uint32, hold bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.lastSeq++
	if o.dropApplied {
		return o.lastSeq, false
	}
	if o.firstAnswered {
		if o.heldSeq == 0 {
			o.heldSeq = o.lastSeq
		}
		if o.heldConn == 0 {
			o.heldConn = connID
		}
		return o.lastSeq, true
	}
	o.firstAnswered = true
	return o.lastSeq, false
}

// snapshot copies into out everything the run recorded, in one acquisition of
// the recorder's lock. The run is still live when this is called, so a second
// acquisition would let a request land between two reads and leave a record
// that describes no instant the run ever passed through: a CreateSubscription
// arriving after the deliveries were copied and before the pair
// recordNewSubscription sets was read puts RecreateAfter's line above every
// delivery in the record, and the row that reads the two together reports a
// client that was given nothing when it had been given everything.
func (s *server) snapshot(out *runRecord) {
	o := s.rec
	o.mu.Lock()
	defer o.mu.Unlock()
	requests := make([]requestRecord, len(o.requests))
	for i, r := range o.requests {
		requests[i] = r.clone()
	}
	out.Requests = requests
	out.Deliveries = append([]deliveryRecord(nil), o.deliveries...)
	out.AcceptedAt = append([]time.Time(nil), o.accepts...)
	out.HeldSeq = o.heldSeq
	out.LastSeq = o.lastSeq
	out.SeqAtRecreate = o.seqAtRecreate
	out.RecreateAfter = o.recreateAfter
	out.ActivationFaulted = o.activationFaulted
	out.Answerable = s.answerableSeqsLocked()
	out.Offered = s.availableLocked()
}

// runRecord is what one run reduces to: what both ends saw, frozen when the
// run ends. A test decides pass or fail from this value alone.
type runRecord struct {
	// Requests is every service request the server decoded, each with the
	// connection generation it arrived on.
	Requests []requestRecord
	// Deliveries is what the application's own callback received. It holds at
	// least one notification whenever HarnessErr is empty, because runScenario
	// waits for the callback before it commands the drop and every path out of
	// it that stops short of that sets HarnessErr.
	Deliveries []deliveryRecord
	// HeldSeq is the sequence number of the notification the server held back.
	// It is non-zero whenever HarnessErr is empty, because runScenario waits
	// for the server to be holding a Publish before it commands the drop and
	// every path out of it that stops short of that sets HarnessErr. The
	// request it was withheld from is never the stream's first Publish:
	// publishStep sets it only on the branch guarded by firstAnswered, which
	// an earlier call has to have taken.
	HeldSeq uint32
	// LastSeq is the highest sequence number the server created.
	LastSeq uint32
	// DropAt is when the connection was closed, not when the drop was
	// commanded.
	DropAt time.Time
	// DropConn is the connection generation the drop was applied to. A
	// request on a higher one arrived on a reconnect; a fixed 2 would miscount
	// a client that dialled twice before the drop.
	DropConn int
	// AcceptedAt is when each connection was accepted, refused ones included.
	// A refused connection is served by nobody, so it shows up here and under
	// no request's Conn.
	AcceptedAt []time.Time
	// Answerable is every sequence number the server would have answered a
	// Republish for with the notification itself, ascending. Anything else got
	// Bad_MessageNotAvailable. Empty on the server that keeps no
	// retransmission queue, and empty under RepublishSubInvalid, where no
	// Republish is answered with anything: every one of them is faulted
	// Bad_SubscriptionIdInvalid instead. runScenario refuses to combine that
	// option with RefusesTransfer so that this stays the whole rule.
	Answerable []uint32
	// Offered is every sequence number the server told the client it still
	// had, which is what a keep-alive and a TransferResult carried verbatim and
	// what a data PublishResponse carried alongside its own sequence number
	// under AdvertisesSentSequence. It is empty under NoRetransmissionQueue,
	// which is the server saying it keeps no retransmission queue at all.
	// Answerable is the part of it the server would really have handed back,
	// and on the default server the two disagree on purpose; see
	// answerableSeqs.
	Offered []uint32
	// SeqAtRecreate is the highest sequence number the server had created when
	// the client's first CreateSubscription after the drop arrived, and zero
	// when no such request arrived. A notification above it was created for
	// the subscription the client put in place of the old one, and the server
	// creates none at all until something is monitored on that subscription,
	// so a delivery above it is the client receiving data again on a
	// subscription it rebuilt. A CreateSubscription that arrives before its
	// own connection's ActivateSession is faulted Bad_SessionNotActivated and
	// never reaches the arm that sets this, which is the same rule
	// republishSeqsAfterDrop reads for a Republish.
	SeqAtRecreate uint32
	// RecreateAfter is how many requests the server had recorded when that
	// same CreateSubscription arrived, and zero when no such request
	// arrived. Requests[RecreateAfter:] is everything recorded after that
	// moment, which is the boundary and no more than that: another
	// connection's goroutine can record a request between the recording of
	// the CreateSubscription and the arm that reads the count, so a request
	// that went over the wire after it can still land below the line; see
	// recorder.recreateAfter. It is set in the one locked block that sets
	// SeqAtRecreate, which is what makes "the client replaced the
	// subscription" and "here is the line its data has to be above" one
	// answer rather than two: a reader cannot find a replacement that the
	// line was never fixed for, or a line with no replacement under it.
	RecreateAfter int
	// HarnessErr is a stage the scenario could not reach; the test reports it
	// as a setup failure.
	HarnessErr string
	// ActivationFaulted is the fixture's own check on itself, and is empty
	// when it holds; see recorder.activationFaulted for what it guards.
	ActivationFaulted string
}

// setupFailed reports why the run never reached the point the test is about,
// or "" if it did. Whatever caused it, no test here can speak past it.
//
// It also carries the fixture's check on itself, and carries it first. Every
// row in the package calls this one and no row has to remember to call
// anything else, which is the point: a guard each row opts into is a guard a
// twelfth row forgets.
//
// Every check of this kind is named …SetupFailed, returns the reason or "",
// and is reported as "could not set the test up: %s": the reason is the
// row's, and the wording is the package's. A row consults one up front when
// nothing it scores could be true without it, and inside its switch when the
// violation it scores would show up anyway -- as the last arm where no
// knownBug stands, and as the first arm where one does, so that a run which
// measured nothing is logged with the known bug rather than passing as if it
// had been scored. republish_after_reconnect_test.go is the one row that
// puts it below the knownBug arm instead, and says in place why.
//
// A row with two kinds of decline splits them, as
// publish_resumes_when_republish_loop_ends_test.go does: the decline that is
// gopcua's own defect goes through the knownBug arm, and the one a
// conforming client earns is a t.Fatalf up with the preconditions.
func setupFailed(run runRecord) string {
	// Ahead of HarnessErr because it is worse: a stage the scenario could not
	// reach leaves the rows nothing to read, while an activation the server
	// did not grant leaves them something that reads as evidence and is not.
	if run.ActivationFaulted != "" {
		return run.ActivationFaulted
	}
	return run.HarnessErr
}

// reconnectSetupFailed reports why the run never got the client back onto an
// activated Session, or "" if it did. A client that never re-activated was
// never offered the Session's subscriptions back, so nothing it did or did
// not do with them is evidence.
func reconnectSetupFailed(run runRecord) string {
	for _, r := range afterDrop(run) {
		if r.Service == "ActivateSessionRequest" {
			return ""
		}
	}
	return "the client never activated a Session after the drop, so the Session's " +
		"subscriptions were never offered back to it"
}

// clientSilentSetupFailed reports why there is nothing of the client's to
// read after the drop, or "" if there is. A client that sent nothing at all
// has not declined to do anything in particular, so no row can score it.
func clientSilentSetupFailed(run runRecord) string {
	if len(afterDrop(run)) == 0 {
		return "the client sent nothing at all after the drop"
	}
	return ""
}

// republishesAfterDrop reads what the client sent on a connection it opened
// after the drop: how many Republish asked for the notification the server
// held, which other sequence numbers it asked for instead, and how many
// requests there were in all. The number asked for is the behaviour, not the
// service name: a client that republishes a number the server never held has
// not noticed what it lost.
//
// A Republish is counted only on a connection the client had already
// activated, which is republishSeqsAfterDrop's rule and for its reason:
// answer() faults a Republish that arrives before its own connection's
// ActivateSession with Bad_SessionNotActivated and never reaches the
// retransmission arm, so the server resends nothing and the client recovers
// nothing. Without the filter a client that pipelines Republish ahead of
// ActivateSession is read as having asked, and republish_after_reconnect_test.go
// reports "this now passes" for a client that got its notification back no
// more than one that never asked at all.
//
// total is every post-drop request and is deliberately not filtered: it is
// the denominator the rows print, not a reading of the client.
func republishesAfterDrop(run runRecord) (asked int, wrongSeqs []uint32, total int) {
	activated := map[int]bool{}
	for _, r := range run.Requests {
		if r.Service == "ActivateSessionRequest" {
			activated[r.Conn] = true
		}
		if r.Conn <= run.DropConn {
			continue
		}
		total++
		switch {
		case r.Service != "RepublishRequest":
		case !activated[r.Conn]:
		case r.RepublishSeq == run.HeldSeq:
			asked++
		default:
			wrongSeqs = append(wrongSeqs, r.RepublishSeq)
		}
	}
	return asked, wrongSeqs, total
}

// republishSeqsAfterDrop is the sequence numbers the client asked the server
// to resend, in the order they went over the wire. It counts only what the
// client sent on a connection it opened after the drop and had already
// activated there: a Republish that arrives before that connection's
// ActivateSession is faulted with Bad_SessionNotActivated rather than
// answered by the retransmission rule, so it is not part of any loop.
// session_unused_before_activation_test.go is the row that scores a client
// for sending one.
func republishSeqsAfterDrop(run runRecord) []uint32 {
	var out []uint32
	activated := map[int]bool{}
	for _, r := range run.Requests {
		switch {
		case r.Service == "ActivateSessionRequest":
			activated[r.Conn] = true
		case r.Service == "RepublishRequest" && r.Conn > run.DropConn && activated[r.Conn]:
			out = append(out, r.RepublishSeq)
		}
	}
	return out
}

// loopRanSetupFailed reports why there was no Republish loop at all, or ""
// if one ran. This is the reason the knownBug arm carries in every row that
// reads the loop: gopcua re-activates and then asks for nothing, so none of
// them has anything to score rather than something to object to.
//
// It subsumes reconnectSetupFailed, so the rows that call both call that one
// first; that order is what makes the knownBug text's "reconnects and
// re-activates" true of every run that reaches this reason.
func loopRanSetupFailed(run runRecord) string {
	if len(republishSeqsAfterDrop(run)) == 0 {
		return "the client sent no Republish after the drop on a connection it had activated, so " +
			"no loop ran"
	}
	return ""
}

// afterDrop is every request the client sent on a connection it opened after
// the drop.
//
// Its rule parts company with the fixture's own whenever the client has more
// than one connection open before the drop: run.DropConn names only the
// connection the held Publish was on, and dropWatcher closes only that one,
// so a second pre-drop connection is above run.DropConn before the drop and
// alive after it, while the fixture's rule is the drop having been applied.
// No client here opens a second connection, so nothing has produced the
// disagreement; data_resumes_on_recreated_subscription_test.go, which cannot
// tolerate it, reads runRecord.RecreateAfter instead.
func afterDrop(run runRecord) []requestRecord {
	var out []requestRecord
	for _, r := range run.Requests {
		if r.Conn > run.DropConn {
			out = append(out, r)
		}
	}
	return out
}

// server is the scripted fixture the client under test talks to.
type server struct {
	rec      *recorder
	ln       *uacp.Listener
	endpoint string

	dropCh chan struct{} // the scenario's one-shot drop command
	opts   runOptions    // the run this server was scripted for

	mu    sync.Mutex         // guards conns
	conns map[int]*uacp.Conn // by connection generation
}

// newServer starts the scripted server on an OS-chosen port.
func newServer(ctx context.Context, opts runOptions) (*server, error) {
	ln, err := uacp.Listen(ctx, "opc.tcp://127.0.0.1:0", nil)
	if err != nil {
		return nil, err
	}
	s := &server{
		rec:      &recorder{},
		ln:       ln,
		endpoint: "opc.tcp://" + ln.Addr().String(),
		dropCh:   make(chan struct{}, 1),
		opts:     opts,
		conns:    map[int]*uacp.Conn{},
	}
	go s.acceptLoop(ctx)
	go s.dropWatcher()
	return s, nil
}

// drop commands a TCP reset of the connection the held Publish is outstanding
// on. The session is not told.
func (s *server) drop() {
	select {
	case s.dropCh <- struct{}{}:
	default:
	}
}

// dropWatcher applies the commanded drop to the connection pinned by heldConn;
// the newest connection could already be a redial.
func (s *server) dropWatcher() {
	<-s.dropCh
	connID := s.rec.heldPublishConn()
	s.mu.Lock()
	conn := s.conns[connID]
	s.mu.Unlock()
	applied := false
	if conn != nil {
		// Marked before the close, so a redial after the reset is refused.
		s.rec.recordDropStarted()
		_ = conn.SetLinger(0)
		applied = conn.Close() == nil
	}
	if applied {
		s.rec.recordDropApplied(s.opts.FurtherLost)
	}
}

func (s *server) acceptLoop(ctx context.Context) {
	for {
		c, err := s.ln.Accept(ctx)
		if err != nil {
			// uacp.Listener.Accept returns an error for a failed UACP
			// handshake as well as for a closed listener, and the two mean
			// opposite things here: one client gave up part way through
			// Hello, or the run is over. Returning on both would end the
			// server for every later connection because one client aborted,
			// and the rows with no knownBug would then fail a client for
			// something the fixture did.
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return
			}
			continue
		}
		s.rec.recordConn()
		// The Hello/Acknowledge handshake is already done, so closing here
		// fails the client's OpenSecureChannel.
		if s.opts.RefuseAfterDrop && s.rec.isDropStarted() {
			_ = c.Close()
			continue
		}
		s.mu.Lock()
		connID := len(s.conns) + 1
		s.conns[connID] = c
		s.mu.Unlock()
		go s.serve(ctx, connID, c)
	}
}

func (s *server) serve(ctx context.Context, connID int, c *uacp.Conn) {
	cfg := &uasc.Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
		Lifetime:          uint32(time.Hour / time.Millisecond),
	}
	errch := make(chan error, 8)
	// Channel, sequence and token ids differ per connection.
	sc, err := uasc.NewServerSecureChannel(s.endpoint, c, cfg, errch, uint32(1000+connID), uint32(100*connID+1), uint32(2000+connID))
	if err != nil {
		return
	}
	defer sc.Close()

	for {
		msg := sc.Receive(ctx)
		if msg == nil || msg.Err != nil {
			return
		}
		req := msg.Request()
		if req == nil {
			continue
		}
		s.rec.recordRequest(connID, req)

		resp := s.answer(connID, req)
		s.rec.recordActivationFault(req, resp)
		if resp == nil {
			continue
		}
		_ = sc.SendResponseWithContext(ctx, msg.RequestID, resp)
	}
}

func (s *server) answer(connID int, req ua.Request) ua.Response {
	h := req.Header().RequestHandle
	if usesSession(serviceName(req)) && !s.rec.isActivated(connID) {
		return newServiceFault(h, ua.StatusBadSessionNotActivated)
	}
	switch r := req.(type) {
	case *ua.CreateSessionRequest:
		return &ua.CreateSessionResponse{
			ResponseHeader:        newResponseHeader(h, ua.StatusOK),
			SessionID:             ua.NewNumericNodeID(1, 6000),
			AuthenticationToken:   ua.NewNumericNodeID(0, 7000),
			RevisedSessionTimeout: 60000,
			ServerEndpoints:       []*ua.EndpointDescription{anonymousAuthEndpoint(s.endpoint)},
			ServerSignature:       &ua.SignatureData{},
		}

	case *ua.ActivateSessionRequest:
		// The session survives the drop, so re-activation succeeds.
		return &ua.ActivateSessionResponse{
			ResponseHeader: newResponseHeader(h, ua.StatusOK),
			ServerNonce:    nil,
			Results:        []ua.StatusCode{},
		}

	case *ua.CloseSessionRequest:
		return &ua.CloseSessionResponse{ResponseHeader: newResponseHeader(h, ua.StatusOK)}

	case *ua.ReadRequest:
		return &ua.ReadResponse{
			ResponseHeader: newResponseHeader(h, ua.StatusOK),
			Results: []*ua.DataValue{{
				EncodingMask: ua.DataValueValue,
				Value:        ua.MustVariant([]string{"http://opcfoundation.org/UA/"}),
			}},
		}

	case *ua.CreateSubscriptionRequest:
		s.rec.recordNewSubscription()
		return &ua.CreateSubscriptionResponse{
			ResponseHeader:            newResponseHeader(h, ua.StatusOK),
			SubscriptionID:            SubID,
			RevisedPublishingInterval: r.RequestedPublishingInterval,
			RevisedLifetimeCount:      1000,
			RevisedMaxKeepAliveCount:  10,
		}

	case *ua.CreateMonitoredItemsRequest:
		if r.SubscriptionID == SubID {
			s.rec.recordItemsMonitored()
		}
		results := make([]*ua.MonitoredItemCreateResult, len(r.ItemsToCreate))
		for i := range r.ItemsToCreate {
			results[i] = &ua.MonitoredItemCreateResult{
				StatusCode:              ua.StatusOK,
				MonitoredItemID:         uint32(i + 1),
				RevisedSamplingInterval: 200,
				RevisedQueueSize:        1,
			}
		}
		return &ua.CreateMonitoredItemsResponse{
			ResponseHeader:  newResponseHeader(h, ua.StatusOK),
			Results:         results,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}

	case *ua.PublishRequest:
		// With nothing monitored there is no data, only keep-alives.
		if !s.rec.isAnyMonitored() {
			time.Sleep(publishPacing)
			return newKeepAliveResponse(h, s.rec.peekNextSeq(), s.available())
		}
		seq, hold := s.rec.publishStep(connID)
		if hold {
			return nil
		}
		time.Sleep(publishPacing)
		return newPublishResponse(h, seq, s.availableWith(seq))

	case *ua.TransferSubscriptionsRequest:
		if s.opts.RefusesTransfer {
			return newServiceFault(h, ua.StatusBadServiceUnsupported)
		}
		results := make([]*ua.TransferResult, len(r.SubscriptionIDs))
		for i := range r.SubscriptionIDs {
			results[i] = &ua.TransferResult{
				StatusCode:               ua.StatusOK,
				AvailableSequenceNumbers: s.available(),
			}
		}
		return &ua.TransferSubscriptionsResponse{
			ResponseHeader:  newResponseHeader(h, ua.StatusOK),
			Results:         results,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}

	case *ua.RepublishRequest:
		// A known gap: the subscription id is neither checked here nor kept by
		// recordRequest, which takes only RetransmitSequenceNumber. A client
		// asking for the right numbers against a subscription that never
		// existed therefore scores as a correct walk in
		// republish_asks_for_each_missed_number_test.go and
		// publish_resumes_when_republish_loop_ends_test.go, and as a correct
		// ask in republish_after_reconnect_test.go. It is an oversight rather
		// than a judgement that the id does not matter: a Publish's
		// acknowledgements are recorded with their SubID, and the
		// CreateMonitoredItems arm does check r.SubscriptionID == SubID.
		// Closing it means faulting a mismatch here and keeping the id on
		// requestRecord so those rows can read it.
		if s.opts.RepublishSubInvalid {
			return newServiceFault(h, ua.StatusBadSubscriptionIDInvalid)
		}
		if slices.Contains(s.answerableSeqs(), r.RetransmitSequenceNumber) {
			return newRepublishResponse(h, r.RetransmitSequenceNumber)
		}
		return newServiceFault(h, ua.StatusBadMessageNotAvailable)
	}
	// Services the scenario does not script fault.
	return newServiceFault(h, ua.StatusBadServiceUnsupported)
}

// answerableSeqs is every sequence number a Republish gets a notification
// back for. Under RefusesTransfer the server keeps the retransmission queue
// OPC 10000-4 §5.14.1.1 describes: the notification it withheld is in it,
// and so is anything FurtherLost queued behind it. Otherwise the server has no
// message to give back at all. Either way OPC 10000-4 §6.7 names
// Bad_MessageNotAvailable as the status that ends the Republish loop, and that
// is what every other sequence number gets. RepublishSubInvalid is answered
// above this and never reaches it, and runScenario refuses to combine the two,
// so the list is empty whenever that option is set.
//
// runRecord.Answerable is this list, so a test reads the one rule the server
// answers by rather than restating it.
func (s *server) answerableSeqs() []uint32 {
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	return s.answerableSeqsLocked()
}

func (s *server) answerableSeqsLocked() []uint32 {
	if !s.opts.RefusesTransfer {
		return nil
	}
	return s.rec.availableSeqsLocked()
}

// available is what the server tells the client it still has: the list a
// keep-alive and a TransferResult carry, and the one availableWith builds a
// data PublishResponse's list from. It is the recorder's availableSeqsLocked,
// except under NoRetransmissionQueue, where the server keeps no retransmission
// queue and says so with an empty list on every answer, as OPC 10000-4
// §5.14.5.2 requires of such a server and as gopcua's own server does.
//
// runRecord.Offered is this list, so a test reads what the client was told
// rather than restating the rule the server told it by.
func (s *server) available() []uint32 {
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	return s.availableLocked()
}

func (s *server) availableLocked() []uint32 {
	if s.opts.NoRetransmissionQueue {
		return []uint32{}
	}
	return s.rec.availableSeqsLocked()
}

// availableWith is the list a data PublishResponse carries: what the server
// still has, and under AdvertisesSentSequence the sequence number of that
// response as well. The server has just created that notification and has not
// given it away yet, so it is available for retransmission by the same rule as
// everything else in the queue, and it sorts above all of them.
func (s *server) availableWith(seq uint32) []uint32 {
	available := s.available()
	if !s.opts.AdvertisesSentSequence || slices.Contains(available, seq) {
		return available
	}
	return append(available, seq)
}

// usesSession reports whether a service request counts as using the Session.
// The fixture faults one that arrives before the connection's
// ActivateSession, and session_unused_before_activation_test.go reads the
// same rule: one statement of one domain fact.
func usesSession(service string) bool {
	switch service {
	case "CreateSessionRequest", "ActivateSessionRequest", "CloseSessionRequest",
		"GetEndpointsRequest", "FindServersRequest":
		return false
	}
	return true
}

func serviceName(v any) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", v), "*ua.")
}

func newResponseHeader(handle uint32, status ua.StatusCode) *ua.ResponseHeader {
	return &ua.ResponseHeader{
		Timestamp:          time.Now(),
		RequestHandle:      handle,
		ServiceResult:      status,
		ServiceDiagnostics: &ua.DiagnosticInfo{},
		StringTable:        []string{},
		AdditionalHeader:   ua.NewExtensionObject(nil),
	}
}

// responseStatus is the ServiceResult a response carried. A response with no
// header at all reads as StatusBadInternalError rather than as success: it is
// not an answer the client can act on, and recordActivationFault is the only
// caller, where reading it as success would be the one wrong answer.
func responseStatus(resp ua.Response) ua.StatusCode {
	if h := resp.Header(); h != nil {
		return h.ServiceResult
	}
	return ua.StatusBadInternalError
}

func newServiceFault(handle uint32, status ua.StatusCode) ua.Response {
	return &ua.ServiceFault{ResponseHeader: newResponseHeader(handle, status)}
}

// newNotification builds a notification whose value is its own sequence
// number, so a delivery record shows which one the client received.
func newNotification(seq uint32) *ua.NotificationMessage {
	return &ua.NotificationMessage{
		SequenceNumber: seq,
		PublishTime:    time.Now(),
		NotificationData: []*ua.ExtensionObject{
			ua.NewExtensionObject(&ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{{
					ClientHandle: 1,
					Value:        &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(int32(seq))},
				}},
				DiagnosticInfos: []*ua.DiagnosticInfo{},
			}),
		},
	}
}

func newPublishResponse(handle, seq uint32, available []uint32) *ua.PublishResponse {
	return &ua.PublishResponse{
		ResponseHeader:           newResponseHeader(handle, ua.StatusOK),
		SubscriptionID:           SubID,
		AvailableSequenceNumbers: available,
		MoreNotifications:        false,
		NotificationMessage:      newNotification(seq),
		Results:                  []ua.StatusCode{},
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
}

// newRepublishResponse hands back a notification the server still has, which
// is how a client recovers one it missed.
func newRepublishResponse(handle, seq uint32) *ua.RepublishResponse {
	return &ua.RepublishResponse{
		ResponseHeader:      newResponseHeader(handle, ua.StatusOK),
		NotificationMessage: newNotification(seq),
	}
}

// newKeepAliveResponse answers a Publish with no data. It carries the sequence
// number the next notification will use.
func newKeepAliveResponse(handle, nextSeq uint32, available []uint32) *ua.PublishResponse {
	return &ua.PublishResponse{
		ResponseHeader: newResponseHeader(handle, ua.StatusOK),
		SubscriptionID: SubID,
		NotificationMessage: &ua.NotificationMessage{
			SequenceNumber:   nextSeq,
			PublishTime:      time.Now(),
			NotificationData: []*ua.ExtensionObject{},
		},
		AvailableSequenceNumbers: available,
		Results:                  []ua.StatusCode{},
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
}

func anonymousAuthEndpoint(url string) *ua.EndpointDescription {
	return &ua.EndpointDescription{
		EndpointURL: url,
		Server: &ua.ApplicationDescription{
			ApplicationURI:  "urn:reconnect:server",
			ProductURI:      "urn:reconnect:server",
			ApplicationName: ua.NewLocalizedText("reconnect"),
			ApplicationType: ua.ApplicationTypeServer,
			DiscoveryURLs:   []string{url},
		},
		SecurityMode:      ua.MessageSecurityModeNone,
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		UserIdentityTokens: []*ua.UserTokenPolicy{{
			PolicyID:  "anonymous_none",
			TokenType: ua.UserTokenTypeAnonymous,
		}},
		TransportProfileURI: "http://opcfoundation.org/UA-Profile/Transport/uatcp-uasc-uabinary",
	}
}

// runOptions configures one scenario run.
type runOptions struct {
	// RefuseAfterDrop closes every connection accepted once the drop has started.
	RefuseAfterDrop bool
	// RefusesTransfer selects the other server this package scripts: one that
	// answers TransferSubscriptions with Bad_ServiceUnsupported, as gopcua's
	// own server does, and that still holds the notification it withheld, so a
	// Republish asking for that sequence number gets it back. The default
	// server is the opposite on both counts.
	RefusesTransfer bool
	// FurtherLost is how many notifications beyond the held one the server
	// creates as the drop is applied and keeps in its retransmission queue,
	// never having reached the client. They give a Republish loop a range to
	// walk; see the package comment. Only the RefusesTransfer server answers a
	// Republish at all, so on the other one this changes nothing but the
	// sequence numbers the stream resumes at.
	FurtherLost int
	// RepublishSubInvalid answers every Republish with
	// Bad_SubscriptionIdInvalid, the status OPC 10000-4 §6.7 makes the Client
	// answer by creating a new Subscription. It cannot be combined with
	// RefusesTransfer: both decide what a Republish is answered with, and
	// runRecord.Answerable could then say neither.
	RepublishSubInvalid bool
	// NoRetransmissionQueue makes the server say it keeps no retransmission
	// queue: every PublishResponse and every TransferResult carries an empty
	// availableSequenceNumbers. OPC 10000-4 §5.14.5.2 gives that list its
	// meaning, and gopcua's own server is one of these. It cannot be combined
	// with RefusesTransfer, which selects the server that does keep the queue
	// and hands notifications back out of it.
	NoRetransmissionQueue bool
	// AdvertisesSentSequence puts the sequence number of a data PublishResponse
	// into that response's own availableSequenceNumbers, which OPC 10000-4
	// §5.14.5.2 requires of a server that keeps a retransmission queue: the
	// list holds everything available for retransmission "including the
	// sequence number of this response if it is not a keep-alive Message".
	// Without it the fixture advertises nothing until it has withheld
	// something, so the one notification it hands over before the drop arrives
	// with an empty list, and a client that reads the list acknowledges
	// nothing in the whole run. It cannot be combined with
	// NoRetransmissionQueue, which is the server that keeps no queue to
	// advertise.
	AdvertisesSentSequence bool
	// ReconnectInterval is the client's wait between reconnection attempts;
	// zero means 100ms.
	ReconnectInterval time.Duration
	// Settle is how long to wait after the drop before snapshotting; zero means
	// settleDefault.
	Settle time.Duration
}

// runScenario drives one reconnect scenario against the scripted server and
// returns the runRecord the test reads. It never aborts: a stage it could not
// reach is recorded in HarnessErr.
func runScenario(opts runOptions) runRecord {
	var out runRecord
	if opts.RepublishSubInvalid && opts.RefusesTransfer {
		out.HarnessErr = "RepublishSubInvalid and RefusesTransfer both decide what a Republish is " +
			"answered with, so runRecord.Answerable could not say what the server would have handed back"
		return out
	}
	if opts.AdvertisesSentSequence && opts.NoRetransmissionQueue {
		out.HarnessErr = "AdvertisesSentSequence has the server advertise the notification it is " +
			"sending and NoRetransmissionQueue has it advertise nothing at all, so the server would " +
			"have been telling the client two different things"
		return out
	}
	if opts.NoRetransmissionQueue && opts.RefusesTransfer {
		out.HarnessErr = "NoRetransmissionQueue says the server keeps no retransmission queue and " +
			"RefusesTransfer selects the server that keeps one and hands notifications back out of " +
			"it, so the server would have been telling the client two different things"
		return out
	}
	if opts.Settle == 0 {
		opts.Settle = settleDefault
	}
	if opts.ReconnectInterval == 0 {
		opts.ReconnectInterval = 100 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := newServer(ctx, opts)
	if err != nil {
		out.HarnessErr = fmt.Sprintf("the scripted server could not start: %v", err)
		return out
	}
	defer func() { srv.ln.Close() }()

	c, err := opcua.NewClient(srv.endpoint,
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(opts.ReconnectInterval),
		opcua.RequestTimeout(3*time.Second),
		opcua.DialTimeout(3*time.Second),
	)
	if err != nil {
		out.HarnessErr = fmt.Sprintf("constructing the client: %v", err)
		return out
	}
	if err := c.Connect(ctx); err != nil {
		out.HarnessErr = fmt.Sprintf("the initial connect failed: %v", err)
		return out
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = c.Close(closeCtx)
		closeCancel()
	}()

	notifs := make(chan *opcua.PublishNotificationData, 32)
	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{Interval: 200 * time.Millisecond}, notifs)
	if err != nil {
		out.HarnessErr = fmt.Sprintf("subscribing: %v", err)
		return out
	}
	go func() {
		for n := range notifs {
			dcn, ok := n.Value.(*ua.DataChangeNotification)
			if !ok || len(dcn.MonitoredItems) == 0 {
				continue
			}
			if seq, ok := dcn.MonitoredItems[0].Value.Value.Value().(int32); ok {
				srv.rec.recordDelivery(uint32(seq))
			}
		}
	}()

	item := opcua.NewMonitoredItemCreateRequestWithDefaults(monitoredNode, ua.AttributeIDValue, 1)
	if _, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, item); err != nil {
		out.HarnessErr = fmt.Sprintf("monitoring the item: %v", err)
		return out
	}

	// Wait until the stream is live: the application's callback has received
	// a notification.
	deadline := time.After(10 * time.Second)
	for {
		if srv.rec.deliveryCount() > 0 {
			break
		}
		select {
		case <-deadline:
			out.HarnessErr = "no notification reached the application before the drop command"
			return out
		case <-time.After(20 * time.Millisecond):
		}
	}

	// Wait until the server is holding a Publish, so that its notification
	// cannot reach the client before the drop.
	heldDeadline := time.After(10 * time.Second)
	for {
		if srv.rec.heldPublishSeq() != 0 {
			break
		}
		select {
		case <-heldDeadline:
			out.HarnessErr = "the client never left a publish outstanding for the server to hold; the loss could not be constructed"
			return out
		case <-time.After(10 * time.Millisecond):
		}
	}

	srv.drop()
	dropDeadline := time.After(2 * time.Second)
	for {
		if srv.rec.isDropApplied() {
			out.DropAt = srv.rec.dropAppliedAt()
			out.DropConn = srv.rec.heldPublishConn()
			break
		}
		select {
		case <-dropDeadline:
			out.HarnessErr = "the commanded transport drop never applied"
			return out
		case <-time.After(10 * time.Millisecond):
		}
	}

	// One notification is lost, not several: see the package comment.
	if held, created := srv.rec.heldPublishSeq(), srv.rec.seqCreatedAtDrop(); held != created {
		out.HarnessErr = fmt.Sprintf("the server held notification %d back and created %d more before the "+
			"drop, so more than the one notification the tests describe was lost", held, created-held)
		return out
	}

	time.Sleep(opts.Settle)

	srv.snapshot(&out)
	return out
}
