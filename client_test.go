package opcua

import (
	"context"
	"testing"

	"github.com/pascaldekloe/goe/verify"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func TestClient_Send_DoesNotPanicWhenDisconnected(t *testing.T) {
	c := NewClient("opc.tcp://example.com:4840")
	err := c.Send(context.Background(), &ua.ReadRequest{}, func(i interface{}) error {
		return nil
	})
	verify.Values(t, "", err, ua.StatusBadServerNotConnected)
}

func TestCloneReadRequest(t *testing.T) {
	tests := []struct {
		name      string
		req, want *ua.ReadRequest
	}{
		{
			name: "empty",
			req:  &ua.ReadRequest{},
			want: &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{},
			},
		},
		{
			name: "keep values",
			req: &ua.ReadRequest{
				MaxAge:             1,
				TimestampsToReturn: 5,
			},
			want: &ua.ReadRequest{
				MaxAge:             1,
				TimestampsToReturn: 5,
				NodesToRead:        []*ua.ReadValueID{},
			},
		},
		{
			name: "set ReadValueID defaults",
			req: &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{
					{
						NodeID:     ua.MustParseNodeID("i=85"),
						IndexRange: "abc",
					},
				},
			},
			want: &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{
					{
						NodeID:       ua.MustParseNodeID("i=85"),
						AttributeID:  ua.AttributeIDValue,
						IndexRange:   "abc",
						DataEncoding: &ua.QualifiedName{},
					},
				},
			},
		},
		{
			name: "keep ReadValueID values",
			req: &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{
					{
						NodeID:      ua.MustParseNodeID("i=85"),
						AttributeID: 15,
						IndexRange:  "abc",
						DataEncoding: &ua.QualifiedName{
							NamespaceIndex: 5,
							Name:           "xxx",
						},
					},
				},
			},
			want: &ua.ReadRequest{
				NodesToRead: []*ua.ReadValueID{
					{
						NodeID:      ua.MustParseNodeID("i=85"),
						AttributeID: 15,
						IndexRange:  "abc",
						DataEncoding: &ua.QualifiedName{
							NamespaceIndex: 5,
							Name:           "xxx",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cloneReadRequest(tt.req)
			verify.Values(t, "", got, tt.want)
		})
	}
}

func TestCloneBrowseRequest(t *testing.T) {
	tests := []struct {
		name      string
		req, want *ua.BrowseRequest
	}{
		{
			name: "empty",
			req:  &ua.BrowseRequest{},
			want: &ua.BrowseRequest{
				View: &ua.ViewDescription{
					ViewID: ua.NewTwoByteNodeID(0),
				},
				NodesToBrowse: []*ua.BrowseDescription{},
			},
		},
		{
			name: "view id missing",
			req: &ua.BrowseRequest{
				View: &ua.ViewDescription{},
			},
			want: &ua.BrowseRequest{
				View: &ua.ViewDescription{
					ViewID: ua.NewTwoByteNodeID(0),
				},
				NodesToBrowse: []*ua.BrowseDescription{},
			},
		},
		{
			name: "keep view id",
			req: &ua.BrowseRequest{
				View: &ua.ViewDescription{
					ViewID: ua.NewTwoByteNodeID(1),
				},
			},
			want: &ua.BrowseRequest{
				View: &ua.ViewDescription{
					ViewID: ua.NewTwoByteNodeID(1),
				},
				NodesToBrowse: []*ua.BrowseDescription{},
			},
		},
		{
			name: "set reference id",
			req: &ua.BrowseRequest{
				NodesToBrowse: []*ua.BrowseDescription{
					{
						NodeID:          ua.MustParseNodeID("i=85"),
						BrowseDirection: ua.BrowseDirectionForward,
						ReferenceTypeID: nil,
						IncludeSubtypes: true,
						NodeClassMask:   1,
						ResultMask:      2,
					},
				},
			},
			want: &ua.BrowseRequest{
				View: &ua.ViewDescription{
					ViewID: ua.NewTwoByteNodeID(0),
				},
				NodesToBrowse: []*ua.BrowseDescription{
					{
						NodeID:          ua.MustParseNodeID("i=85"),
						BrowseDirection: ua.BrowseDirectionForward,
						ReferenceTypeID: ua.NewNumericNodeID(0, id.References),
						IncludeSubtypes: true,
						NodeClassMask:   1,
						ResultMask:      2,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cloneBrowseRequest(tt.req)
			verify.Values(t, "", got, tt.want)
		})
	}
}
