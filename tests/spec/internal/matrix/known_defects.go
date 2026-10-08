package matrix

import (
	"slices"
	"strings"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
)

// The unfiled issues the triage named, one label per group of cases
// with one root cause.
var (
	issueFailedSubscriptionStep      = Unfiled("failed-subscription-step", "a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it")
	issueConnectionFailureOnActivate = Unfiled("connection-failure-on-activate", "a connection failure during ActivateSession makes the client forget its session without retrying or closing it")
	issueTimedOutActivationSession   = Unfiled("timed-out-activation-session", "after an ActivateSession timeout the old session stays open on the server")
	issueDrainedConnectionError      = Unfiled("drained-connection-error", "the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel")
	issueHELHandshakeHang            = Unfiled("hel-handshake-hang", "the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it")
)

// recreatePathFaults lists the faults under which the client does not
// reactivate its session but creates a new one. On that path it
// recreates its subscriptions and resumes publishing, so the delivery
// and publishing entries below, which belong to the reactivation path,
// do not cover them.
var recreatePathFaults = []string{
	"CutAfterResponse/OpenSecureChannel", "DelayAboveTimeout/ActivateSession",
	"RequestLost/ActivateSession", "ResponseLost/ActivateSession"}

// cutPublishFaults lists the faults that cut the connection on the
// Publish service: the workload answers one held Publish right after
// the arm on them, so their armed cut fires on that exchange and their
// labelled HaveFired and ResumePublishing checks pass on main.
var cutPublishFaults = []string{
	"CutAfterResponse/Publish", "RequestLost/Publish", "ResponseLost/Publish"}

