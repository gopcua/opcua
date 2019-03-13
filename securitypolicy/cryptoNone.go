// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

// None functions: return zero values
func blockSizeNone() int {
	return 1
}

func minPaddingNone() int {
	return 0
}

func encryptNone(src []byte) ([]byte, error) {
	var b []byte
	return append(b, src...), nil
}

func decryptNone(src []byte) ([]byte, error) {
	var b []byte
	return append(b, src...), nil
}

func signatureNone([]byte) ([]byte, error) {
	return make([]byte, 0), nil
}

func verifySignatureNone(_, _ []byte) error {
	return nil
}
