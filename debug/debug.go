// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package debug provides functions for debug logging.
package debug

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
)

// Flags contains the debug flags set by OPC_DEBUG.
//
//   - codec : print detailed debugging information when encoding/decoding
var Flags = os.Getenv("OPC_DEBUG")

// FlagSet returns true if the OPC_DEBUG environment variable contains the
// given flag.
func FlagSet(name string) bool {
	return slices.Contains(strings.Fields(Flags), name)
}

// ToJSON returns the JSON representation of v.
func ToJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}
