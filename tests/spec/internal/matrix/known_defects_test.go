package matrix_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/matrix/suites"
)

// The triage's case lists, restated: every row says which issue labels
// one (scenario, check) triple carries for exactly its listed faults —
// or, with no fault list, for every applicable fault of its scenario
// except those in the exception list. The unit test asserts the
// known-defect table labels every listed case and nothing else.

// recreatePathFaults lists the faults whose labelled checks passed in
// every one of runs 7, 8 and 9, so no entry covers them.
var recreatePathFaults = []string{
	"CutAfterResponse/OpenSecureChannel", "DelayAboveTimeout/ActivateSession",
	"RequestLost/ActivateSession", "ResponseLost/ActivateSession"}

// cutPublishFaults lists the faults that cut the connection on the
// Publish service: the arm exchange answers one held Publish right
// after the arm, so the cut fires there, and the sentinel exchange the
// reconnect answers never reaches a fault.
var cutPublishFaults = []string{
	"CutAfterResponse/Publish", "RequestLost/Publish", "ResponseLost/Publish"}

var triageCases = []struct {
	issue    string
	check    string
	scenario string
	faults   []string
	except   []string
}{
	{"issue-879", "RepublishesFromNextSequence", "SessionSurvives", nil, nil},
	{"issue-879", "SendsNoPublishBeforeNotAvailable", "SessionSurvives", nil, nil},
	{"issue-879", "KeepsSubscriptionID", "SessionSurvives", nil, nil},
	{"issue-879", "SendsNoTransferForOwnSubscription", "SessionSurvives", nil, nil},
	{"issue-879", "DeliverEachValueOnce", "SessionSurvives", nil, recreatePathFaults},
	{"issue-879", "ResumePublishing", "SessionSurvives", nil, recreatePathFaults},
	{"issue-879", "RecreatesAfterRefusal", "SubscriptionsLost", nil, nil},
	{"issue-879", "RepublishesRecreatedFromOne", "SubscriptionsLost", nil, nil},
	{"issue-895", "ResumePublishing", "SubscriptionsLost", nil, recreatePathFaults},
	{"issue-895", "KeepsPublishingAfterCancelThenSubscribe", "CancelThenSubscribe", nil, nil},
	{"issue-895", "ResumePublishing", "CancelThenSubscribe", nil, nil},
	{"issue-895", "KeepOneSubscriptionPerClientSubscription", "SubscriptionsLost", nil, recreatePathFaults},
	{"issue-879", "HaveFired", "SessionSurvives", []string{
		"CutAfterResponse/Publish", "CutAfterResponse/Republish",
		"DelayAboveTimeout/Publish", "DelayAboveTimeout/Republish",
		"DelayBelowTimeout/Publish", "DelayBelowTimeout/Republish",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations",
		"Overload/Publish/Bad_TooManyPublishRequests", "Overload/Republish/Bad_ResourceUnavailable",
		"Overload/Republish/Bad_TooManyOperations", "RequestLost/Publish", "RequestLost/Republish",
		"ResponseLost/Publish", "ResponseLost/Republish"}, cutPublishFaults},
	{"issue-879", "HaveFired", "SubscriptionsLost", []string{
		"CutAfterResponse/CreateMonitoredItems", "CutAfterResponse/CreateSubscription", "CutAfterResponse/Publish", "CutAfterResponse/Republish",
		"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription", "DelayAboveTimeout/Publish", "DelayAboveTimeout/Republish",
		"DelayBelowTimeout/CreateMonitoredItems", "DelayBelowTimeout/CreateSubscription", "DelayBelowTimeout/Publish", "DelayBelowTimeout/Republish",
		"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
		"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations", "Overload/Publish/Bad_TooManyPublishRequests",
		"Overload/Republish/Bad_ResourceUnavailable", "Overload/Republish/Bad_TooManyOperations",
		"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/Publish", "RequestLost/Republish",
		"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/Publish", "ResponseLost/Republish"}, cutPublishFaults},
	{"issue-879", "ResumePublishing", "SessionLost", []string{
		"CutAfterResponse/Publish", "DelayAboveTimeout/Read",
		"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations",
		"Overload/Publish/Bad_TooManyPublishRequests", "RequestLost/Publish", "RequestLost/Read",
		"ResponseLost/Publish", "ResponseLost/Read"}, cutPublishFaults},
	{"issue-879", "DeliverEachValueOnce", "SessionLost", []string{"ResponseLost/Publish"}, []string{"ResponseLost/Publish"}},
	{"issue-879", "KeepOneSubscriptionPerClientSubscription", "SessionLost", []string{
		"DelayAboveTimeout/Read", "RequestLost/Read", "ResponseLost/Read"}, nil},
	{"issue-879", "RecreatesAfterRefusal", "SessionLost", []string{"RequestLost/Read", "ResponseLost/Read"}, nil},
	{"a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it",
		"CloseEveryKnownSession", "SessionLost", []string{
			"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
			"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
			"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
			"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
			"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
			"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions"}, nil},
	{"a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it",
		"KeepOneSessionOpen", "SessionLost", []string{
			"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
			"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
			"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
			"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
			"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
			"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions"}, nil},
	{"a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it",
		"RecreatesAfterRefusal", "SessionLost", []string{"CutAfterResponse/CreateSubscription"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CloseEveryKnownSession", "SessionSurvives", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CloseEveryKnownSession", "SubscriptionsLost", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CloseEveryKnownSession", "SessionLost", []string{"CutAfterResponse/CreateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"KeepOneSessionOpen", "SessionSurvives", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"KeepOneSessionOpen", "SubscriptionsLost", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"KeepOneSessionOpen", "SessionLost", []string{"CutAfterResponse/CreateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CreatesNoSession", "SessionSurvives", []string{
			"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"ReactivatesSession", "SessionSurvives", []string{"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession"}, nil},
	{"a connection failure during ActivateSession makes the client forget its session without retrying or closing it",
		"CreatesSessionOnlyAfterActivateFailed", "SessionLost", []string{"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "CreatesSessionOnlyAfterActivateFailed", "SessionLost", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "KeepOneSessionOpen", "SessionLost", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "KeepOneSubscriptionPerClientSubscription", "SessionLost", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "RecreatesAfterRefusal", "SessionLost", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "ResumePublishing", "SessionLost", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "CloseEveryKnownSession", "SessionSurvives", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "CloseEveryKnownSession", "SubscriptionsLost", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "CreatesNoSession", "SessionSurvives", []string{"Link/HELUnanswered"}, nil},
	{"the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it", "ReactivatesSession", "SessionSurvives", []string{"Link/HELUnanswered"}, nil},
	{"issue-828", "CreatesSessionOnlyAfterActivateFailed", "SessionLost", []string{"Link/Stall"}, nil},
	{"issue-828", "DeliverEachValueOnce", "SessionLost", []string{"Link/Stall"}, nil},
	{"issue-828", "RecreatesAfterRefusal", "SessionLost", []string{"Link/Stall"}, nil},
	{"issue-828", "ResumePublishing", "SessionLost", []string{"Link/Stall"}, nil},
	{"issue-828", "CreatesNoSession", "SessionSurvives", []string{"Link/Stall"}, nil},
	{"issue-828", "ReactivatesSession", "SessionSurvives", []string{"Link/Stall"}, nil},
	{"after an ActivateSession timeout the old session stays open on the server",
		"CloseEveryKnownSession", "SessionSurvives", []string{"DelayAboveTimeout/ActivateSession"}, nil},
	{"after an ActivateSession timeout the old session stays open on the server",
		"CloseEveryKnownSession", "SubscriptionsLost", []string{"DelayAboveTimeout/ActivateSession"}, nil},
	{"after an ActivateSession timeout the old session stays open on the server",
		"KeepOneSessionOpen", "SessionSurvives", []string{"DelayAboveTimeout/ActivateSession"}, nil},
	{"after an ActivateSession timeout the old session stays open on the server",
		"KeepOneSessionOpen", "SubscriptionsLost", []string{"DelayAboveTimeout/ActivateSession"}, nil},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"CloseEveryKnownSession", "SessionLost", []string{"CutAfterResponse/CreateMonitoredItems"}, nil},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"CloseEveryKnownSession", "SessionSurvives", []string{"CutAfterResponse/Read"}, nil},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"CloseEveryKnownSession", "SubscriptionsLost", []string{"CutAfterResponse/Read"}, nil},
	{"the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel",
		"ResumePublishing", "SessionLost", []string{"CutAfterResponse/CreateMonitoredItems"}, nil},
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
		for _, f := range fault.AllFaults {
			if row.faults != nil && !slices.Contains(row.faults, f.Name()) {
				continue
			}
			if slices.Contains(row.except, f.Name()) {
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

	cases, err := matrix.Plan([]matrix.Suite{suites.P4_06_07(), suites.P4_05_14()}, fault.AllFaults, matrix.KnownDefects())
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
