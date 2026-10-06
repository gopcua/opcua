package matrix

import (
	"slices"

	"github.com/gopcua/opcua/tests/spec/faults"
)

// knownDefects holds the predicted failures of the §6.7 suite on
// main's client, written from the measured part4 verdicts before the
// first matrix run: one entry per rule that failed there in its
// hand-written spec, labelled with that spec's issue, applying to
// every fault of the scenario where the rule runs.
var knownDefects = []KnownDefect{
	{Issue: "issue-879", Check: "RepublishesFromNextSequence", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "SendsNoPublishBeforeNotAvailable", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "KeepsSubscriptionID", Applies: everyFaultOf("SessionSurvives")},
	{Issue: Unfiled("client sends a TransferSubscriptions request for a subscription its own session owns"), Check: "SendsNoTransferForOwnSubscription", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-895", Check: "RecreatesAfterRefusal", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-879", Check: "RepublishesRecreatedFromOne", Applies: everyFaultOf("SubscriptionsLost")},
}

// everyFaultOf says a defect applies to every fault of one scenario.
func everyFaultOf(name string) func(scenario string, f faults.Fault) bool {
	return func(scenario string, _ faults.Fault) bool {
		return scenario == name
	}
}

// KnownDefects returns the predicted defect table of the §6.7 suite.
func KnownDefects() []KnownDefect {
	return slices.Clone(knownDefects)
}
