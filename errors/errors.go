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
	Type    interface{}
	Message string
}

// NewErrTooShortToDecode creates a ErrTooShortToDecode.
func NewErrTooShortToDecode(decodedType interface{}, msg string) *ErrTooShortToDecode {
	return &ErrTooShortToDecode{
		Type:    decodedType,
		Message: msg,
	}
}

// Error returns the type of receiver of decoder method and some additional message.
func (e *ErrTooShortToDecode) Error() string {
	return fmt.Sprintf("too short to decode as %T: %s", e.Type, e.Message)
}

// ErrInvalidLength indicates the value in Length field is invalid.
type ErrInvalidLength struct {
	Type    interface{}
	Message string
}

// NewErrInvalidLength creates a ErrInvalidLength.
func NewErrInvalidLength(rcvType interface{}, msg string) *ErrInvalidLength {
	return &ErrInvalidLength{
		Type:    rcvType,
		Message: msg,
	}
}

// Error returns the type of receiver and some additional message.
func (e *ErrInvalidLength) Error() string {
	return fmt.Sprintf("got invalid Length in %T: %s", e.Type, e.Message)
}

// ErrUnsupported indicates the value in Version field is invalid.
type ErrUnsupported struct {
	Type    interface{}
	Message string
}

// NewErrUnsupported creates a ErrUnsupported.
func NewErrUnsupported(unsupportedType interface{}, msg string) *ErrUnsupported {
	return &ErrUnsupported{
		Type:    unsupportedType,
		Message: msg,
	}
}

// Error returns the type of receiver and some additional message.
func (e *ErrUnsupported) Error() string {
	return fmt.Sprintf("unsupported %T: %s", e.Type, e.Message)
}

// ErrInvalidType indicates the value in Type/Code field is invalid.
type ErrInvalidType struct {
	Type    interface{}
	Action  string
	Message string
}

// NewErrInvalidType creates a ErrInvalidType.
//
// The parameter action is the action taken when this error is raised(e.g., "decode").
func NewErrInvalidType(invalidType interface{}, action, msg string) *ErrInvalidType {
	return &ErrInvalidType{
		Type:    invalidType,
		Action:  action,
		Message: msg,
	}
}

// Error returns the type of receiver and some additional message.
func (e *ErrInvalidType) Error() string {
	return fmt.Sprintf("cannot %s as %T: %s", e.Action, e.Type, e.Message)
}

// ErrReceiverNil indicates the receiver is nil.
type ErrReceiverNil struct {
	Type interface{}
}

// NewErrReceiverNil creates a ErrReceiverNil.
func NewErrReceiverNil(rcvType interface{}) *ErrReceiverNil {
	return &ErrReceiverNil{
		Type: rcvType,
	}
}

// Error returns the type of receiver.
func (e *ErrReceiverNil) Error() string {
	return fmt.Sprintf("Receiver %T is nil.", e.Type)
}
