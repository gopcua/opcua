package matrix_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/tests/spec/matrix"
	"github.com/gopcua/opcua/tests/spec/matrix/suites"
)

// The triage's case lists, restated: every row says which issue labels
// one (scenario, check) triple carries for exactly its listed faults —
// or, with no fault list, for every applicable fault of its scenario.
// The unit test asserts the known-defect table labels every listed
// case and nothing else.
var triageCases = []struct {
	issue    string
	check    string
	scenario string
	faults   []string
}{
	{"issue-879", "RepublishesFromNextSequence", "SessionSurvives", nil},
	{"issue-879", "SendsNoPublishBeforeNotAvailable", "SessionSurvives", nil},
	{"issue-879", "KeepsSubscriptionID", "SessionSurvives", nil},
	{"issue-879", "SendsNoTransferForOwnSubscription", "SessionSurvives", nil},
	{"issue-879", "DeliverEachValueOnce", "SessionSurvives", nil},
	{"issue-879", "ResumePublishing", "SessionSurvives", nil},
	{"issue-879", "RecreatesAfterRefusal", "SubscriptionsLost", nil},
	{"issue-879", "RepublishesRecreatedFromOne", "SubscriptionsLost", nil},
	{"issue-895", "ResumePublishing", "SubscriptionsLost", nil},
	{"issue-895", "KeepOneSubscriptionPerClientSubscription", "SubscriptionsLost", nil},
	{"issue-895", "DeliverEachValueOnce", "SubscriptionsLost", nil},
	{"issue-879", "HaveFired", "SessionSurvives", []string{
		"CutAfterResponse/Publish", "CutAfterResponse/Republish",
		"DelayAboveTimeout/Publish", "DelayAboveTimeout/Republish",
		"DelayBelowTimeout/Publish", "DelayBelowTimeout/Republish",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations",
		"Overload/Publish/Bad_TooManyPublishRequests", "Overload/Republish/Bad_ResourceUnavailable",
		"Overload/Republish/Bad_TooManyOperations", "RequestLost/Publish", "RequestLost/Republish",
		"ResponseLost/Publish", "ResponseLost/Republish"}},
	{"issue-879", "HaveFired", "SubscriptionsLost", []string{
		"CutAfterResponse/CreateMonitoredItems", "CutAfterResponse/CreateSubscription", "CutAfterResponse/Publish", "CutAfterResponse/Republish",
		"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription", "DelayAboveTimeout/Publish", "DelayAboveTimeout/Republish",
		"DelayBelowTimeout/CreateMonitoredItems", "DelayBelowTimeout/CreateSubscription", "DelayBelowTimeout/Publish", "DelayBelowTimeout/Republish",
		"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
		"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations", "Overload/Publish/Bad_TooManyPublishRequests",
		"Overload/Republish/Bad_ResourceUnavailable", "Overload/Republish/Bad_TooManyOperations",
		"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/Publish", "RequestLost/Republish",
		"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/Publish", "ResponseLost/Republish"}},
	{"issue-879", "ResumePublishing", "SessionLost", []string{
		"CutAfterResponse/Publish", "DelayAboveTimeout/Read",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations",
		"Overload/Publish/Bad_TooManyPublishRequests", "RequestLost/Publish", "RequestLost/Read",
		"ResponseLost/Publish", "ResponseLost/Read"}},
	{"issue-879", "DeliverEachValueOnce", "SessionLost", []string{"ResponseLost/Publish"}},
	{"issue-879", "KeepOneSubscriptionPerClientSubscription", "SessionLost", []string{
		"DelayAboveTimeout/Read", "RequestLost/Read", "ResponseLost/Read"}},
	{"issue-879", "RecreatesAfterRefusal", "SessionLost", []string{"RequestLost/Read", "ResponseLost/Read"}},
	{"a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it",
		"CloseEveryKnownSession", "SessionLost", []string{
			"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
			"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
			"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
			"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
			"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
			"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions"}},
	{"a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it",
		"KeepOneSessionOpen", "SessionLost", []string{
			"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
			"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
			"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
			"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
			"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
			"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions"}},
	{"a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it",
		"RecreatesAfterRefusal", "SessionLost", []string{"CutAfterResponse/CreateSubscription"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CloseEveryKnownSession", "SessionSurvives", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CloseEveryKnownSession", "SubscriptionsLost", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CloseEveryKnownSession", "SessionLost", []string{"CutAfterResponse/CreateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"KeepOneSessionOpen", "SessionSurvives", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"KeepOneSessionOpen", "SubscriptionsLost", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"KeepOneSessionOpen", "SessionLost", []string{"CutAfterResponse/CreateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CreatesNoSession", "SessionSurvives", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"ReactivatesSession", "SessionSurvives", []string{"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession"}},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CreatesSessionOnlyAfterActivateFailed", "SessionLost", []string{"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession"}},
	{"issue-919", "CreatesSessionOnlyAfterActivateFailed", "SessionLost", []string{"Link/HELUnanswered"}},
	{"issue-919", "KeepOneSessionOpen", "SessionLost", []string{"Link/HELUnanswered"}},
	{"issue-919", "KeepOneSubscriptionPerClientSubscription", "SessionLost", []string{"Link/HELUnanswered"}},
	{"issue-919", "RecreatesAfterRefusal", "SessionLost", []string{"Link/HELUnanswered"}},
	{"issue-919", "ResumePublishing", "SessionLost", []string{"Link/HELUnanswered"}},
	{"issue-919", "CloseEveryKnownSession", "SessionSurvives", []string{"Link/HELUnanswered"}},
	{"issue-919", "CloseEveryKnownSession", "SubscriptionsLost", []string{"Link/HELUnanswered"}},
	{"issue-919", "CreatesNoSession", "SessionSurvives", []string{"Link/HELUnanswered"}},
	{"issue-919", "ReactivatesSession", "SessionSurvives", []string{"Link/HELUnanswered"}},
	{"issue-828", "CloseEveryKnownSession", "SessionSurvives", []string{"Link/Stall"}},
	{"issue-828", "CloseEveryKnownSession", "SubscriptionsLost", []string{"Link/Stall"}},
	{"issue-828", "CreatesSessionOnlyAfterActivateFailed", "SessionLost", []string{"Link/Stall"}},
	{"issue-828", "DeliverEachValueOnce", "SessionLost", []string{"Link/Stall"}},
	{"issue-828", "RecreatesAfterRefusal", "SessionLost", []string{"Link/Stall"}},
	{"issue-828", "ResumePublishing", "SessionLost", []string{"Link/Stall"}},
	{"issue-828", "CreatesNoSession", "SessionSurvives", []string{"Link/Stall"}},
	{"issue-828", "ReactivatesSession", "SessionSurvives", []string{"Link/Stall"}},
	{"after an ActivateSession timeout the old session stays open on the server",
		"CloseEveryKnownSession", "SessionSurvives", []string{"DelayAboveTimeout/ActivateSession"}},
	{"after an ActivateSession timeout the old session stays open on the server",
		"CloseEveryKnownSession", "SubscriptionsLost", []string{"DelayAboveTimeout/ActivateSession"}},
	{"after an ActivateSession timeout the old session stays open on the server",
		"KeepOneSessionOpen", "SessionSurvives", []string{"DelayAboveTimeout/ActivateSession"}},
	{"after an ActivateSession timeout the old session stays open on the server",
		"KeepOneSessionOpen", "SubscriptionsLost", []string{"DelayAboveTimeout/ActivateSession"}},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"CloseEveryKnownSession", "SessionLost", []string{"CutAfterResponse/CreateMonitoredItems"}},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"CloseEveryKnownSession", "SessionSurvives", []string{"CutAfterResponse/Read"}},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"CloseEveryKnownSession", "SubscriptionsLost", []string{"CutAfterResponse/Read"}},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"ResumePublishing", "SessionLost", []string{"CutAfterResponse/CreateMonitoredItems"}},
}

