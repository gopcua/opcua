package faults

import (
	"fmt"
	"slices"
)

// Message names a message the client sends to a server, from the UACP HEL
// handshake through the secure-channel messages to the OPC UA service
// requests. The zero value is not a valid message; it only anchors the
// constant block, so every member below it has a non-zero value.
type Message int

const (
	messageInvalid Message = iota
	HEL
	OpenSecureChannel
	CloseSecureChannel
	CreateSession
	ActivateSession
	CloseSession
	Read
	CreateSubscription
	CreateMonitoredItems
	DeleteSubscriptions
	Publish
	Republish
	TransferSubscriptions
)

var messageNames = [...]string{
	messageInvalid:        "messageInvalid",
	HEL:                   "HEL",
	OpenSecureChannel:     "OpenSecureChannel",
	CloseSecureChannel:    "CloseSecureChannel",
	CreateSession:         "CreateSession",
	ActivateSession:       "ActivateSession",
	CloseSession:          "CloseSession",
	Read:                  "Read",
	CreateSubscription:    "CreateSubscription",
	CreateMonitoredItems:  "CreateMonitoredItems",
	DeleteSubscriptions:   "DeleteSubscriptions",
	Publish:               "Publish",
	Republish:             "Republish",
	TransferSubscriptions: "TransferSubscriptions",
}

// String returns the constant's name, "ActivateSession" or "HEL" for
// example.
func (m Message) String() string {
	if m < 0 || int(m) >= len(messageNames) {
		return fmt.Sprintf("Message(%d)", int(m))
	}
	return messageNames[m]
}

// Reason explains why a fault cannot occur in a given run.
type Reason struct{ Text string }

// A Fault is one fault the failure matrix can arm. Name is the case name,
// "ResponseLost/ActivateSession" for example. Available reports nil when the
// fault can occur in a run that sends the given messages, and a reason why
// it cannot otherwise.
type Fault interface {
	Name() string
	Available(sends []Message) *Reason
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
	msg  Message
}

func (f messageFault) Name() string {
	return f.kind.String() + "/" + f.msg.String()
}

func (f messageFault) Available(sends []Message) *Reason {
	if !slices.Contains(sends, f.msg) {
		return &Reason{Text: f.msg.String() + " is not sent"}
	}
	return nil
}

type overloadFault struct {
	status  overloadStatus
	service Message
}

func (f overloadFault) Name() string {
	return "Overload/" + f.service.String() + "/" + f.status.String()
}

func (f overloadFault) Available(sends []Message) *Reason {
	if !slices.Contains(sends, f.service) {
		return &Reason{Text: f.service.String() + " is not sent"}
	}
	return nil
}

type linkFault struct{ kind linkKind }

func (f linkFault) Name() string { return "Link/" + f.kind.String() }

func (f linkFault) Available(sends []Message) *Reason {
	switch f.kind {
	case closedOnAccept, listenerClosed, helUnanswered:
		if !slices.Contains(sends, HEL) {
			return &Reason{Text: "no reconnect: HEL is not sent"}
		}
	}
	return nil
}

type serverFault struct{ kind serverKind }

func (f serverFault) Name() string { return "Server/" + f.kind.String() }

func (f serverFault) Available(sends []Message) *Reason {
	switch f.kind {
	case duplicateSequence, skippedSequence:
		if !slices.Contains(sends, Publish) {
			return &Reason{Text: "Publish is not sent"}
		}
	}
	return nil
}

type consumerFault struct{}

func (consumerFault) Name() string                      { return "Consumer/Slow" }
func (consumerFault) Available(sends []Message) *Reason { return nil }

// AllFaults is the full fault catalogue, built once at package init. The
// matrix enumerates it in full; it is never sampled.
var AllFaults = buildAllFaults()

func buildAllFaults() []Fault {
	var faults []Fault
	for _, k := range []messageKind{requestLost, responseLost, cutAfterResponse, delayBelowTimeout, delayAboveTimeout} {
		for m := HEL; int(m) < len(messageNames); m++ {
			if k == delayAboveTimeout && m == HEL {
				continue
			}
			if k != requestLost && m == CloseSecureChannel {
				continue
			}
			faults = append(faults, messageFault{kind: k, msg: m})
		}
	}
	for _, s := range []overloadStatus{badTooManyOperations, badResourceUnavailable} {
		for _, svc := range serviceMessages() {
			faults = append(faults, overloadFault{status: s, service: svc})
		}
	}
	faults = append(faults, overloadFault{status: badTooManyPublishRequests, service: Publish})
	for _, k := range []linkKind{closedOnAccept, listenerClosed, stall, helUnanswered} {
		faults = append(faults, linkFault{kind: k})
	}
	for _, k := range []serverKind{pause, duplicateSequence, skippedSequence} {
		faults = append(faults, serverFault{kind: k})
	}
	faults = append(faults, consumerFault{})
	return faults
}

func serviceMessages() []Message {
	var services []Message
	for m := HEL; int(m) < len(messageNames); m++ {
		if m != HEL && m != OpenSecureChannel && m != CloseSecureChannel {
			services = append(services, m)
		}
	}
	return services
}
