// A client read of a tag concurrent with the application's write to it
// observes no conflicting access: while an application goroutine publishes
// to MapNamespace, a client reading the same tag over the wire completes
// its reads against a live writer, and no reader or the runtime ever
// observes a torn or conflicting state.
//
// The clause is the Go memory model, not an OPC UA shall: Attribute reads
// ns.Data without the namespace lock (server/namespace_map.go:195
// and :256) while
// SetValue writes it under the lock (server/namespace_map.go:48), and a
// Go map read concurrent with a map write is a data race the runtime
// catches and reports as "WARNING: DATA RACE", failing the run of the
// binary that raced. A map read concurrent with a map write is also a
// fatal runtime abort ("concurrent map read and map write",
// go/src/runtime), but the report the arms key
// on is the detector's. The specification's
// nearest clause is OPC 10000-3 §5.6.2 8.58, "In general Atomicity is
// expected of OPC UA read and write operations" - descriptive rather
// than a shall, and not checkable in this tree. gopcua does not have the
// behaviour yet: measured 2026-11-05 with go test -count=1 -race -run
// TestReadWhileWritingIsRaceFree ./tests/server/, which failed with
// "WARNING: DATA RACE" on the read at server/namespace_map.go:195 in
// (*MapNamespace).Attribute against the SetValue write.
//
// Neighbouring rows: setvalue_on_a_subscribed_tag_returns_test.go and
// setattribute_on_a_subscribed_tag_returns_test.go are the write half of
// the same property, and this row is why the ledger in
// tests/server/README.md calls an RLock in Attribute "the obvious fix"
// with care: each half alone is passed trivially by breaking the other,
// and a change that passes all three is the only shape a fix can take.
// Neither of those rows scores a concurrent reader; this one scores
// only the reads, not the write. The fixture configures no logger, so
// the logger-guarded whole-map read of ns.Data
// (server/namespace_map.go:178) is a declared blind spot the row
// cannot score.
//
// The row runs against the in-tree gopcua server holding one
// MapNamespace and a real gopcua client connected over the wire - the
// shared fixture in harness.go - brought up inside the window child the
// body spawns, with the writer on its own goroutine and the reads
// issued through the client's Read service. That Read asks for the
// tag's Value and DataType attributes, so a fix must lock every
// ns.Data read in Attribute.
// See harness.go.
//
// The body declares the knownBug constant inside the test function and
// scores the window in a child of the test binary: the race
// detector's failure cannot be an arm in-process - it fails the test
// binary itself, so a knownBug arm can never intercept it - so the
// window runs in a child, selected by an environment variable, and the
// arms read the child's output. The child-process scorer lives in this
// file, and the child prints its read count on stdout under the marker
// race_free_reads_completed, which the parent parses. The arm that logs
// the known bug keys on the detector's report naming the Attribute
// frame - the bug's presence, which a fixed library does not show; the
// arm that demands
// the constant's deletion keys on the child exiting clean after at least
// ten reads inside a two-second window - the floor is what stops an
// empty run from passing. A clean child under the floor, a child that
// never printed its count, or a binary not built with -race are each a
// decline, not a pass.
//
// A fix that only adds RLock to Attribute wedges the server: the reads
// complete until the first write takes the namespace lock and notifies,
// and from then on the writer never returns from SetValue - the
// notification path re-enters Attribute while the caller holds the lock -
// and every later Read blocks too, so the read count says little and the
// arm keys on the child's wedge line instead. The child times the
// writer's stop out, prints its wedge line and exits failed with no
// detector report to show. That is gopcua's own behaviour, so the row
// scores it as a failure, not a decline - the proxy the
// knownBug arm keys on is gone and the outcome is still wrong, which is
// the wrong-fix arm tests/README.md describes. The deletion arm
// therefore fires only for a fix under which the writer still returns -
// RLock plus notifying outside the lock. Makefile `test` runs this
// package with -race.
//
// The wedge happens only through a subscription: SetValue notifies under
// the lock, and the notification re-enters Attribute only when a
// monitored item exists on the tag. So the window child must hold a
// monitored item on the tag its writer writes, even though the row
// scores only the reads. Without one, an RLock-only fix
// completes its reads cleanly, the child exits clean past the floor, and
// the row demands the constant's deletion for a fix that still deadlocks
// every server with a subscriber - the two write-half rows are named "on
// a subscribed tag" for the same reason. The read count is asserted
// separately because the detector only reports interleavings it sees, so
// a pass is evidence rather than proof; the writer publishes a strictly
// increasing int32 sequence and a read that moves backwards is a torn
// observation no detector needs to be watching for.
//
// The window child paces its writer a writeInterval apart; the pace
// keeps the window an alternation of writes and reads rather than a hot
// loop the floor's reads have to squeeze between. No drain of the
// subscription's notifications is needed on top of that: nothing reads
// the client's 16-entry h.notify (harness.go:135), so once it fills the
// client's notification delivery blocks (Subscription.notify,
// subscription.go:331) and the client stops publishing, which leaves the
// Read service and the server's own notification loop unaffected
// (server/subscription_service.go:379 and :426); the row's outcome was
// measured the same with and without a drain (2026-11-07).
//
// No reference stack has been read for this behaviour. The instrument
// the verdict rests on - the Go race detector - is outside gopcua; what
// that does and does not settle for a server row is in README.md.

