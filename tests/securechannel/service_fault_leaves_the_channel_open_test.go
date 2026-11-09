//go:build conformance

// A ServiceFault is the answer to the one request that drew it, and it
// leaves the SecureChannel open: the fault is a service-level response,
// not a communication fault, so a client that receives one keeps the
// channel and its next request travels on the same connection.
//
// The clauses: OPC 10000-4 §7.34: "The ServiceFault parameter shall be
// returned instead of the Service response message when the
// serviceResult is a StatusCode with Severity Bad." and its Table 173
// entry: "ServiceFault … An error response sent when a service level
// error occurs." OPC 10000-4 §5.3: "A bad code is returned in
// serviceResult if the Service itself failed." "The Client shall always
// check the response parameters, especially all StatusCodes associated
// with each operation." "In some cases, parsing errors are fatal and it
// is not possible to return a fault." "In these cases, these errors may
// be treated as a communication fault which requires the SecureChannel
// to be re-established (see 5.6)." §5.3 names only the fatal parsing
// errors, where no fault can be returned, as a reason to re-establish
// the SecureChannel; a ServiceFault is the case where a fault was
// returned - a service-level answer to one request - and no clause says
// it ends the channel. Part 4 gives the channel and the session their
// own closing services.
//
// Neighbouring rows: security_token_lifetime_is_measured_from_receipt_
// test.go and expired_security_token_is_rejected_test.go score the
// token lifecycle that surrounds the channel, where this row scores
// the channel's own survival; received_sequence_number_is_verified_
// test.go scores the channel's answer to a chunk that fails
// verification, which Part 6 does order closed, where a ServiceFault
// is a well-formed answer no clause orders anything of.
//
// The row runs against the scripted peer from harness.go, under
// SecurityPolicy None, with the client under test built by the row as
// the real opcua.Client with AutoReconnect(true): with
// AutoReconnect(false) the unfixed monitor marks the client closed
// without touching the wire, so the wire cannot separate a fixed tree
// from an unfixed one, and the row does not score what it cannot
// separate. Connect itself sends a Read of the namespace array,
// NodeID i=2255, before the row sends any request of its own, and an
// unfixed client's reconnect sends that Read again on its own, so the
// moment a Read is sent cannot tell the row's Reads from the
// library's: the row marks its own Reads with a MaxAge no Connect or
// reconnect Read carries, the script faults only the first marked
// Read - with a ServiceFault carrying Bad_TooManyOperations - and
// answers every other Read normally, the second marked one included,
// and the records are read back by that mark. The peer answers
// CreateSession and ActivateSession normally, and it keeps serving
// connections, so everything a client does to come back - a new
// connection, a new HEL/ACK and OPN, the activation again - is
// visible in its records. The Read answers the script sends -
// Connect's own, and every marked Read but the faulted one - are
// built by the harness's readResponse, and the check that the faulted
// Read returned the injected status is the harness's isServiceFault.
// See harness.go.
//
// The row asserts, from the peer's records alone, that the second
// marked Read was answered on the original connection - the same
// connection generation, no new connection accepted, the original
// connection never ended after the fault by a close the client made,
// the channel never closed or replaced on it after the fault - and it
// deliberately asserts nothing about the error the faulted Read
// returned beyond the setup check that it was the injected one,
// nothing about the session, and nothing about which error a fixed
// client logs. The client's monitor tears down and redials on its own
// goroutine, so the drop is not in the peer's records the moment the
// row's calls return: a body that read them then scored the unfixed
// tree as kept, a false pass. The row waits out a settle window
// before scoring the channel kept: it polls the peer's records for
// the drop evidence - a new connection accepted, the original
// connection ended, the second marked Read arriving on a later
// connection, the channel closed or replaced on the original
// connection - and only a window that closes holding none of the four
// is a kept channel. The fourth fact catches the drop that keeps the
// TCP connection open, and a fixed tree cannot show it in the run:
// the client's only OPN after the channel opens is its renewal, 75 %
// of the hour-long token away and asking for a Renew, and the CLO the
// client's own close at teardown sends lands after the row has
// scored. The original connection's end counts as evidence only when
// the client ended it: the harness's read bound ends a connection
// that sat idle that long, and the close it leaves looks in the
// records exactly like a client's own, so a window of the bound or
// longer would score a fixed client's quiet connection as the
// original connection ending - a false drop, which an earlier draft
// of this row hit on PR #904's tree - which is why the close records
// mark the bound's ends and the row does not count them. A fixed
// client merely waits the window out, and waiting only observes, so
// every fact the window scores was put in the records by the client's
// own teardown or not at all; the one verdict a window can get wrong
// now is kept - never drop - when it closes before a teardown puts
// its facts in the records, and a teardown that lands after the
// window looks to the row exactly like the fix: the row cannot tell
// the two apart, which is why the window's bound is generous against
// the milliseconds the unfixed monitor's teardown takes. The second
// marked Read is sent while a reconnect may be in flight on an
// unfixed tree - a client between channels may refuse the send - so
// the row retries it within a bound until the peer has decoded it,
// and only what the peer received is scored, never the error a
// refused send returned. It declines, and does not score, when the
// fixture fails its half: the peer's own failure, the initial connect
// failing, the fault not answering the first marked Read, the second
// marked Read neither arriving nor the channel being dropped. A frame
// the harness flags as the client's own bytes is not a decline: the
// setup check reports it as gopcua's own behaviour, and the row
// scores the run as gopcua's failure with t.Errorf, never declines
// it. The knownBug constant records that gopcua does not do this yet:
// the receive path bubbles a Bad serviceResult into the message's
// error at uasc/secure_channel.go:426 (if status :=
// resp.Header().ServiceResult; status != ua.StatusOK { msg.Err =
// status }), the dispatcher forwards that error to the client's
// channel-error path at uasc/secure_channel.go:274 (case s.errch <-
// msg.Err), and the monitor answers any error there by marking the
// connection disconnected (client.go:325, client.go:341) and
// recreating the secure channel (client.go:380, client.go:416), so on
// main the faulted Read is followed by a teardown and a redial. The
// knownBug arm is the drop: a new connection accepted, the original
// connection ended, the second marked Read arriving on a later
// connection, or the channel closed or replaced on the original
// connection, each after the fault and none of which a fixed client
// shows; what deletes the constant is the fix that stops a
// service-level fault from being treated as a channel error, as PR
// #904 does; measured on PR #911 the row still logs the known bug,
// because that fix touches the token's clock and not the fault path.
// When the constant stands, the drop is logged as the known bug; when
// the channel is kept, that is a hard error telling whoever runs it
// to delete the constant, every arm that reads it, and anything only
// those arms used.
//
// No reference stack has been read for this behaviour; the measurement
// on main and on the two candidate fixes is the row's own.
package securechannel

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// serviceFaultStatus is the ServiceFault status the script answers the
// first marked Read with. It is Bad, so it travels as a ServiceFault
// per OPC 10000-4 §7.34, and no client path mints it on its
// own, so a returned fault is unambiguously the injected one.
var serviceFaultStatus = ua.StatusBadTooManyOperations

