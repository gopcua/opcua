// Copyright 2018-2019 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/sha1"
)

// Thumbprint returns the thumbprint of a DER-encoded certificate
func Thumbprint(c []byte) []byte {
	thumbprint := sha1.Sum(c)

	return thumbprint[:]
}