// knownDefects holds the known-defect table of the §6.7 suite and the
// §5.14 predictions: the predicted failures of the original run, the
// entries the triage of the corrected matrix's fault-specific failures
// added, and the §5.14 entries measured before the suite existed —
// one issue per group of cases with one root cause. Each entry applies
// to exactly its group's cases: by scenario, by fault predicate, or by
// enumerated fault name where no predicate fits.
//
// The §5.14 publish-timeout rule has no entry: a held Publish timed out
// on a first subscription already passes on main. The two conditional
// §5.14 rules — RepublishesSkippedSequence and
// PublishesAgainAfterTooManyPublishRequests — have no measured verdict
// on any tree yet, so they get no prediction; the measurement decides.
var knownDefects = []SuiteDefect{
	{Issue: "issue-879", Check: "RepublishesFromNextSequence", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "SendsNoPublishBeforeNotAvailable", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "KeepsSubscriptionID", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "SendsNoTransferForOwnSubscription", Applies: everyFaultOf("SessionSurvives")},
	{Issue: "issue-879", Check: "DeliverEachValueOnce", Applies: everyFaultExcept("SessionSurvives", recreatePathFaults...)},
	{Issue: "issue-879", Check: "ResumePublishing", Applies: everyFaultExcept("SessionSurvives", recreatePathFaults...)},
	{Issue: "issue-879", Check: "HaveFired", Applies: faultTargetingExcept("SessionSurvives", cutPublishFaults, "Publish", "Republish")},
	{Issue: "issue-879", Check: "RecreatesAfterRefusal", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-879", Check: "RepublishesRecreatedFromOne", Applies: everyFaultOf("SubscriptionsLost")},
	{Issue: "issue-879", Check: "HaveFired", Applies: faultTargetingExcept("SubscriptionsLost", cutPublishFaults, "CreateSubscription", "CreateMonitoredItems", "Publish", "Republish")},
	{Issue: "issue-895", Check: "ResumePublishing", Applies: everyFaultExcept("SubscriptionsLost", recreatePathFaults...)},
	{Issue: "issue-895", Check: "KeepOneSubscriptionPerClientSubscription", Applies: everyFaultExcept("SubscriptionsLost", recreatePathFaults...)},
	{Issue: "issue-879", Check: "ResumePublishing", Applies: faultsNamed("SessionLost",
		"DelayAboveTimeout/Read",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations",
		"Overload/Publish/Bad_TooManyPublishRequests", "RequestLost/Read", "ResponseLost/Read")},
	{Issue: "issue-879", Check: "KeepOneSubscriptionPerClientSubscription", Applies: faultsNamed("SessionLost",
		"DelayAboveTimeout/Read", "RequestLost/Read", "ResponseLost/Read")},
	{Issue: "issue-879", Check: "RecreatesAfterRefusal", Applies: faultsNamed("SessionLost",
		"RequestLost/Read", "ResponseLost/Read")},
	{Issue: issueFailedSubscriptionStep,
		Check: "CloseEveryKnownSession", Applies: faultsNamed("SessionLost",
			"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
			"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
			"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
			"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
			"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
			"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions")},
	{Issue: issueFailedSubscriptionStep,
		Check: "KeepOneSessionOpen", Applies: faultsNamed("SessionLost",
			"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
			"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
			"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
			"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
			"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
			"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions")},
	{Issue: issueFailedSubscriptionStep,
		Check: "RecreatesAfterRefusal", Applies: faultsNamed("SessionLost", "CutAfterResponse/CreateSubscription")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "CloseEveryKnownSession", Applies: faultsNamedIn([]string{"SessionSurvives", "SubscriptionsLost"},
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "KeepOneSessionOpen", Applies: faultsNamedIn([]string{"SessionSurvives", "SubscriptionsLost"},
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "CloseEveryKnownSession", Applies: faultsNamed("SessionLost", "CutAfterResponse/CreateSession")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "KeepOneSessionOpen", Applies: faultsNamed("SessionLost", "CutAfterResponse/CreateSession")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "CreatesNoSession", Applies: faultsNamed("SessionSurvives",
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "ReactivatesSession", Applies: faultsNamed("SessionSurvives",
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession")},
	{Issue: issueConnectionFailureOnActivate,
		Check: "CreatesSessionOnlyAfterActivateFailed", Applies: faultsNamed("SessionLost",
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession")},
	{Issue: issueHELHandshakeHang, Check: "CreatesSessionOnlyAfterActivateFailed", Applies: faultsNamed("SessionLost", "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "KeepOneSessionOpen", Applies: faultsNamed("SessionLost", "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "KeepOneSubscriptionPerClientSubscription", Applies: faultsNamed("SessionLost", "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "RecreatesAfterRefusal", Applies: faultsNamed("SessionLost", "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "ResumePublishing", Applies: faultsNamed("SessionLost", "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "CloseEveryKnownSession", Applies: faultsNamedIn([]string{"SessionSurvives", "SubscriptionsLost"}, "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "CreatesNoSession", Applies: faultsNamed("SessionSurvives", "Link/HELUnanswered")},
	{Issue: issueHELHandshakeHang, Check: "ReactivatesSession", Applies: faultsNamed("SessionSurvives", "Link/HELUnanswered")},
	{Issue: "issue-828", Check: "CreatesSessionOnlyAfterActivateFailed", Applies: faultsNamed("SessionLost", "Link/Stall")},
	{Issue: "issue-828", Check: "DeliverEachValueOnce", Applies: faultsNamed("SessionLost", "Link/Stall")},
	{Issue: "issue-828", Check: "RecreatesAfterRefusal", Applies: faultsNamed("SessionLost", "Link/Stall")},
	{Issue: "issue-828", Check: "ResumePublishing", Applies: faultsNamed("SessionLost", "Link/Stall")},
	{Issue: "issue-828", Check: "CreatesNoSession", Applies: faultsNamed("SessionSurvives", "Link/Stall")},
	{Issue: "issue-828", Check: "ReactivatesSession", Applies: faultsNamed("SessionSurvives", "Link/Stall")},
	{Issue: issueTimedOutActivationSession,
		Check: "CloseEveryKnownSession", Applies: faultsNamedIn([]string{"SessionSurvives", "SubscriptionsLost"}, "DelayAboveTimeout/ActivateSession")},
	{Issue: issueTimedOutActivationSession,
		Check: "KeepOneSessionOpen", Applies: faultsNamedIn([]string{"SessionSurvives", "SubscriptionsLost"}, "DelayAboveTimeout/ActivateSession")},
	{Issue: issueDrainedConnectionError,
		Check: "CloseEveryKnownSession", Applies: faultsNamed("SessionLost", "CutAfterResponse/CreateMonitoredItems")},
	{Issue: issueDrainedConnectionError,
		Check: "CloseEveryKnownSession", Applies: faultsNamedIn([]string{"SessionSurvives", "SubscriptionsLost"}, "CutAfterResponse/Read")},
	{Issue: issueDrainedConnectionError,
		Check: "ResumePublishing", Applies: faultsNamed("SessionLost", "CutAfterResponse/CreateMonitoredItems")},
	// The §5.14 predictions: the parked publish loop is the #895
	// mechanism, so every fault of CancelThenSubscribe fails the cycle
	// count and the sentinel both.
	{Issue: "issue-895", Check: "KeepsPublishingAfterCancelThenSubscribe", Applies: everyFaultOf("CancelThenSubscribe")},
	{Issue: "issue-895", Check: "ResumePublishing", Applies: everyFaultOf("CancelThenSubscribe")},
}

// everyFaultOf says a defect applies to every fault of one scenario.
func everyFaultOf(name string) func(scenario string, f fault.Fault) bool {
	return func(scenario string, _ fault.Fault) bool {
		return scenario == name
	}
}

// everyFaultExcept says a defect applies to every fault of one
// scenario except the named ones.
func everyFaultExcept(name string, except ...string) func(scenario string, f fault.Fault) bool {
	return func(scenario string, f fault.Fault) bool {
		return scenario == name && !slices.Contains(except, f.Name())
	}
}

// faultTargeting says a defect applies to every fault of one scenario
// that targets one of the named services.
func faultTargeting(scenario string, services ...string) func(string, fault.Fault) bool {
	return func(name string, f fault.Fault) bool {
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

// faultTargetingExcept says a defect applies to every fault of one
// scenario that targets one of the named services, except the faults
// the workload gives an exchange of their own to fire on.
func faultTargetingExcept(scenario string, except []string, services ...string) func(string, fault.Fault) bool {
	targeting := faultTargeting(scenario, services...)
	return func(name string, f fault.Fault) bool {
		return targeting(name, f) && !slices.Contains(except, f.Name())
	}
}

// faultsNamed says a defect applies to the named faults of one
// scenario.
func faultsNamed(scenario string, names ...string) func(string, fault.Fault) bool {
	return func(name string, f fault.Fault) bool {
		return name == scenario && slices.Contains(names, f.Name())
	}
}

// faultsNamedIn says a defect applies to the named faults of any of
// the scenarios.
func faultsNamedIn(scenarios []string, names ...string) func(string, fault.Fault) bool {
	return func(name string, f fault.Fault) bool {
		return slices.Contains(scenarios, name) && slices.Contains(names, f.Name())
	}
}

// KnownDefects returns the known-defect table of the §6.7 suite.
func KnownDefects() []SuiteDefect {
	return slices.Clone(knownDefects)
}