// mark is the MaxAge the row stamps its own Reads with: a
// fractional value no Read of the library's carries - Connect's own
// Read of the namespace array and a reconnecting client's both go out
// with MaxAge zero - so the script can fault exactly the first marked
// Read and the records can tell the row's Reads from the library's.
const mark = 9876.5

// settleWindow is how long the row polls the peer's records for the
// drop evidence before scoring the channel kept. The client's monitor
// tears down and redials on its own goroutine, so the window is
// generous against the milliseconds that teardown takes on a loaded
// machine. It must stay under the harness's read bound: a connection
// that the bound ends for idling leaves a close the row must not
// count as the client's own.
const settleWindow = 8 * time.Second

// retryBound is how long the row keeps sending the second
// marked Read until the peer has decoded it. A client between
// channels refuses the send, and an unfixed monitor's redial - a new
// connection, the activation and the namespace read again - completes
// in well under a second even loaded, so the bound is a safety margin
// and never the thing the verdict waits on.
const retryBound = 15 * time.Second

// callBound is the context bound on each call the row
// makes: Connect, every marked Read and the close at teardown. It
// sits above the milliseconds a healthy exchange takes, so a loaded
// machine finishes inside it, and it keeps a client wedged past its
// own timers from holding the row.
const callBound = 6 * time.Second

// poll is how often the row rereads the peer's records
// while it waits. It only observes, so the interval is a choice about
// the row's own cost and nothing else.
const poll = 25 * time.Millisecond

