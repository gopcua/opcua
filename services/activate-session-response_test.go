// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeActivateSessionResponse(t *testing.T) {
	cases := []struct {
		input []byte
		want  *ActivateSessionResponse
	}{
		{ // Without dummy nonce, results nor diags
			[]byte{
				// TypeID
				0x01, 0x00, 0xd6, 0x01,
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
				// ServerNonce
				0xff, 0xff, 0xff, 0xff,
				// Results
				0x00, 0x00, 0x00, 0x00,
				// DiagnosticInfos
				0x00, 0x00, 0x00, 0x00,
			},
			NewActivateSessionResponse(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, nil, nil, nil, nil, nil,
			),
		},
		{ // With dummy nonce, no results and diags
			[]byte{
				// TypeID
				0x01, 0x00, 0xd6, 0x01,
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
				// ServerNonce
				0x04, 0x00, 0x00, 0x00,
				0xde, 0xad, 0xbe, 0xef,
				// Results
				0x00, 0x00, 0x00, 0x00,
				// DiagnosticInfos
				0x00, 0x00, 0x00, 0x00,
			},
			NewActivateSessionResponse(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, nil, nil, []byte{0xde, 0xad, 0xbe, 0xef}, nil, nil,
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeActivateSessionResponse(c.input)
		if err != nil {
			t.Fatal(err)
		}

		// need some manipulation here.
		got.Payload = nil
		c.want.TypeID.NamespaceURI = nil
		c.want.ResponseHeader.AdditionalHeader.TypeID.NamespaceURI = nil
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeActivateSessionResponse(t *testing.T) {
	cases := []struct {
		input *ActivateSessionResponse
		want  []byte
	}{
		{ // Without dummy nonce, results nor diags
			NewActivateSessionResponse(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, nil, nil, nil, nil, nil,
			),
			[]byte{
				// TypeID
				0x01, 0x00, 0xd6, 0x01,
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
				// ServerNonce
				0xff, 0xff, 0xff, 0xff,
				// Results
				0x00, 0x00, 0x00, 0x00,
				// DiagnosticInfos
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{ // With dummy nonce, no results and diags
			NewActivateSessionResponse(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, nil, nil, []byte{0xde, 0xad, 0xbe, 0xef}, nil, nil,
			),
			[]byte{
				// TypeID
				0x01, 0x00, 0xd6, 0x01,
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
				// ServerNonce
				0x04, 0x00, 0x00, 0x00,
				0xde, 0xad, 0xbe, 0xef,
				// Results
				0x00, 0x00, 0x00, 0x00,
				// DiagnosticInfos
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}

	for i, c := range cases {
		got, err := c.input.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestActivateSessionResponseLen(t *testing.T) {
	cases := []struct {
		input *ActivateSessionResponse
		want  int
	}{
		{ // Without dummy nonce, results nor diags
			NewActivateSessionResponse(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, nil, nil, nil, nil, nil,
			),
			40,
		},
		{ // With dummy nonce, no results and diags
			NewActivateSessionResponse(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, nil, nil, []byte{0xde, 0xad, 0xbe, 0xef}, nil, nil,
			),
			44,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
