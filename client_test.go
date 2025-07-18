package opcua

import (
	"context"
	"testing"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Send_DoesNotPanicWhenDisconnected(t *testing.T) {
	c, err := NewClient("opc.tcp://example.com:4840")
	require.NoError(t, err, "NewClient failed")

	err = c.Send(context.Background(), &ua.ReadRequest{}, func(i ua.Response) error {
		return nil
	})
	require.Equal(t, ua.StatusBadServerNotConnected, err)
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
			require.Equal(t, tt.want, got)
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
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClient_SetState(t *testing.T) {
	tests := []struct {
		name      string
		state     ConnState
		withChan  bool
		ctxCancel bool
	}{
		{
			name:     "set state without channel",
			state:    Connected,
			withChan: false,
		},
		{
			name:     "set state with channel",
			state:    Connecting,
			withChan: true,
		},
		{
			name:      "set state with cancelled context",
			state:     Disconnected,
			withChan:  true,
			ctxCancel: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var opts []Option
			var stateCh chan ConnState
			if tt.withChan {
				stateCh = make(chan ConnState, 1)
				opts = append(opts, StateChangedCh(stateCh))
			}

			c, err := NewClient("opc.tcp://example.com:4840", opts...)
			require.NoError(t, err)

			if tt.ctxCancel {
				cancel()
			}

			c.setState(ctx, tt.state)

			// Verify state was set correctly
			require.Equal(t, tt.state, c.State())

			// Verify channel received state if channel exists
			if tt.withChan && !tt.ctxCancel {
				select {
				case state := <-stateCh:
					require.Equal(t, tt.state, state)
				default:
					t.Fatal("expected state on channel but got none")
				}
			}
		})
	}
}

func TestClient_LoadNil(t *testing.T) {
	t.Run("normal client init", func(t *testing.T) {
		c, err := NewClient("opc.tcp://dummy")
		require.NoError(t, err)
		assert.NoError(t, c.Close(context.TODO()))
		assert.Nil(t, c.SecureChannel())
		assert.Nil(t, c.Session())
	})
	t.Run("abnormal client init", func(t *testing.T) {
		c := new(Client)
		assert.NoError(t, c.Close(context.TODO()))
		assert.Nil(t, c.SecureChannel())
		assert.Nil(t, c.Session())
	})
}
