package server

import (
	"fmt"

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
	// todo(fs): what is this supposed to log?
	srv.logger.Debug("New Node Set", "last_modified", nodes.LastModifiedAttr)

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
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
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
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
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
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
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
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
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
		if len(ot.DisplayName) > 0 {
			attrs[ua.AttributeIDDisplayName] = DataValueFromValue(ua.NewLocalizedText(ot.DisplayName[0].Value))
		}
		if len(ot.Description) > 0 {
			attrs[ua.AttributeIDDescription] = DataValueFromValue(ua.NewLocalizedText(ot.Description[0].Value))
		}
		attrs[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(ua.NodeClassVariable))

		var refs References = make([]*ua.ReferenceDescription, 0)

		n := NewNode(nid, attrs, refs, nil)
		ns, err := srv.Namespace(int(nid.Namespace()))
		if err != nil {
			// This namespace doesn't exist.
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
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
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
			return err
		}
		ns.AddNode(n)
	}

	// set up the objects
	for i := range nodes.UAObject {
		ot := nodes.UAObject[i]
		nid := ua.MustParseNodeID(ot.NodeIdAttr)
		if ot.NodeIdAttr == "i=85" {
			srv.logger.Debug("doing objects.")
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
			srv.logger.Warn("Could Not Find Namespace", "ns", nid.Namespace())
			return err
		}
		ns.AddNode(n)
	}

	return nil
}

func (srv *Server) refsImportNodeSet(nodes *schema.UANodeSet) error {
	srv.logger.Debug("New Node Set", "last_modified", nodes.LastModifiedAttr)

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
			srv.logger.Warn("error loading alias", "alias", alias)
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
			srv.logger.Error("Duplicate reference", "type", alias)
			continue
		}

		_, ok = reftypes[aliases[alias]]
		if !ok {
			reftypes[aliases[alias]] = rt // sometimes they use node id
		} else {
			srv.logger.Error("Duplicate reference", "type", aliases[alias])
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
			srv.logger.Warn("Error loading node", "node_id", rt.NodeIdAttr)
		}

		for rid := range rt.References.Reference {
			ref := rt.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				srv.logger.Warn("can't find reference", "node", ref.Value, "refType", ref.ReferenceTypeAttr, "refTarget", rt.BrowseNameAttr)
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
			srv.logger.Debug("doing BaseDataType")
		}

		for rid := range dt.References.Reference {
			ref := dt.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				srv.logger.Warn("cannot find reference", "node_id", ref.Value, "ref_type", ref.ReferenceTypeAttr, "browse_name", dt.BrowseNameAttr)
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
				srv.logger.Warn("cannot find reference", "node_id", ref.Value, "ref_type", ref.ReferenceTypeAttr, "browse_name", ot.BrowseNameAttr)
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
				srv.logger.Warn("can't find reference", "node_id", ref.Value, "ref_type", ref.ReferenceTypeAttr, "browse_name", ot.BrowseNameAttr)
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
				srv.logger.Warn("can't find reference", "node_id", ref.Value, "ref_type", ref.ReferenceTypeAttr, "browse_name", ot.BrowseNameAttr)
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
				srv.logger.Warn("can't find reference", "node_id", ref.Value, "ref_type", ref.ReferenceTypeAttr, "browse_name", ot.BrowseNameAttr)
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
			srv.logger.Debug("doing root.")
		}

		for rid := range ot.References.Reference {
			ref := ot.References.Reference[rid]
			refnodeid := ua.MustParseNodeID(ref.Value)
			n := srv.Node(refnodeid)
			if n == nil {
				srv.logger.Warn("can't find reference", "node_id", ref.Value, "ref_type", ref.ReferenceTypeAttr, "browse_name", ot.BrowseNameAttr)
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
