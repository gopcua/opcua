// After a failed OpenSecureChannel, a client must wait before it tries again.
// OPC 10000-4 §6.7: "If the OpenSecureChannel fails, the Client should delay
// the retry for a configurable time."
//
// The clause only says "should", so the test checks gopcua's own contract: a
// client configured with ReconnectInterval(d) leaves at least d between
// successive attempts.
//
// After the drop the server accepts every connection and closes it at once, so
// each attempt fails at OpenSecureChannel. The server timestamps each accept,
// and the client's attempts are read from those. See harness.go.
//
// A client with a fixed wait longer than the configured interval would pass at
// one interval, so the scenario runs at two, and the shorter run must also
// stay under the longer interval.
package reconnect

import (
	"testing"
	"time"
)

const (
	shortInterval = 300 * time.Millisecond
	longInterval  = 1200 * time.Millisecond

	// settle is how long each run goes on after the drop: about eleven
	// attempts at the short interval and about three at the long one.
	settle = 3500 * time.Millisecond

	// minAttempts is the fewest attempts that give one gap to measure.
	minAttempts = 2
)

func TestRetryDelayAfterFailedOpen(t *testing.T) {
	shortest, attempts, ok := smallestGap(t, shortInterval)
	if !ok {
		return
	}
	if shortest < shortInterval {
		t.Errorf("the client retried %s after the previous attempt, sooner than its configured "+
			"interval of %s (%d attempts after the drop)", shortest, shortInterval, attempts)
	}

	longest, attempts, ok := smallestGap(t, longInterval)
	if !ok {
		return
	}
	if longest < longInterval {
		t.Errorf("the client retried %s after the previous attempt, sooner than its configured "+
			"interval of %s (%d attempts after the drop)", longest, longInterval, attempts)
	}

	// A client with a fixed wait longer than the short interval passes both
	// checks above, so the short run must also stay under the long interval.
	if shortest >= longInterval {
		t.Errorf("configured with %s the client still left at least %s between attempts, "+
			"as it did when configured with %s: it ignores the setting", shortInterval, shortest, longInterval)
	}
	t.Logf("smallest gap between attempts: %s at an interval of %s, %s at %s",
		shortest, shortInterval, longest, longInterval)
}

// smallestGap runs the scenario with the client configured to wait interval
// and returns the smallest gap between its attempts after the drop. ok is false
// if the client made too few attempts for there to be a gap, which it reports
// as a failure.
func smallestGap(t *testing.T, interval time.Duration) (smallest time.Duration, attempts int, ok bool) {
	t.Helper()
	if settle < minAttempts*interval {
		t.Fatalf("settle is %s, too short for %d attempts at an interval of %s: "+
			"a client honouring the interval could not make enough attempts to measure a gap. Lengthen settle",
			settle, minAttempts, interval)
	}
	run := runScenario(scenarioOptions{
		ReconnectInterval: interval,
		RefuseAfterDrop:   true,
		Settle:            settle,
	})

	// A client cannot cause either of these.
	switch {
	case run.HarnessErr != "":
		t.Fatalf("could not set the test up: %s", run.HarnessErr)
	case !run.DropApplied:
		t.Fatalf("could not set the test up: the connection was never dropped")
	}

	// The client opens a connection at the start of every attempt. The server
	// refuses them all, and Accepts counts refused connections too.
	var at []time.Time
	for _, a := range run.Accepts {
		if a.After(run.DropAt) {
			at = append(at, a)
		}
	}

	if len(at) < minAttempts {
		t.Errorf("the client was configured to retry every %s and made %d connection attempt(s) "+
			"in the %s after the drop, where %d are expected: it is not honouring the setting",
			interval, len(at), settle, minAttempts)
		return 0, len(at), false
	}

	smallest = at[1].Sub(at[0])
	for i := 2; i < len(at); i++ {
		if gap := at[i].Sub(at[i-1]); gap < smallest {
			smallest = gap
		}
	}
	return smallest, len(at), true
}
