// Package message names the messages the client sends to an OPC UA
// server, from the UACP HEL handshake through the secure-channel
// messages to the service requests, for the failure matrix to key its
// faults on. It imports nothing from tests/spec so the fault
// catalogue and the harness can both use it without a cycle.
package message

import "fmt"

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

// String returns the constant's name, "ActivateSession" or "HEL" for
// example.
func (m Message) String() string {
	if m < 0 || int(m) >= len(names) {
		return fmt.Sprintf("Message(%d)", int(m))
	}
	return names[m]
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
