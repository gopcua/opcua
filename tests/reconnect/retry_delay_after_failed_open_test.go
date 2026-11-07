//go:build conformance

// After a failed OpenSecureChannel, a client must wait before it tries again.
// OPC 10000-4 §6.7: "If the OpenSecureChannel fails, the Client should delay
// the retry for a configurable time."
//
// The clause only says "should", so the test checks gopcua's own contract: a
// client configured with ReconnectInterval(d) leaves at least d between
// successive attempts.
//
// No other row here measures when the client acts, only what it sends, so
// this one has no neighbouring row.
//
// After the drop the server accepts every connection and closes it at once, so
// each attempt fails at OpenSecureChannel. The server timestamps each accept,
// and the client's attempts are read from those. See harness.go.
//
// A client with a fixed wait longer than the configured interval would pass at
// one interval, so the scenario runs at two, and the shorter run must also
// stay under the longer interval.
//
// Two known-good clients fail this, for different reasons, and neither is a
// reason to drop it. Eclipse Milo's reconnect backoff is hard-coded, one
// second doubling to sixteen — read from source, not run here — so it has no
// configured interval to honour.
// node-opcua 2.187.19 does have a configured interval and ignores it:
// measured against this scenario at 300ms, 1.2s and 5s it reconnected about
// six thousand times a second in each case, median gap 0.145ms, because its
// backoff fires once and then its endpoint probe loop runs free. That is a
// defect in node-opcua rather than a disagreement about the clause.
// open62541 and UA-.NETStandard were not read.
//
// gopcua passes it, measured by this row, which is why the row carries no
// knownBug constant.
//
// The test is kept because it checks gopcua's own contract rather than the
// clause above, which only says "should".

package reconnect

import (
	"testing"
	"time"
)

const (
	shortInterval = 300 * time.Millisecond
	longInterval  = 1200 * time.Millisecond

	// settleRetries replaces settleDefault for these runs: long enough for
	// about eleven attempts at the short interval and three at the long one.
	settleRetries = 3500 * time.Millisecond

	// minRetryAttempts is the fewest attempts that give one gap to measure.
	minRetryAttempts = 2
)

func TestRetryDelayAfterFailedOpen(t *testing.T) {
	// A tripwire, not a check: all of it is constants, so a settleRetries
	// shortened, or a longInterval lengthened, until the window no longer
	// holds minRetryAttempts fails the build rather than a run. The long
	// interval is the binding one; the short one is well inside the window.
	//
	// It budgets a whole interval more than the attempts need: minRetryAttempts
	// attempts leave minRetryAttempts-1 gaps between them, and the extra
	// interval is slack for the time the client spends noticing the drop
	// before the first attempt, which no gap covers.
	const _ = uint64(settleRetries - minRetryAttempts*longInterval)

	var gaps []time.Duration
	for _, interval := range []time.Duration{shortInterval, longInterval} {
		run := runScenario(runOptions{
			ReconnectInterval: interval,
			RefuseAfterDrop:   true,
			Settle:            settleRetries,
		})
		if why := setupFailed(run); why != "" {
			t.Fatalf("could not set the test up: %s", why)
		}

		smallest, attempts := smallestGap(run)
		if attempts < minRetryAttempts {
			t.Errorf("the client was configured to retry every %s and made %d connection attempt(s) "+
				"in the %s after the drop, where %d are expected: it is not honouring the setting",
				interval, attempts, settleRetries, minRetryAttempts)
			return
		}
		if smallest < interval {
			t.Errorf("the client retried %s after the previous attempt, sooner than its configured "+
				"interval of %s (%d attempts after the drop)", smallest, interval, attempts)
		}
		gaps = append(gaps, smallest)
	}
	// Both are smallest gaps: one from the run configured short, one from the
	// run configured long. A client that honours the setting leaves a smaller
	// smallest gap in the short run than the long interval itself.
	atShort, atLong := gaps[0], gaps[1]

	if atShort >= longInterval {
		t.Errorf("configured with %s the client still left at least %s between attempts, "+
			"as it did when configured with %s: it ignores the setting", shortInterval, atShort, longInterval)
	}
	t.Logf("smallest gap between attempts: %s at an interval of %s, %s at %s",
		atShort, shortInterval, atLong, longInterval)
}

// smallestGap reports the smallest gap between the connection attempts the
// client made after the drop, and how many it made. It reads AcceptedAt rather
// than request connections because a refused attempt leaves no request. Below
// minRetryAttempts there is no gap to measure and smallest is zero.
func smallestGap(run runRecord) (smallest time.Duration, attempts int) {
	var at []time.Time
	for _, a := range run.AcceptedAt {
		if a.After(run.DropAt) {
			at = append(at, a)
		}
	}
	if len(at) < minRetryAttempts {
		return 0, len(at)
	}

	smallest = at[1].Sub(at[0])
	for i := 2; i < len(at); i++ {
		if gap := at[i].Sub(at[i-1]); gap < smallest {
			smallest = gap
		}
	}
	return smallest, len(at)
}
