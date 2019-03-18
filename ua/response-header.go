// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"time"
)

// ResponseHeader represents a Response Header in each services.
//
// Specification: Part 4, 7.29
// type ResponseHeader struct {
// 	Timestamp          time.Time
// 	RequestHandle      uint32
// 	ServiceResult      uint32
// 	ServiceDiagnostics *DiagnosticInfo
// 	StringTable        []string
// 	AdditionalHeader   *AdditionalHeader
// }

// NewResponseHeader creates a new ResponseHeader.
func NewResponseHeader(timestamp time.Time, handle uint32, code StatusCode, diag *DiagnosticInfo, strs []string, additionalHeader *ExtensionObject) *ResponseHeader {
	r := &ResponseHeader{
		Timestamp:          timestamp,
		RequestHandle:      handle,
		ServiceResult:      code,
		ServiceDiagnostics: diag,
		StringTable:        strs,
		AdditionalHeader:   additionalHeader,
	}
	if diag == nil {
		r.ServiceDiagnostics = NewNullDiagnosticInfo()
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
