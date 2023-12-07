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
	"github.com/gopcua/opcua/ua"
	"github.com/pascaldekloe/goe/verify"
)

func TestReadNodeIDWithDecodeFunc(t *testing.T) {
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

	nodeID := ua.NewStringNodeID(2, "IntValZero")

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

	ua.RegisterExtensionObjectFunc(ua.NewStringNodeID(2, "IntValType"), nil, decodeFunc)
	defer ua.Deregister(ua.NewStringNodeID(2, "IntValType"))

	resp, err := c.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]interface{}{"i": float64(0)} // TODO: float64? yay json!
	if got := resp.Results[0].Value.Value().(*ua.ExtensionObject).Value; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v want %#v for a node with an unknown type", got, want)
	}
}

type ExtraComplex struct {
	ignore, i, j int64
}

// TestCallMethod, but instead of passing Complex{3,8} as an input argument, we pass ExtraComplex{42,3,8}
// We expect the same result only because we register the nodeID for Complex objects with a custom encode func
// Imagine ExtraComplex as a newer version of the API, and encodefunc allows for backwards compatibility
func TestCallMethodWithEncodeFunc(t *testing.T) {
	complexNodeID := ua.NewStringNodeID(2, "ComplexType")

	encode := func(v interface{}) ([]byte, error) {
		// map ExtraComplex -> Complex, dropping 'ignore' field
		e, ok := v.(*ua.ExtensionObject)
		if !ok {
			return nil, fmt.Errorf("expected extensionobject")
		}
		if ec, ok := e.Value.(*ExtraComplex); ok {
			e.Value = &Complex{ec.i, ec.j}
			return e.Encode()
		}
		return ua.DefaultEncodeExtensionObject(e)
	}

	ua.RegisterExtensionObjectFunc(complexNodeID, encode, nil)
	defer ua.Deregister(complexNodeID)

	req := &ua.CallMethodRequest{
		ObjectID: ua.NewStringNodeID(2, "main"),
		MethodID: ua.NewStringNodeID(2, "sumOfSquare"),
		InputArguments: []*ua.Variant{
			ua.MustVariant(ua.NewExtensionObject(&ExtraComplex{42, 3, 8}, &ua.ExpandedNodeID{NodeID: complexNodeID})),
		},
	}
	out := []*ua.Variant{ua.MustVariant(int64(9 + 64))}

	ctx := context.Background()

	srv := NewServer("method_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)

	resp, err := c.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.StatusCode, ua.StatusOK; got != want {
		t.Fatalf("got status %v want %v", got, want)
	}
	if got, want := resp.OutputArguments, out; !verify.Values(t, "", got, want) {
		t.Fail()
	}
}

func TestReadUnregisteredExtensionObject(t *testing.T) {
	// TODO ask server for description, then decode anyways?
}
