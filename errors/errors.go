// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Cause Cause returns the underlying cause of the error, if possible.
// An error value has a cause if it implements the following interface:
//
// This function just wraps the Cause in pkg/errors library.
func Cause(err error) error {
	return errors.Cause(err)
}

// Errorf formats according to a format specifier and returns the string as a value
// that satisfies error. Errorf also records the stack trace at the point it was called.
//
// This function just wraps the Errorf in pkg/errors library.
func Errorf(format string, args ...interface{}) error {
	return errors.Errorf(format, args...)
}

// New returns an error that formats as the given text.
//
// This function just wraps the New in pkg/errors library.
func New(text string) error {
	return errors.New(text)
}

// WithMessage annotates err with a new message. If err is nil, WithMessage returns nil.
//
// This function just wraps the WithMessage in pkg/errors library.
func WithMessage(err error, message string) error {
	return errors.WithMessage(err, message)
}

// WithStack annotates err with a stack trace at the point WithStack was called.
// If err is nil, WithStack returns nil.
//
// This function just wraps the WithStack in pkg/errors library.
func WithStack(err error) error {
	return errors.WithStack(err)
}

// Wrap returns an error annotating err with a stack trace at the point Wrap is called,
// and the supplied message. If err is nil, Wrap returns nil.
//
// This function just wraps the Wrap in pkg/errors library.
func Wrap(err error, message string) error {
	return errors.Wrap(err, message)
}

// Wrapf returns an error annotating err with a stack trace at the point Wrapf is called,
// and the format specifier. If err is nil, Wrapf returns nil.
//
// This function just wraps the Wrapf in pkg/errors library.
func Wrapf(err error, format string, args ...interface{}) error {
	return errors.Wrapf(err, format, args...)
}

// Frame represents a program counter inside a stack frame.
//
// This type is just an alias of Frame in pkg/errors library.
type Frame = errors.Frame

// StackTrace is stack of Frames from innermost (newest) to outermost (oldest).
//
// This type is just an alias of Frame in pkg/errors library.
type StackTrace = errors.StackTrace

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
