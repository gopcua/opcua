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

	// settleRetries replaces settleDefault for these runs: long enough for
	// about eleven attempts at the short interval and three at the long one.
	settleRetries = 3500 * time.Millisecond

	// minAttempts is the fewest attempts that give one gap to measure.
	minAttempts = 2
)

func TestRetryDelayAfterFailedOpen(t *testing.T) {
	// The long interval is the binding one: a window that holds minAttempts at
	// it holds more at the short one.
	if settleRetries < minAttempts*longInterval {
		t.Fatalf("could not set the test up: the settle window of %s holds only %d attempts at an "+
			"interval of %s, too few to measure a gap. Lengthen settleRetries",
			settleRetries, int(settleRetries/longInterval), longInterval)
	}

	shortRun := retryScenario(shortInterval)
	if why := setupFailed(shortRun); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	shortest, attempts := smallestGap(shortRun)
	if attempts < minAttempts {
		t.Fatalf("the client was configured to retry every %s and made %d connection attempt(s) "+
			"in the %s after the drop, where %d are expected: it is not honouring the setting",
			shortInterval, attempts, settleRetries, minAttempts)
	}
	if shortest < shortInterval {
		t.Errorf("the client retried %s after the previous attempt, sooner than its configured "+
			"interval of %s (%d attempts after the drop)", shortest, shortInterval, attempts)
	}

	longRun := retryScenario(longInterval)
	if why := setupFailed(longRun); why != "" {
		t.Fatalf("could not set the test up: %s", why)
	}
	longest, longAttempts := smallestGap(longRun)
	if longAttempts < minAttempts {
		t.Fatalf("the client was configured to retry every %s and made %d connection attempt(s) "+
			"in the %s after the drop, where %d are expected: it is not honouring the setting",
			longInterval, longAttempts, settleRetries, minAttempts)
	}
	if longest < longInterval {
		t.Errorf("the client retried %s after the previous attempt, sooner than its configured "+
			"interval of %s (%d attempts after the drop)", longest, longInterval, longAttempts)
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

// retryScenario runs the scenario with the client configured to wait interval
// between attempts.
func retryScenario(interval time.Duration) record {
	return runScenario(scenarioOptions{
		ReconnectInterval: interval,
		RefuseAfterDrop:   true,
		Settle:            settleRetries,
	})
}

// smallestGap reports the smallest gap between the connection attempts the
// client made after the drop, and how many it made. Below minAttempts there is
// no gap to measure and smallest is zero.
func smallestGap(run record) (smallest time.Duration, attempts int) {
	// The client opens a connection at the start of every attempt, and Accepts
	// counts the refused ones too.
	var at []time.Time
	for _, a := range run.Accepts {
		if a.After(run.DropAt) {
			at = append(at, a)
		}
	}
	if len(at) < minAttempts {
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
