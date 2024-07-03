package attrs

import "github.com/gopcua/opcua/ua"

func BrowseName(name string) *ua.QualifiedName {
	return &ua.QualifiedName{Name: name}
}

func DisplayName(name, locale string) *ua.LocalizedText {
	lt := &ua.LocalizedText{Text: name, Locale: locale}
	lt.UpdateMask()
	return lt
}

func InverseName(name, locale string) *ua.LocalizedText {
	lt := &ua.LocalizedText{Text: name, Locale: locale}
	lt.UpdateMask()
	return lt
}

func NodeClass(n ua.NodeClass) uint32 {
	return uint32(n)
}

func DataType(id *ua.NodeID) *ua.ExpandedNodeID {
	return &ua.ExpandedNodeID{NodeID: id}
}
