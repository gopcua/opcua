//go:build integration
// +build integration

package uatest

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

// TestRead performs an integration test to read values
// from an OPC/UA server.
func TestReadUnknowNodeID(t *testing.T) {
	ctx := context.Background()

	srv := NewServer("read_unknow_node_id_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	// read node with unknown extension object
	// This should be OK
	nodeWithUnknownType := ua.NewStringNodeID(2, "IntValZero")

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeWithUnknownType},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := resp.Results[0].Status, ua.StatusBadDataTypeIDUnknown; got != want {
		t.Errorf("got status %v want %v for a node with an unknown type", got, want)
	}

	// check that the connection is still usable by reading another node.
	_, err = c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID: ua.NewNumericNodeID(0, id.Server_ServerStatus_State),
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestReadUnknowNodeIDWithDecodeFunc(t *testing.T) {
	ctx := context.Background()

	srv := NewServer("read_unknow_node_id_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	// read node with unknown extension object
	// This should be OK
	nodeWithUnknownType := ua.NewStringNodeID(2, "IntValZero")

	decodeFunc := func(b []byte, v interface{}) error {
		// decode into map[string]interface, which means
		// decode into dynamically generated go type
		// then json marshal/unmarshal :)
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			return fmt.Errorf("incorrect type to decode into")
		}
		r := &struct {
			I int64 `json:"i"`
		}{} // TODO generate dynamically
		buf := ua.NewBuffer(b)
		buf.ReadStruct(r)
		out := map[string]interface{}{}
		b, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, &out); err != nil {
			return err
		}
		reflect.Indirect(rv).Set(reflect.ValueOf(out))
		return nil
	}
	// note: encodefunc is nil
	ua.RegisterExtensionObjectFunc(ua.NewStringNodeID(2, "IntValType"), nil, decodeFunc)

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeWithUnknownType},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]interface{}{"i": float64(0)} // TODO: float64? yay json!
	if got := resp.Results[0].Value.Value().(*ua.ExtensionObject).Value; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v want %#v for a node with an unknown type", got, want)
	}

	// check that the connection is still usable by reading another node.
	_, err = c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{
				NodeID: ua.NewNumericNodeID(0, id.Server_ServerStatus_State),
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
}