func TestServiceFaultLeavesTheChannelOpen(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is
	// logged in place of a failure. ../README.md says when to delete
	// it.
	const knownBug = "a Bad serviceResult is bubbled into the " +
		"channel-error path and the monitor recreates the secure " +
		"channel, so a ServiceFault that answered one request costs " +
		"the whole channel and the client comes back on a new connection"

	// The script answers the client's own exchanges normally - a
	// CreateSession with a session worth activating, the activation,
	// every Read the library sends for the namespace array, a
	// CloseSession - and faults exactly the first marked Read it
	// decodes, answering every marked Read after it normally, the
	// second one included. markedReads is counted on the peer's serve
	// goroutines, one per connection, so the count must be atomic. The
	// token's lifetime is an hour, so the renewal the client schedules
	// at 75 % of it stays out of the row's way.
	var markedReads atomic.Uint32
	token := channelToken{
		channelID: 1,
		tokenID:   1,
		createdAt: time.Now(),
		lifetime:  uint32(time.Hour / time.Millisecond),
	}
	peer, teardown, why := startPeer(peerScript{
		issue: token,
		renew: token,
		answer: func(req ua.Request, reqID uint32) ua.Response {
			switch r := req.(type) {
			case *ua.CreateSessionRequest:
				return createSessionResponse(reqID)
			case *ua.ActivateSessionRequest:
				return &ua.ActivateSessionResponse{
					ResponseHeader:  responseHeader(reqID, ua.StatusOK),
					Results:         []ua.StatusCode{},
					DiagnosticInfos: []*ua.DiagnosticInfo{},
				}
			case *ua.CloseSessionRequest:
				return &ua.CloseSessionResponse{
					ResponseHeader: responseHeader(reqID, ua.StatusOK),
				}
			case *ua.ReadRequest:
				if r.MaxAge == mark && markedReads.Add(1) == 1 {
					return &ua.ServiceFault{
						ResponseHeader: responseHeader(reqID, serviceFaultStatus),
					}
				}
				return readResponse(r, reqID)
			}
			return nil
		},
	})
	t.Cleanup(teardown)
	if why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}

	client, err := opcua.NewClient("opc.tcp://"+peer.addr(),
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
	)
	if err != nil {
		t.Fatalf("could not set the test up: build the client: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(),
			callBound)
		defer cancel()
		_ = client.Close(ctx)
	})

	connectCtx, connectCancel := context.WithTimeout(context.Background(),
		callBound)
	connectErr := client.Connect(connectCtx)
	connectCancel()
	if connectErr != nil {
		if why, fromGopcua := peer.setupCheck(); why != "" {
			if fromGopcua {
				t.Errorf("gopcua's own behaviour made this run "+
					"unscoreable: %s", why)
				return
			}
			t.Fatalf("could not set the test up: %s", why)
		}
		t.Fatalf("could not set the test up: the initial connect "+
			"failed: %v", connectErr)
	}

	firstCtx, firstCancel := context.WithTimeout(context.Background(),
		callBound)
	_, firstErr := client.Read(firstCtx, markedRead())
	firstCancel()
	if why, fromGopcua := peer.setupCheck(); why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
			return
		}
		t.Fatalf("could not set the test up: %s", why)
	}
	if !isServiceFault(firstErr, serviceFaultStatus) {
		t.Fatalf("could not set the test up: the first marked Read "+
			"returned %v, not the injected ServiceFault %s, so the "+
			"fault never answered the request the story needs it to "+
			"answer", firstErr, serviceFaultStatus)
	}

	secondArrived := false
	retryDeadline := time.Now().Add(retryBound)
	for !secondArrived && time.Now().Before(retryDeadline) {
		readCtx, readCancel := context.WithTimeout(context.Background(),
			callBound)
		_, _ = client.Read(readCtx, markedRead())
		readCancel()
		secondArrived = markedReads.Load() >= 2
		if !secondArrived {
			time.Sleep(poll)
		}
	}

	// The settle window: the drop is not in the records the moment the
	// row's calls return - the monitor tears down and redials on its
	// own goroutine - so the row polls the records for the drop
	// evidence before it scores the channel kept.
	var facts []string
	settleDeadline := time.Now().Add(settleWindow)
	for {
		facts = dropEvidence(peer.record())
		secondArrived = markedReads.Load() >= 2
		if len(facts) > 0 || !time.Now().Before(settleDeadline) {
			break
		}
		time.Sleep(poll)
	}

	if why, fromGopcua := peer.setupCheck(); why != "" {
		if fromGopcua {
			t.Errorf("gopcua's own behaviour made this run unscoreable: %s", why)
			return
		}
		t.Fatalf("could not set the test up: %s", why)
	}
	if !secondArrived && len(facts) == 0 {
		t.Fatalf("could not set the test up: the second marked Read " +
			"did not arrive at the peer and the channel was not " +
			"dropped within the row's bounds, so the run cannot be " +
			"scored")
	}

	switch {
	case len(facts) > 0 && knownBug != "":
		t.Logf("known bug: %s (the drop evidence was: %s)", knownBug,
			strings.Join(facts, "; "))
	case len(facts) == 0 && knownBug != "":
		t.Errorf("this now passes (the second marked Read was answered " +
			"on the original connection, no new connection was " +
			"accepted, the original connection never ended after the " +
			"fault and the channel was never closed or replaced on " +
			"it) - delete the knownBug constant, every arm that " +
			"reads it, and anything only those arms used, in the " +
			"commit that fixed it")
	case len(facts) == 0:
		t.Logf("the faulted Read left the channel open: the second " +
			"marked Read was answered on the original connection, no " +
			"new connection was accepted, the original connection " +
			"never ended after the fault and the channel was never " +
			"closed or replaced on it")
	default:
		t.Errorf("the faulted Read was followed by a drop (%s), and "+
			"no knownBug constant stands",
			strings.Join(facts, "; "))
	}
}

