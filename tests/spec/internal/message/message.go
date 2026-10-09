// Package message names the messages the client sends to an OPC UA
// server, from the UACP HEL handshake through the secure-channel
// messages to the service requests, for the failure matrix to key its
// faults on. It imports nothing from tests/spec so the fault
// catalogue and the harness can both use it without a cycle.
package message

import (
	"fmt"
	"sync"
)

// Message names a message the client sends to a server. The zero
// value is not a valid message; it only anchors the constant block,
// so every member below it has a non-zero value.
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

var names = [...]string{
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

// String returns the message's name, "ActivateSession" or "HEL" for
// example.
func (m Message) String() string {
	if m >= 0 && int(m) < len(names) {
		return names[m]
	}
	if name, ok := addedName(m); ok {
		return name
	}
	return fmt.Sprintf("Message(%d)", int(m))
}

// added holds the names Of gave a Message beyond the constants. The
// Message for added[i] is len(names)+i.
var (
	addedMu sync.Mutex
	added   []string
)

// Of returns the Message named name. For a constant's name it returns
// that constant. For any other name it returns a Message made on first
// use and the same one afterwards, so a fault can target a service
// that has no constant.
func Of(name string) Message {
	for _, m := range Members() {
		if names[m] == name {
			return m
		}
	}
	addedMu.Lock()
	defer addedMu.Unlock()
	for i, a := range added {
		if a == name {
			return Message(len(names) + i)
		}
	}
	added = append(added, name)
	return Message(len(names) + len(added) - 1)
}

// Named says whether m is a constant other than the zero value, or a
// Message that Of returned.
func (m Message) Named() bool {
	if m > messageInvalid && int(m) < len(names) {
		return true
	}
	_, ok := addedName(m)
	return ok
}

func addedName(m Message) (string, bool) {
	i := int(m) - len(names)
	addedMu.Lock()
	defer addedMu.Unlock()
	if i < 0 || i >= len(added) {
		return "", false
	}
	return added[i], true
}

// Members returns every valid Message, in declaration order, without
// the invalid zero value.
func Members() []Message {
	members := make([]Message, 0, len(names)-1)
	for m := HEL; int(m) < len(names); m++ {
		members = append(members, m)
	}
	return members
}
