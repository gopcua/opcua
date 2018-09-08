// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeHello(t *testing.T) {
	cases := []struct {
		input []byte
		want  *Hello
	}{
		{ // Normal Hello
			[]byte{ // Hello message
				// MessageType: HEL
				0x48, 0x45, 0x4c,
				// Chunk Type: F
				0x46,
				// MessageSize: 70
				0x46, 0x00, 0x00, 0x00,
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
				// EndPointURL
				0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
				0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
				0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
				0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
				0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
				0x65, 0x72,
			},
			NewHello(
				0,                                        //Version
				65280,                                    // ReceiveBufSize
				65535,                                    // SendBufSize
				4000,                                     // MaxMessageSize
				"opc.tcp://wow.its.easy:11111/UA/Server", // EndPointURL
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeHello(c.input)
		if err != nil {
			t.Fatal(err)
		}

		got.Payload = nil
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeHello(t *testing.T) {
	cases := []struct {
		input *Hello
		want  []byte
	}{
		{ // Normal Hello
			NewHello(
				0,                                        //Version
				65280,                                    // ReceiveBufSize
				65535,                                    // SendBufSize
				4000,                                     // MaxMessageSize
				"opc.tcp://wow.its.easy:11111/UA/Server", // EndPointURL
			),
			[]byte{ // Hello message
				// MessageType: HEL
				0x48, 0x45, 0x4c,
				// Chunk Type: F
				0x46,
				// MessageSize: 70
				0x46, 0x00, 0x00, 0x00,
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
				// EndPointURL
				0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
				0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
				0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
				0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
				0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
				0x65, 0x72,
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

func TestHelloLen(t *testing.T) {
	cases := []struct {
		input *Hello
		want  int
	}{
		{ // Normal Hello
			NewHello(
				0,                                        //Version
				65280,                                    // ReceiveBufSize
				65535,                                    // SendBufSize
				4000,                                     // MaxMessageSize
				"opc.tcp://wow.its.easy:11111/UA/Server", // EndPointURL
			),
			70,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
