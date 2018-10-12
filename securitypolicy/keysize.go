// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import "crypto/rsa"

// Because rsa.PublicKey.Size() was only added in Go 1.11
func keySize(pub *rsa.PublicKey) int {
	return (pub.N.BitLen() + 7) / 8
}
