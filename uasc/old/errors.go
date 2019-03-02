// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import "github.com/wmnsk/gopcua/errors"

// Errors used across UASC.
// XXX - to be integrated in errors package.
var (
	ErrInvalidState = errors.New("invalid state")
	ErrTimeout      = errors.New("timed out")
)

// Errors for SecureChannel handling.
// XXX - to be integrated in errors package.
var (
	ErrUnexpectedMessage       = errors.New("got unexpected message")
	ErrSecureChannelNotOpened  = errors.New("secure channel not opened")
	ErrSecurityModeUnsupported = errors.New("got request with unsupported SecurityMode")
	ErrRejected                = errors.New("rejected by server")
)

// Errors for Session handling.
// XXX - to be integrated in errors package.
var (
	ErrInvalidAuthenticationToken = errors.New("invalid AuthenticationToken")
	ErrSessionNotActivated        = errors.New("session is not activated")
	ErrInvalidSignatureAlgorithm  = errors.New("algorithm in signature doesn't match")
	ErrInvalidSignatureData       = errors.New("signature is invalid")
)
