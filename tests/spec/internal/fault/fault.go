package fault

import (
	"fmt"
	"slices"

	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/ua"
)

// Reason explains why a fault cannot occur in a given run.
type Reason struct{ Text string }

// A Fault is one fault the failure matrix can arm. Name is the case name,
// "ResponseLost/ActivateSession" for example. Available reports nil when the
// fault can occur in a run that sends the given messages, and a reason why
// it cannot otherwise.
type Fault interface {
	Name() string
	Available(sends []message.Message) *Reason
	Options() []harness.Option
	Inject(env *harness.Environment) *harness.Injected
}

type messageKind int

const (
	messageKindInvalid messageKind = iota
	requestLost
	responseLost
	cutAfterResponse
	delayBelowTimeout
	delayAboveTimeout
)

var messageKindNames = [...]string{
	messageKindInvalid: "messageKindInvalid",
	requestLost:        "RequestLost",
	responseLost:       "ResponseLost",
	cutAfterResponse:   "CutAfterResponse",
	delayBelowTimeout:  "DelayBelowTimeout",
	delayAboveTimeout:  "DelayAboveTimeout",
}

func (k messageKind) String() string {
	if k < 0 || int(k) >= len(messageKindNames) {
		return fmt.Sprintf("messageKind(%d)", int(k))
	}
	return messageKindNames[k]
}

type overloadStatus int

const (
	overloadStatusInvalid overloadStatus = iota
	badTooManyOperations
	badResourceUnavailable
	badTooManyPublishRequests
)

var overloadStatusNames = [...]string{
	overloadStatusInvalid:     "overloadStatusInvalid",
	badTooManyOperations:      "Bad_TooManyOperations",
	badResourceUnavailable:    "Bad_ResourceUnavailable",
	badTooManyPublishRequests: "Bad_TooManyPublishRequests",
}

func (s overloadStatus) StatusCode() ua.StatusCode {
	switch s {
	case badTooManyOperations:
		return ua.StatusBadTooManyOperations
	case badResourceUnavailable:
		return ua.StatusBadResourceUnavailable
	case badTooManyPublishRequests:
		return ua.StatusBadTooManyPublishRequests
	}
	return 0
}

func (s overloadStatus) String() string {
	if s < 0 || int(s) >= len(overloadStatusNames) {
		return fmt.Sprintf("overloadStatus(%d)", int(s))
	}
	return overloadStatusNames[s]
}

type linkKind int

const (
	linkKindInvalid linkKind = iota
	closedOnAccept
	listenerClosed
	stall
	helUnanswered
)

var linkKindNames = [...]string{
	linkKindInvalid: "linkKindInvalid",
	closedOnAccept:  "ClosedOnAccept",
	listenerClosed:  "ListenerClosed",
	stall:           "Stall",
	helUnanswered:   "HELUnanswered",
}

func (k linkKind) String() string {
	if k < 0 || int(k) >= len(linkKindNames) {
		return fmt.Sprintf("linkKind(%d)", int(k))
	}
	return linkKindNames[k]
}

type serverKind int

const (
	serverKindInvalid serverKind = iota
	pause
	duplicateSequence
	skippedSequence
)

var serverKindNames = [...]string{
	serverKindInvalid: "serverKindInvalid",
	pause:             "Pause",
	duplicateSequence: "DuplicateSequence",
	skippedSequence:   "SkippedSequence",
}

func (k serverKind) String() string {
	if k < 0 || int(k) >= len(serverKindNames) {
		return fmt.Sprintf("serverKind(%d)", int(k))
	}
	return serverKindNames[k]
}

type messageFault struct {
	kind messageKind
	msg  message.Message
}

func (f messageFault) Name() string {
	return f.kind.String() + "/" + f.msg.String()
}

func (f messageFault) Available(sends []message.Message) *Reason {
	if !slices.Contains(sends, f.msg) {
		return &Reason{Text: f.msg.String() + " is not sent"}
	}
	return nil
}

type overloadFault struct {
	status  overloadStatus
	service message.Message
}

func (f overloadFault) Name() string {
	return "Overload/" + f.service.String() + "/" + f.status.String()
}

func (f overloadFault) Available(sends []message.Message) *Reason {
	if !slices.Contains(sends, f.service) {
		return &Reason{Text: f.service.String() + " is not sent"}
	}
	return nil
}

type linkFault struct{ kind linkKind }

func (f linkFault) Name() string { return "Link/" + f.kind.String() }

func (f linkFault) Available(sends []message.Message) *Reason {
	switch f.kind {
	case closedOnAccept, listenerClosed, helUnanswered:
		if !slices.Contains(sends, message.HEL) {
			return &Reason{Text: "no reconnect: HEL is not sent"}
		}
	}
	return nil
}

type serverFault struct{ kind serverKind }

func (f serverFault) Name() string { return "Server/" + f.kind.String() }

func (f serverFault) Available(sends []message.Message) *Reason {
	switch f.kind {
	case duplicateSequence, skippedSequence:
		if !slices.Contains(sends, message.Publish) {
			return &Reason{Text: "Publish is not sent"}
		}
	}
	return nil
}

type consumerFault struct{}

func (consumerFault) Name() string                              { return "Consumer/Slow" }
func (consumerFault) Available(sends []message.Message) *Reason { return nil }

// controlFault arms nothing: its case runs the scenario without any
// fault, so the checks measure what the client does to a plain
// transport loss.
type controlFault struct{}

func (controlFault) Name() string                              { return "Control/None" }
func (controlFault) Available(sends []message.Message) *Reason { return nil }
func (controlFault) Options() []harness.Option                 { return nil }
func (controlFault) Inject(env *harness.Environment) *harness.Injected {
	return harness.NewInjected(func() bool { return true })
}

// AllFaults is the full fault catalogue, built once at package init. The
// matrix enumerates it in full; it is never sampled.
var AllFaults = buildAllFaults()

func buildAllFaults() []Fault {
	var faults []Fault
	for _, k := range []messageKind{requestLost, responseLost, cutAfterResponse, delayBelowTimeout, delayAboveTimeout} {
		for _, m := range message.Members() {
			if k == delayAboveTimeout && m == message.HEL {
				continue
			}
			if k != requestLost && m == message.CloseSecureChannel {
				continue
			}
			faults = append(faults, messageFault{kind: k, msg: m})
		}
	}
	for _, s := range []overloadStatus{badTooManyOperations, badResourceUnavailable} {
		for _, svc := range overloadedServices() {
			faults = append(faults, overloadFault{status: s, service: svc})
		}
	}
	faults = append(faults, overloadFault{status: badTooManyPublishRequests, service: message.Publish})
	for _, k := range []linkKind{closedOnAccept, listenerClosed, stall, helUnanswered} {
		faults = append(faults, linkFault{kind: k})
	}
	for _, k := range []serverKind{pause, duplicateSequence, skippedSequence} {
		faults = append(faults, serverFault{kind: k})
	}
	faults = append(faults, consumerFault{})
	faults = append(faults, controlFault{})
	return faults
}

// overloadedServices returns the services the scripted server answers
// itself, the only ones an overload fault can answer: the in-tree
// server keeps the session services and Read unexported, so the
// harness cannot answer those without changing server code.
func overloadedServices() []message.Message {
	return []message.Message{
		message.CreateSubscription,
		message.CreateMonitoredItems,
		message.DeleteSubscriptions,
		message.Publish,
		message.Republish,
		message.TransferSubscriptions,
	}
}
