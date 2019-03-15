// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/utils/codectest"
)

func TestAdditionalHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "no-body",
			Struct: NewAdditionalHeader(
				ua.NewExpandedNodeID(
					false, false,
					ua.NewTwoByteNodeID(255),
					"", 0,
				),
				0x00,
			),
			Bytes: []byte{0x00, 0xff, 0x00},
		},
	}
	codectest.Run(t, cases)
}