// TestKnownDefectsMatchExactlyTheTriageCases asserts the known-defect
// table labels every case the triage lists for its group, and nothing
// else among the applicable cases.
func TestKnownDefectsMatchExactlyTheTriageCases(t *testing.T) {
	labelsOfText := map[string]string{}
	for _, listed := range matrix.UnfiledDefects() {
		label, text, found := strings.Cut(listed, ": ")
		if !found {
			t.Fatalf("an unfiled defect is listed without its label: %q", listed)
		}
		labelsOfText[text] = label
	}
	resolve := func(issue string) string {
		if label, isText := labelsOfText[issue]; isText {
			return label
		}
		return issue
	}

	expected := map[string][]string{}
	for _, row := range triageCases {
		issue := resolve(row.issue)
		for _, f := range faults.AllFaults {
			if row.faults != nil && !slices.Contains(row.faults, f.Name()) {
				continue
			}
			key := row.scenario + "/" + f.Name() + "/" + row.check
			if !slices.Contains(expected[key], issue) {
				expected[key] = append(expected[key], issue)
			}
		}
	}

	if len(expected) == 0 {
		t.Fatalf("the triage's case lists produced no expected labelling, so the test matched nothing")
	}

	cases, err := matrix.Plan([]matrix.Suite{suites.P4_06_07()}, faults.AllFaults, matrix.KnownDefects())
	if err != nil {
		t.Fatalf("Plan over the §6.7 suite returned an error: %v", err)
	}
	seen := 0
	for _, c := range cases {
		if c.Skip != nil {
			continue
		}
		for _, check := range c.Checks {
			key := c.Path[1] + "/" + c.Path[2] + "/" + check.Name
			var actual []string
			for _, label := range check.Labels {
				if strings.HasPrefix(label, "issue-") || strings.HasPrefix(label, "unfiled-") {
					actual = append(actual, label)
				}
			}
			sort.Strings(actual)
			want := append([]string{}, expected[key]...)
			sort.Strings(want)
			if len(want) > 0 {
				seen++
			}
			if strings.Join(actual, ",") != strings.Join(want, ",") {
				t.Errorf("%s carries issues %v, want %v", key, actual, want)
			}
		}
	}
	if seen == 0 {
		t.Fatalf("no applicable case carried any issue, so the test matched nothing")
	}
}
