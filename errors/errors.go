// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
)

// New returns an error that formats as the given text.
// This function just wraps the errors.New from Golang standard errors library.
func New(text string) error {
	return errors.New(text)
}

// ErrTooShortToDecode indicates the length of user input is too short to be decoded.
type ErrTooShortToDecode struct {
	Type interface{}
	Msg  string
}

// Error returns the type of receiver of decoder method and some additional message.
func (e *ErrTooShortToDecode) Error() string {
	return fmt.Sprintf("too short to decode as %T: %s", e.Type, e.Msg)
}

// ErrInvalidLength indicates the value in Length field is invalid.
type ErrInvalidLength struct {
	Type interface{}
	Msg  string
}

// Error returns the type of receiver and some additional message.
func (e *ErrInvalidLength) Error() string {
	return fmt.Sprintf("got invalid Length in %T: %s", e.Type, e.Msg)
}

// ErrUnsupported indicates the value in Version field is invalid.
type ErrUnsupported struct {
	Type interface{}
	Msg  string
}

// Error returns the type of receiver and some additional message.
func (e *ErrUnsupported) Error() string {
	return fmt.Sprintf("unsupported %T: %s", e.Type, e.Msg)
}

// ErrInvalidType indicates the value in Type/Code field is invalid.
type ErrInvalidType struct {
	Type   interface{}
	Action string
	Msg    string
}

// Error returns the type of receiver and some additional message.
func (e *ErrInvalidType) Error() string {
	return fmt.Sprintf("cannot %s as %T: %s", e.Action, e.Type, e.Msg)
}

// ErrReceiverNil indicates the receiver is nil.
type ErrReceiverNil struct {
	Type interface{}
}

// Error returns the type of receiver.
func (e *ErrReceiverNil) Error() string {
	return fmt.Sprintf("Receiver %T is nil", e.Type)
}