// markedRead is the Read the row drives: the namespace
// array node, the very node Connect's own Read asks for, stamped with
// the MaxAge mark that tells the row's Reads from the library's. No arm
// scores the value the script answers it with.
func markedRead() *ua.ReadRequest {
	return &ua.ReadRequest{
		MaxAge: mark,
		NodesToRead: []*ua.ReadValueID{{
			NodeID:      ua.NewNumericNodeID(0, 2255),
			AttributeID: ua.AttributeIDValue,
		}},
	}
}

// isMarkedRead reports whether req is one of the row's own
// marked Reads.
func isMarkedRead(req ua.Request) bool {
	r, ok := req.(*ua.ReadRequest)
	return ok && r.MaxAge == mark
}

// firstMarkedRead returns the connection generation and the moment
// of the first marked Read the peer decoded; the caller has
// verified one arrived. The fault answered that Read, so the moment
// anchors what counts as "after the fault".
func firstMarkedRead(rec peerRecord) (conn int, at time.Time) {
	for _, r := range rec.Requests {
		if isMarkedRead(r.Req) {
			return r.Conn, r.At
		}
	}
	return 0, time.Time{}
}

// dropEvidence names the facts in the peer's records that say the
// channel was dropped after the fault - a new connection accepted,
// the original connection ended by a close the client made, a marked
// Read arriving on a later connection than the faulted one, the
// channel closed or replaced on the original connection - or holds
// none, which is the channel kept. A close the harness's read bound
// wrote does not count - closeRecord in harness.go says why - and
// the settle window is what keeps the two from meeting.
func dropEvidence(rec peerRecord) []string {
	faultConn, faultAt := firstMarkedRead(rec)
	var facts []string
	for _, a := range rec.Accepts {
		if a.Conn > faultConn {
			facts = append(facts, "a new connection was accepted "+
				"after the fault")
			break
		}
	}
	for _, c := range rec.Closed {
		if c.Conn == faultConn && !c.Bound && c.At.After(faultAt) {
			facts = append(facts, "the original connection ended "+
				"after the fault by a close the client made")
		}
	}
	for _, r := range rec.Requests {
		if r.Conn > faultConn && isMarkedRead(r.Req) {
			facts = append(facts, "a marked Read arrived on a later "+
				"connection")
			break
		}
	}
	closedOrReplaced := false
	for _, r := range rec.Requests {
		if _, ok := r.Req.(*ua.CloseSecureChannelRequest); ok &&
			r.Conn == faultConn && r.At.After(faultAt) {
			closedOrReplaced = true
		}
	}
	for _, o := range rec.OPNs {
		if o.Conn == faultConn && o.At.After(faultAt) &&
			o.Type == ua.SecurityTokenRequestTypeIssue {
			closedOrReplaced = true
		}
	}
	if closedOrReplaced {
		facts = append(facts, "the channel was closed or replaced on "+
			"the original connection after the fault")
	}
	return facts
}

// createSessionResponse is the normal CreateSession answer:
// a session id, an authentication token and a revised timeout, with a
// non-null ServerSignature - the client dereferences it - and no
// endpoints, which leaves the client on the default anonymous policy.
// Under SecurityPolicy None the client verifies no signature, so no
// forged session state is needed for the response to be well formed.
func createSessionResponse(reqID uint32) *ua.CreateSessionResponse {
	return &ua.CreateSessionResponse{
		ResponseHeader:        responseHeader(reqID, ua.StatusOK),
		SessionID:             ua.NewNumericNodeID(0, 1),
		AuthenticationToken:   ua.NewNumericNodeID(0, 2),
		RevisedSessionTimeout: 3600000,
		ServerSignature:       &ua.SignatureData{},
	}
}
