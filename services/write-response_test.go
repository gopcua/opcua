// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/wmnsk/gopcua/status"
	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestWriteResponse(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "single-result",
			Struct: NewWriteResponse(
				NewResponseHeader(
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
				),
				nil,
				0,
			),
			Bytes: []byte{
				// TypeID
				0x01, 0x00, 0xa4, 0x02,
				// Timestamp
				0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
				// RequestHandle
				0x01, 0x00, 0x00, 0x00,
				// ServiceResult
				0x00, 0x00, 0x00, 0x00,
				// ServiceDiagnostics
				0x00,
				// StringTable
				0x00, 0x00, 0x00, 0x00,
				// AdditionalHeader
				0x00, 0x00, 0x00,
				// Results
				0x01, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				// DiagnosticInfos
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			Name: "multiple-results",
			Struct: NewWriteResponse(
				NewResponseHeader(
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
				),
				nil,
				0, status.BadUserAccessDenied,
			),
			Bytes: []byte{
				// TypeID
				0x01, 0x00, 0xa4, 0x02,
				// Timestamp
				0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
				// RequestHandle
				0x01, 0x00, 0x00, 0x00,
				// ServiceResult
				0x00, 0x00, 0x00, 0x00,
				// ServiceDiagnostics
				0x00,
				// StringTable
				0x00, 0x00, 0x00, 0x00,
				// AdditionalHeader
				0x00, 0x00, 0x00,
				// Results
				0x02, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x1f, 0x80,
				// DiagnosticInfos
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		v, err := DecodeWriteResponse(b)
		if err != nil {
			return nil, err
		}
		v.Payload = nil
		return v, nil
	})

	t.Run("service-id", func(t *testing.T) {
		id := new(WriteResponse).ServiceType()
		if got, want := id, uint16(ServiceTypeWriteResponse); got != want {
			t.Fatalf("got %d want %d", got, want)
		}
	})
}
