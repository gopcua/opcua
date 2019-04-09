// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

func flatten(b ...[]byte) []byte {
	var x []byte
	for _, buf := range b {
		x = append(x, buf...)
	}
	return x
}
