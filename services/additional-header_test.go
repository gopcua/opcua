// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestAdditionalHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "no-body",
			Struct: NewAdditionalHeader(
				datatypes.NewExpandedNodeID(
					false, false,
					datatypes.NewTwoByteNodeID(255),
					"", 0,
				),
				0x00,
			),
			Bytes: []byte{0x00, 0xff, 0x00},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeAdditionalHeader(b)
	})
}
