// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"
	"time"

	"github.com/gopcua/opcua/ua"
)

// ResponseHeader represents a Response Header in each services.
//
// Specification: Part 4, 7.29
type ResponseHeader struct {
	Timestamp          time.Time
	RequestHandle      uint32
	ServiceResult      uint32
	ServiceDiagnostics *ua.DiagnosticInfo
	StringTable        []string
	AdditionalHeader   *AdditionalHeader
}

// NewResponseHeader creates a new ResponseHeader.
func NewResponseHeader(timestamp time.Time, handle, code uint32, diag *ua.DiagnosticInfo, strs []string, additionalHeader *AdditionalHeader) *ResponseHeader {
	r := &ResponseHeader{
		Timestamp:          timestamp,
		RequestHandle:      handle,
		ServiceResult:      code,
		ServiceDiagnostics: diag,
		StringTable:        strs,
		AdditionalHeader:   additionalHeader,
	}
	if diag == nil {
		r.ServiceDiagnostics = ua.NewNullDiagnosticInfo()
	}

	return r
}

// String returns ResponseHeader in string.
func (r *ResponseHeader) String() string {
	return fmt.Sprintf("%v, %d, %v, %v, %v, %v",
		r.Timestamp,
		r.RequestHandle,
		r.ServiceResult,
		r.ServiceDiagnostics,
		r.StringTable,
		r.AdditionalHeader,
	)
}