package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

// The child-process scorer's names. The child prints the marker and
// the parent parses it, so both must spell it the same way.
const (
	// raceRowWindowEnv selects the window branch when the row re-runs
	// itself as a child; the one func Test per file is kept that way.
	raceRowWindowEnv = "RACE_ROW_WINDOW"

	// raceRowReadsMarker is the stdout marker the window child prints its
	// read count under and the parent parses; the floor is read off it.
	raceRowReadsMarker = "race_free_reads_completed"

	// raceReport is the detector's report line and raceFrameAttribute the
	// frame in it that names the read the knownBug constant stands for.
	// The known-bug arm requires both: a report outside that frame is a
	// race the constant does not cover, whether or not it stands.
	raceReport         = "WARNING: DATA RACE"
	raceFrameAttribute = "(*MapNamespace).Attribute"

	// raceRowWedgeText is the wedge clause the window child prints when
	// its writer does not stop after the window, and the wrong-fix arm's
	// key: a lock-only fix wedges the writer, and the arm catches that
	// fix by its own report of the wedge.
	raceRowWedgeText = "SetValue is wedged on the namespace lock"

	// The window child's failure-line prefixes and runChild's kill wrap,
	// named for the same reason as the marker: a reword on one side of
	// the exec must not silently reroute the other's arms.
	raceRowDeclinePrefix  = "could not set the test up: "
	raceRowScoredPrefix   = "gopcua's own behaviour made this run unscoreable: "
	raceRowKilledByParent = "killed by the parent"
)

const (
	// windowLen is the window the child reads inside, and readFloor the
	// number of completed reads below which a clean child is an empty
	// run; tests/server/README.md's ledger names both, at two seconds
	// and ten.
	windowLen = 2 * time.Second
	readFloor = 10

	// childWait is how long the parent waits on a child before killing
	// it. It sits above the child's window, the respond deadline on its
	// writer's stop select and the ~30s teardown lag tests/server/README.md
	// measures
	// for a wedged server, so a wedged child is read from its output
	// rather than lost.
	childWait = 60 * time.Second

	// writeInterval spaces the writer's publishes. It keeps the window an
	// alternation of writes and reads.
	writeInterval = 20 * time.Millisecond
)

