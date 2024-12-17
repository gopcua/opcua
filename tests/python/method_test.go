//go:build integration

package uatest

import (
	"context"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

type Complex struct {
	i, j int64
}

func TestCallMethod(t *testing.T) {
	ua.RegisterExtensionObject(ua.NewStringNodeID(2, "ComplexType"), new(Complex))

	tests := []struct {
		req *ua.CallMethodRequest
		out []*ua.Variant
	}{
		{
			req: &ua.CallMethodRequest{
				ObjectID: ua.NewStringNodeID(2, "main"),
				MethodID: ua.NewStringNodeID(2, "even"),
				InputArguments: []*ua.Variant{
					ua.MustVariant(int64(12)),
				},
			},
			out: []*ua.Variant{ua.MustVariant(true)},
		},
		{
			req: &ua.CallMethodRequest{
				ObjectID: ua.NewStringNodeID(2, "main"),
				MethodID: ua.NewStringNodeID(2, "square"),
				InputArguments: []*ua.Variant{
					ua.MustVariant(int64(3)),
				},
			},
			out: []*ua.Variant{ua.MustVariant(int64(9))},
		},
		{
			req: &ua.CallMethodRequest{
				ObjectID: ua.NewStringNodeID(2, "main"),
				MethodID: ua.NewStringNodeID(2, "sumOfSquare"),
				InputArguments: []*ua.Variant{
					ua.MustVariant(ua.NewExtensionObject(&Complex{3, 8})),
				},
			},
			out: []*ua.Variant{ua.MustVariant(int64(9 + 64))},
		},
	}

	ctx := context.Background()

	srv := NewPythonServer("method_server.py")
	defer srv.Close()

	c, err := opcua.NewClient(srv.Endpoint, srv.Opts...)
	require.NoError(t, err, "NewClient failed")

	err = c.Connect(ctx)
	require.NoError(t, err, "Connect failed")
	defer c.Close(ctx)

	for _, tt := range tests {
		t.Run(tt.req.ObjectID.String(), func(t *testing.T) {
			resp, err := c.Call(ctx, tt.req)
			require.NoError(t, err, "Call failed")
			require.Equal(t, ua.StatusOK, resp.StatusCode, "StatusCode not equal")
			require.Equal(t, tt.out, resp.OutputArguments, "OuptutArgs not equal")
		})
	}
}
