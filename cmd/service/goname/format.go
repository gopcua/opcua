// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package goname

import "strings"

var (
	idents = strings.NewReplacer(
		"Guid", "GUID",
		"Id", "ID",
		"Json", "JSON",
		"QualityOfService", "QoS",
		"Tcp", "TCP",
		"Uadp", "UADP",
		"Uri", "URI",
		"Url", "URL",
		"Xml", "XML",
	)

	fixes = strings.NewReplacer(
		"IDentity", "Identity",
	)
)

func Format(s string) string {
	return fixes.Replace(idents.Replace(s))
}