func TestReadWhileWritingIsRaceFree(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it.
	const knownBug = "a client Read concurrent with the application's SetValue " +
		"conflicts with it on MapNamespace.Data: Attribute reads ns.Data without " +
		"the namespace lock, server/namespace_map.go:195 and :256, while " +
		"SetValue writes it under the lock"

	// The window branch: this binary re-running itself as the child whose
	// output the parent's arms read. A setup failure here is a fixture
	// cause - the parent never reached a window it could score - so the
	// contract's decline line, "could not set the test up: %s", is the
	// right one for the child to print; the parent reads it as the
	// decline it is.
	if os.Getenv(raceRowWindowEnv) != "" {
		h, teardown, why := startServerAndClient()
		defer teardown()
		if why != "" {
			t.Fatalf(raceRowDeclinePrefix+"%s", why)
		}

		// The subscription is load-bearing even though the row scores no
		// notification: the header says the notification path's re-entry
		// into Attribute is the only thing that turns an RLock-only fix
		// into the wedge the wrong-fix arm exists to catch.
		subTeardown, why := h.subscribeTo(tag)
		if why != "" {
			t.Fatalf(raceRowDeclinePrefix+"%s", why)
		}
		defer subTeardown()

		// The writer publishes a strictly increasing int32 sequence; a
		// read that moves backwards is a torn observation no detector
		// needs to be watching for. The channel is unbuffered on purpose
		// - the only signal is the close after the loop returns.
		stop := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := int32(1); ; i++ {
				select {
				case <-stop:
					return
				case <-time.After(writeInterval):
					h.ns.SetValue(tag, i)
				}
			}
		}()

		windowEnd := time.Now().Add(windowLen)
		var reads, backwards int
		var last int32
		for time.Now().Before(windowEnd) {
			// A read that the server never answers - the wedge a
			// lock-only fix produces - comes back at this context's
			// deadline, and the loop then finds its window behind it.
			rctx, cancel := context.WithTimeout(context.Background(), respond)
			resp, err := h.client.Read(rctx, &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{
					{NodeID: ua.NewStringNodeID(h.ns.ID(), tag)},
					{NodeID: ua.NewStringNodeID(h.ns.ID(), tag), AttributeID: ua.AttributeIDDataType},
				},
			})
			cancel()
			if err != nil || len(resp.Results) != 2 || resp.Results[0].Status != ua.StatusOK ||
				resp.Results[1].Status != ua.StatusOK {
				continue
			}
			if resp.Results[0].Value == nil {
				continue
			}
			v, ok := resp.Results[0].Value.Value().(int32)
			if !ok {
				continue
			}
			reads++
			if v < last {
				backwards++
			}
			last = v
		}

		// The read count travels to the parent on stdout, under the named
		// marker, before anything a wedged library could hang or fail:
		// the parent reads it even from a child killed on its way out. It
		// is not the only such early line: when the writer-stop wait
		// below times out, the child also prints the wedge text to
		// stdout at that moment, ahead of its own t.Errorf, which
		// without -test.v is written only after the teardowns.
		fmt.Printf("%s=%d\n", raceRowReadsMarker, reads)

		close(stop)
		select {
		case <-done:
		case <-time.After(respond):
			// stdout first: the wedge line must reach the parent even if
			// the teardowns outlive the childWait budget.
			fmt.Printf("%s\n", raceRowWedgeText)
			t.Errorf(raceRowScoredPrefix+"the writer did not stop within %v "+
				"after the window closed, so %s", respond, raceRowWedgeText)
		}

		if backwards > 0 {
			t.Errorf(raceRowScoredPrefix+"%d of %d reads observed a value that "+
				"moved backwards, which is a torn observation no detector needs "+
				"to be watching for", backwards, reads)
		}
		return
	}

	// The parent branch. A binary not built with -race cannot score the
	// row at all - the detector has nothing to report - so that is a
	// decline before any child runs; the window child's output is what
	// the arms read.
	if !raceDetector {
		t.Fatalf("could not set the test up: this binary was not built with " +
			"-race, so the detector cannot score the row; Makefile test runs it")
	}

	windowOut, windowErr := runChild(raceRowWindowEnv)

	// The known-bug arm keys on the detector's report naming the
	// Attribute frame, a presence a fixed library does not show, so the
	// exit code is consulted only past this arm, as README.md says.
	if everyRaceReportNamesAttribute(windowOut) && knownBug != "" {
		t.Logf("known bug: %s (the detector reported the conflict on the "+
			"frame the constant names, in the window child: %s)",
			knownBug, excerpt(windowOut))
		return
	}
	if strings.Contains(windowOut, raceReport) {
		t.Errorf("the detector reported a conflict in the window child that "+
			"the knownBug arm does not cover - a race outside %q, or the "+
			"known race back after the constant was deleted (child output: %s)",
			raceFrameAttribute, excerpt(windowOut))
		return
	}

	// With no report in the output the exit and the count are material: a
	// clean child past the floor is the deletion arm's precondition, and
	// the count is what separates a scored window from an empty run.
	reads, why := parseReads(windowOut)
	if why != "" {
		t.Fatalf(raceRowDeclinePrefix+"%s (the window child exited %v; "+
			"its output: %s)", why, windowErr, excerpt(windowOut))
	}

	// A lost child - killed by the parent at childWait (runChild's wrap)
	// or by anything else (exec's error spells "signal: killed") - is a
	// decline when it did not self-report a failure of its own; the wedge
	// arm below is for a child that lived to self-report one.
	if windowErr != nil &&
		!strings.Contains(windowOut, raceRowScoredPrefix) &&
		!strings.Contains(windowOut, raceRowDeclinePrefix) {
		t.Fatalf(raceRowDeclinePrefix+"the window child was lost - it exited "+
			"%v with no failure of its own (output: %s)",
			windowErr, excerpt(windowOut))
	}

	switch {
	case windowErr == nil && reads >= readFloor && knownBug != "":
		t.Errorf("this now passes (the window child completed %d reads in %v "+
			"under -race with no detector report) - delete the knownBug "+
			"constant, every arm that reads it, and anything only those arms "+
			"used, in the commit that fixed it", reads, windowLen)
	case windowErr == nil && reads >= readFloor:
		t.Logf("the window child completed %d reads in %v under -race with "+
			"no detector report", reads, windowLen)
	case windowErr == nil:
		t.Fatalf("could not set the test up: the window child exited clean "+
			"after %d reads in %v, under the floor of %d, so the window "+
			"reached nothing the detector could have scored (child output: %s)",
			reads, windowLen, readFloor, excerpt(windowOut))
	case strings.Contains(windowOut, raceRowWedgeText):
		// The wrong-fix arm: no report, but the child's own wedge line.
		// The header says why this is gopcua's own behaviour and a scored
		// failure, not a decline.
		t.Errorf("gopcua's own behaviour made this run unscoreable: the window "+
			"child reported %q - reads completed until the first write took "+
			"the namespace lock and notified, and then the writer never "+
			"returned, the wedge a lock-only fix produces, so there is no "+
			"clean window to score (child output: %s)",
			raceRowWedgeText, excerpt(windowOut))
	default:
		t.Errorf("gopcua's own behaviour made this run unscoreable: the window "+
			"child completed %d reads, then failed a check of its own after "+
			"the window - its line: %s", reads, excerpt(windowOut))
	}
}

