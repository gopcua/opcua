// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeAcknowledge(t *testing.T) {
	cases := []struct {
		input []byte
		want  *Acknowledge
	}{
		{ // Normal Acknowledge
			[]byte{
				// MessageType: ACK
				0x41, 0x43, 0x4b,
				// Chunk Type: F
				0x46,
				// MessageSize: 28
				0x1c, 0x00, 0x00, 0x00,
				// Version: 0
				0x00, 0x00, 0x00, 0x00,
				// ReceiveBufSize: 65280
				0x00, 0xff, 0x00, 0x00,
				// SendBufSize: 65535
				0xff, 0xff, 0x00, 0x00,
				// MaxMessageSize: 4000
				0xa0, 0x0f, 0x00, 0x00,
				// MaxChunkCount: 0
				0x00, 0x00, 0x00, 0x00,
			},
			NewAcknowledge(
				0,     //Version
				65280, // ReceiveBufSize
				65535, // SendBufSize
				4000,  // MaxMessageSize
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeAcknowledge(c.input)
		if err != nil {
			t.Fatal(err)
		}

		got.Payload = nil
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeAcknowledge(t *testing.T) {
	cases := []struct {
		input *Acknowledge
		want  []byte
	}{
		{ // Normal Acknowledge
			NewAcknowledge(
				0,     //Version
				65280, // ReceiveBufSize
				65535, // SendBufSize
				4000,  // MaxMessageSize
			),
			[]byte{ // Acknowledge message
				// MessageType: ACK
				0x41, 0x43, 0x4b,
				// Chunk Type: F
				0x46,
				// MessageSize: 28
				0x1c, 0x00, 0x00, 0x00,
				// Version: 0
				0x00, 0x00, 0x00, 0x00,
				// ReceiveBufSize: 65280
				0x00, 0xff, 0x00, 0x00,
				// SendBufSize: 65535
				0xff, 0xff, 0x00, 0x00,
				// MaxMessageSize: 4000
				0xa0, 0x0f, 0x00, 0x00,
				// MaxChunkCount: 0
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

func TestAcknowledgeLen(t *testing.T) {
	cases := []struct {
		input *Acknowledge
		want  int
	}{
		{ // Normal Acknowledge
			NewAcknowledge(
				0,     //Version
				65280, // ReceiveBufSize
				65535, // SendBufSize
				4000,  // MaxMessageSize
			),
			28,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
