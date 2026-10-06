package matrix

import (
	"slices"
	"strings"

	"github.com/gopcua/opcua/tests/spec/faults"
)

// knownDefects holds the predicted failures of the §6.7 suite on
// main's client, written from the measured part4 verdicts before the
// first matrix run, and the scenario-level failures the corrected
// harness measured in its control cases and never-fired faults: one
// entry per failing behaviour, labelled with its issue, applying to
// every fault of the scenario — or, for the never-fired faults, to
// every fault targeting the messages the client does not send.
var knownDefects = []KnownDefect{
	{Issue: "issue-879", Check: "RepublishesFromNextSequence", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "SendsNoPublishBeforeNotAvailable", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "KeepsSubscriptionID", Applies: everyFaultOf("SessionSurvives")},
	{Issue: Unfiled("client sends a TransferSubscriptions request for a subscription its own session owns"), Check: "SendsNoTransferForOwnSubscription", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-895", Check: "RecreatesAfterRefusal", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-879", Check: "RepublishesRecreatedFromOne", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-879", Check: "DeliverEachValueOnce", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "ResumePublishing", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-895", Check: "ResumePublishing", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-895", Check: "KeepOneSubscriptionPerClientSubscription", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: Unfiled("the retained value under the forgotten subscription is lost; a correct client cannot retransmit it either"), Check: "DeliverEachValueOnce", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-879", Check: "HaveFired", Applies: faultTargeting("SessionSurvives", "Publish", "Republish")},
	{Issue: "issue-879", Check: "HaveFired", Applies: faultTargeting("SubscriptionsLost", "CreateSubscription", "CreateMonitoredItems", "Publish", "Republish")},
}

// everyFaultOf says a defect applies to every fault of one scenario.
func everyFaultOf(name string) func(scenario string, f faults.Fault) bool {
	return func(scenario string, _ faults.Fault) bool {
		return scenario == name
	}
}

// faultTargeting says a defect applies to every fault of one scenario
// that targets one of the named services.
func faultTargeting(scenario string, services ...string) func(string, faults.Fault) bool {
	return func(name string, f faults.Fault) bool {
		if name != scenario {
			return false
		}
		parts := strings.Split(f.Name(), "/")
		if len(parts) < 2 {
			return false
		}
		switch parts[0] {
		case "RequestLost", "ResponseLost", "CutAfterResponse", "DelayBelowTimeout", "DelayAboveTimeout", "Overload":
			return slices.Contains(services, parts[1])
		}
		return false
	}
}

// KnownDefects returns the predicted defect table of the §6.7 suite.
func KnownDefects() []KnownDefect {
	return slices.Clone(knownDefects)
}