// everyRaceReportNamesAttribute reports whether the child's output holds
// at least one detector report and every report names the Attribute frame.
func everyRaceReportNamesAttribute(out string) bool {
	blocks := strings.Split(out, raceReport)
	if len(blocks) <= 1 {
		return false
	}
	for _, block := range blocks[1:] {
		if !strings.Contains(block, raceFrameAttribute) {
			return false
		}
	}
	return true
}

// runChild re-runs this test binary with -test.run pinned to this row's
// test function and the branch env var set, waits up to childWait for it,
// and returns its combined output and exit error. A child that could not
// be started at all returns empty output with the reason as the error.
func runChild(branchEnv string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), childWait)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestReadWhileWritingIsRaceFree$")
	cmd.Env = append(os.Environ(), branchEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil && ctx.Err() != nil {
		err = fmt.Errorf("%s at the %v childWait budget", raceRowKilledByParent, childWait)
	}
	return string(out), err
}

// parseReads reads the child's read count off its marker line. It returns
// 0 and the reason no count could be read when the line is missing or
// malformed.
func parseReads(out string) (int, string) {
	prefix := raceRowReadsMarker + "="
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(line, prefix))
		if err != nil {
			return 0, fmt.Sprintf("the child's read-count line %q does not parse", line)
		}
		return n, ""
	}
	return 0, "the child printed no read-count line, so its window cannot be scored"
}

// excerpt trims a child's output to a size a failure line can carry,
// keeping the head and the tail: the detector's report line sits at the
// head of a child's output and its own failure line prints last, so
// trimming cannot hide either.
func excerpt(out string) string {
	const half = 1000
	out = strings.TrimSpace(out)
	if len(out) <= 2*half {
		return out
	}
	return out[:half] + "..." + out[len(out)-half:]
}
