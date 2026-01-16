package server

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/gopcua/opcua/schema"
	"github.com/gopcua/opcua/ua"
)

func (srv *Server) ImportNodeSet(nodes *schema.UANodeSet) error {
	err := srv.namespacesImportNodeSet(nodes)
	if err != nil {
		return fmt.Errorf("problem creating namespaces: %w", err)
	}
	err = srv.nodesImportNodeSet(nodes)
	if err != nil {
		return fmt.Errorf("problem creating nodes: %w", err)
	}
	err = srv.refsImportNodeSet(nodes)
	if err != nil {
		return fmt.Errorf("problem creating references: %w", err)
	}
	return nil
}

func (srv *Server) namespacesImportNodeSet(nodes *schema.UANodeSet) error {
	if nodes.NamespaceUris == nil {
		return nil
	}
	for i := range nodes.NamespaceUris.Uri {
		_ = NewNodeNameSpace(srv, nodes.NamespaceUris.Uri[i])
	}
	return nil
}

func (srv *Server) nodesImportNodeSet(nodes *schema.UANodeSet) error {

	log.Printf("New Node Set: %s", nodes.LastModifiedAttr)

	reftypes := make(map[string]*schema.UAReferenceType)

	// the first thing we have to do is go thorugh and define all the nodes.
	// set up the reference types.
	for i := range nodes.UAReferenceType {
		rt := nodes.UAReferenceType[i]
		reftypes[rt.BrowseNameAttr] = rt // sometimes they use browse name
		reftypes[rt.NodeIdAttr] = rt     // sometimes they use node id

		nid := ua.MustParseNodeID(rt.NodeIdAttr)

		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(rt.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: rt.BrowseNameAttr})
		attrs[ua.AttributeIDIsAbstract] = DataValueFromValue(rt.IsAbstractAttr)
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(rt.UserWriteMaskAttr)
		attrs[ua.AttributeIDSymmetric] = DataValueFromValue(rt.SymmetricAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(rt.WriteMaskAttr)
		if len(rt.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(rt.DisplayName[0].Value))
		}
		if len(rt.InverseName) > 0 {
			attrs[ua.AttributeIDInverseName] = DataValueFromValue(ua.NewLocalizedText(rt.InverseName[0].Value))
		} else {
			attrs[ua.AttributeIDInverseName] = DataValueFromValue(ua.NewLocalizedText(""))
		}
		if len(rt.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(rt.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassReferenceType))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)
		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}
		ns.AddNode(n)
	}

	// set up the data types.
	for i := range nodes.UADataType {
		dt := nodes.UADataType[i]
		nid := ua.MustParseNodeID(dt.NodeIdAttr)

		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(dt.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: dt.BrowseNameAttr})
		attrs[ua.AttributeIDIsAbstract] = DataValueFromValue(dt.IsAbstractAttr)
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(dt.UserWriteMaskAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(dt.WriteMaskAttr)
		if len(dt.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(dt.DisplayName[0].Value))
		}
		if len(dt.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(dt.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassDataType))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)

		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}

		ns.AddNode(n)
	}

	// set up the object types
	for i := range nodes.UAObjectType {
		ot := nodes.UAObjectType[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(ot.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: ot.BrowseNameAttr})
		attrs[ua.AttributeIDIsAbstract] = DataValueFromValue(ot.IsAbstractAttr)
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(ot.UserWriteMaskAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(ot.WriteMaskAttr)
		if len(ot.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(ot.DisplayName[0].Value))
		}
		if len(ot.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(ot.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassObjectType))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)
		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}
		ns.AddNode(n)
	}

	// set up the variable Types
	for i := range nodes.UAVariableType {
		ot := nodes.UAVariableType[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(ot.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: ot.BrowseNameAttr})
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(ot.UserWriteMaskAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(ot.WriteMaskAttr)
		if len(ot.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(ot.DisplayName[0].Value))
		}
		if len(ot.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(ot.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassVariableType))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)
		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}
		ns.AddNode(n)
	}

	// set up the variables
	for i := range nodes.UAVariable {
		ot := nodes.UAVariable[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)

		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(ot.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: ot.BrowseNameAttr})
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(ot.UserWriteMaskAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(ot.WriteMaskAttr)
		attrs[ua.AttributeIDHistorizing] = DataValueFromValue(ot.HistorizingAttr)
		attrs[ua.AttributeIDValueRank] = DataValueFromValue(ot.ValueRankAttr)

		dtidx := slices.IndexFunc(nodes.UADataType, func(dt *schema.UADataType) bool {
			if strings.Compare(dt.BrowseNameAttr, ot.DataTypeAttr) == 0 {
				return true
			}

			return strings.Compare(dt.NodeIdAttr, ot.DataTypeAttr) == 0
		})

		if dtidx >= 0 {
			dt := nodes.UADataType[dtidx]
			attrs[ua.AttributeIDDataType] = DataValueFromValue(ua.MustParseNodeID(dt.NodeIdAttr))
		} else {
			aliasidx := slices.IndexFunc(nodes.Aliases.Alias, func(a *schema.NodeIdAlias) bool {
				return strings.Compare(a.AliasAttr, ot.DataTypeAttr) == 0
			})
			if aliasidx >= 0 {
				dt := nodes.Aliases.Alias[aliasidx]
				attrs[ua.AttributeIDDataType] = DataValueFromValue(ua.MustParseNodeID(dt.Value))
			}
		}

		if len(ot.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(ot.DisplayName[0].Value))
		}
		if len(ot.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(ot.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassVariable))

		var valueFunc ValueFunc
		newValueFuncFromData := func(v any) ValueFunc {
			dv := DataValueFromValue(v)
			return func() *ua.DataValue { return dv }
		}

		if ot.Value != nil {
			if ot.Value.StringAttr != nil {
				valueFunc = newValueFuncFromData(ot.Value.StringAttr.Data)
			} else if ot.Value.StringListAttr != nil {
				data := make([]string, 0, len(ot.Value.StringListAttr.Data))
				for _, s := range ot.Value.StringListAttr.Data {
					data = append(data, s.Data)
				}
				valueFunc = newValueFuncFromData(data)
			} else if ot.Value.DateTimeAttr != nil {
				valueFunc = newValueFuncFromData(ot.Value.DateTimeAttr.Data)
			} else if ot.Value.Int32Attr != nil {
				valueFunc = newValueFuncFromData(ot.Value.Int32Attr.Data)
			} else if ot.Value.UInt32Attr != nil {
				valueFunc = newValueFuncFromData(ot.Value.UInt32Attr.Data)
			} else if ot.Value.Int32ListAttr != nil {
				data := make([]int32, 0, len(ot.Value.Int32ListAttr.Data))
				for _, i := range ot.Value.Int32ListAttr.Data {
					data = append(data, i.Data)
				}
				valueFunc = newValueFuncFromData(data)
			} else if ot.Value.BoolAttr != nil {
				valueFunc = newValueFuncFromData(ot.Value.BoolAttr.Data)
			} else if ot.Value.TextAttr != nil {
				v := ua.NewLocalizedTextWithLocale(ot.Value.TextAttr.Text, ot.Value.TextAttr.Locale)
				valueFunc = newValueFuncFromData(v)
			} else if ot.Value.TextListAttr != nil {
				data := make([]*ua.LocalizedText, 0, len(ot.Value.TextListAttr.Data))
				for _, t := range ot.Value.TextListAttr.Data {
					data = append(data, ua.NewLocalizedTextWithLocale(t.Text, t.Locale))
				}
				valueFunc = newValueFuncFromData(data)
			} else if ot.Value.ExtObjAttr != nil {
				extObj := ot.Value.ExtObjAttr
				if extObj.TypeID.Identifier == "i=297" {
					arg := &ua.Argument{
						Name:            extObj.Body.Argument.Name,
						DataType:        ua.MustParseNodeID(extObj.Body.Argument.DataType.Identifier),
						ValueRank:       int32(extObj.Body.Argument.ValueRank),
						ArrayDimensions: make([]uint32, 0, len(extObj.Body.Argument.ArrayDimensions.Data)),
						Description:     ua.NewLocalizedText(extObj.Body.Argument.Description.Text),
					}

					for _, ad := range extObj.Body.Argument.ArrayDimensions.Data {
						arg.ArrayDimensions = append(arg.ArrayDimensions, ad.Data)
					}
					v := ua.NewExtensionObject(arg)
					valueFunc = newValueFuncFromData(v)
				}
			} else if ot.Value.ExtObjListAttr != nil {
				if !slices.ContainsFunc(ot.Value.ExtObjListAttr.Data, func(eo schema.ValueExtensionObject) bool {
					return eo.TypeID.Identifier != "i=297"
				}) {
					data := make([]*ua.ExtensionObject, 0, len(ot.Value.ExtObjListAttr.Data))
					for _, extObj := range ot.Value.ExtObjListAttr.Data {
						arg := &ua.Argument{
							Name:            extObj.Body.Argument.Name,
							DataType:        ua.MustParseNodeID(extObj.Body.Argument.DataType.Identifier),
							ValueRank:       int32(extObj.Body.Argument.ValueRank),
							ArrayDimensions: make([]uint32, 0, len(extObj.Body.Argument.ArrayDimensions.Data)),
							Description:     ua.NewLocalizedText(extObj.Body.Argument.Description.Text),
						}

						for _, ad := range extObj.Body.Argument.ArrayDimensions.Data {
							arg.ArrayDimensions = append(arg.ArrayDimensions, ad.Data)
						}
						data = append(data, ua.NewExtensionObject(arg))
					}
					v, _ := ua.NewVariant(data)
					valueFunc = newValueFuncFromData(v)
				}
			} else if ot.Value.QualifiedNameAttr != nil {
				valueFunc = newValueFuncFromData(&ua.QualifiedName{
					NamespaceIndex: uint16(ot.Value.QualifiedNameAttr.NamespaceIndex),
					Name:           ot.Value.QualifiedNameAttr.Name,
				})
			} else {
				if ot.ReleaseStatusAttr != "Deprecated" {
					srv.cfg.logger.Warn("failed to parse value for datatype " + ot.DataTypeAttr)
				}
			}
		}

		var refs References = make([]*ua.ReferenceDescription, 0)
		n := NewNode(nid, attrs, refs, valueFunc)

		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}

		ns.AddNode(n)
	}

	// set up the methods
	for i := range nodes.UAMethod {
		ot := nodes.UAMethod[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)

		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(ot.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: ot.BrowseNameAttr})
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(ot.UserWriteMaskAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(ot.WriteMaskAttr)

		if len(ot.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(ot.DisplayName[0].Value))
		}
		if len(ot.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(ot.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassMethod))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)

		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}
		ns.AddNode(n)
	}

	// set up the objects
	for i := range nodes.UAObject {
		ot := nodes.UAObject[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		if ot.NodeIdAttr == "i=85" {
			log.Printf("doing objects.")
		}
		var attrs Attributes = make(map[ua.AttributeID]*ua.DataValue)
		attrs[ua.AttributeIDAccessRestrictions] = DataValueFromValue(ot.AccessRestrictionsAttr)
		attrs[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{NamespaceIndex: nid.Namespace(), Name: ot.BrowseNameAttr})
		attrs[ua.AttributeIDUserWriteMask] = DataValueFromValue(ot.UserWriteMaskAttr)
		attrs[ua.AttributeIDWriteMask] = DataValueFromValue(ot.WriteMaskAttr)
		if len(ot.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(ot.DisplayName[0].Value))
		}
		if len(ot.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(ot.Description[0].Value))
		}

		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassObject))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)
		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("Could Not Find Namespace %d", nid.Namespace())
			}
			return err
		}
		ns.AddNode(n)
	}

	return nil
}
func (srv *Server) refsImportNodeSet(nodes *schema.UANodeSet) error {

	log.Printf("New Node Set: %s", nodes.LastModifiedAttr)

	failures := 0
	reftypes := make(map[string]*schema.UAReferenceType)
	for i := range nodes.UAReferenceType {
		rt := nodes.UAReferenceType[i]
		reftypes[rt.BrowseNameAttr] = rt // sometimes they use browse name
		reftypes[rt.NodeIdAttr] = rt     // sometimes they use node id
	}

	aliases := make(map[string]string)
	for i := range nodes.Aliases.Alias {
		alias := nodes.Aliases.Alias[i]
		aliases[alias.AliasAttr] = alias.Value
	}

	// any of the aliases could be reference types, so we have to check them all and add them to the reftypes map
	// if they are.
	for alias := range aliases {
		aliasID := ua.MustParseNodeID(aliases[alias])
		refnode := srv.Node(aliasID)
		if refnode == nil {
			if srv.cfg.logger != nil {
				srv.cfg.logger.Warn("error loading alias %s", alias)
			}
			continue
		}
		rt := new(schema.UAReferenceType)
		rt.UAType = new(schema.UAType)
		rt.UAType.UANode = new(schema.UANode)
		rt.BrowseNameAttr = alias
		rt.NodeIdAttr = aliases[alias]
		isSymmetricValue, err := refnode.Attribute(ua.AttributeIDSymmetric)
		if err == nil {
			rt.SymmetricAttr = isSymmetricValue.Value.Value.Value().(bool)
		}

		_, ok := reftypes[alias]
		if !ok {
			reftypes[alias] = rt // sometimes they use browse name
		} else {
			if srv.cfg.logger != nil {
				srv.cfg.logger.Error("Duplicate reference type %s", alias)
			}
			continue
		}

		_, ok = reftypes[aliases[alias]]
		if !ok {
			reftypes[aliases[alias]] = rt // sometimes they use node id
		} else {
			if srv.cfg.logger != nil {
				srv.cfg.logger.Error("Duplicate reference type %s", aliases[alias])
			}
			continue
		}

	}

	// the first thing we have to do is go thorugh and define all the nodes.
	// set up the reference types.
	for i := range nodes.UAReferenceType {
		rt := nodes.UAReferenceType[i]

		nodeid := ua.MustParseNodeID(rt.NodeIdAttr)
		node := srv.Node(nodeid)
		if node == nil {
			log.Printf("Error loading node %s", rt.NodeIdAttr)
		}

		for rid := range rt.References.Reference {
			ref := rt.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, rt.BrowseNameAttr)
				failures++
				continue
			}

			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}
			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}
		}

	}

	// set up the data types.
	for i := range nodes.UADataType {
		dt := nodes.UADataType[i]
		nid := ua.MustParseNodeID(dt.NodeIdAttr)
		node := srv.Node(nid)

		if nid.IntID() == 24 {
			log.Printf("doing BaseDataType")
		}

		for rid := range dt.References.Reference {
			ref := dt.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, dt.BrowseNameAttr)
				failures++
				continue
			}

			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}

			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}

		}

	}

	// set up the object types
	for i := range nodes.UAObjectType {
		ot := nodes.UAObjectType[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		node := srv.Node(nid)

		for rid := range ot.References.Reference {
			ref := ot.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, ot.BrowseNameAttr)
				failures++
				continue
			}
			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}
			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}
		}
	}

	// set up the variable Types
	for i := range nodes.UAVariableType {
		ot := nodes.UAVariableType[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		node := srv.Node(nid)

		for rid := range ot.References.Reference {
			ref := ot.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, ot.BrowseNameAttr)
				failures++
				continue
			}
			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}
			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}

		}

	}

	// set up the variables
	for i := range nodes.UAVariable {
		ot := nodes.UAVariable[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		node := srv.Node(nid)

		for rid := range ot.References.Reference {
			ref := ot.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, ot.BrowseNameAttr)
				failures++
				continue
			}
			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}
			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}

		}

	}

	// set up the methods
	for i := range nodes.UAMethod {
		ot := nodes.UAMethod[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		node := srv.Node(nid)

		for rid := range ot.References.Reference {
			ref := ot.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, ot.BrowseNameAttr)
				failures++
				continue
			}
			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}
			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}
		}

	}

	// set up the objects
	for i := range nodes.UAObject {
		ot := nodes.UAObject[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		node := srv.Node(nid)
		if ot.NodeIdAttr == "i=84" {
			log.Printf("doing root.")
		}

		for rid := range ot.References.Reference {
			ref := ot.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				log.Printf("can't find node %s as %s reference to %s", ref.Value, ref.ReferenceTypeAttr, ot.BrowseNameAttr)
				failures++
				continue
			}
			if ref.IsForwardAttr == nil {
				v := true
				ref.IsForwardAttr = &v
			}
			reftypeid := ua.MustParseNodeID(reftypes[ref.ReferenceTypeAttr].NodeIdAttr)
			node.AddRef(n, RefType(reftypeid.IntID()), *ref.IsForwardAttr)
			if !reftypes[ref.ReferenceTypeAttr].SymmetricAttr {
				n.AddRef(node, RefType(reftypeid.IntID()), !*ref.IsForwardAttr)
			}

		}

	}

	return nil
}
