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
	I, J int64
}

func TestCallMethod(t *testing.T) {
	ua.RegisterExtensionObject(ua.NewStringNodeID(2, "ComplexType"), new(Complex))

	tests := []struct {
		name string
		req  *ua.CallMethodRequest
		resp *ua.CallMethodResult
	}{
		{
			name: "even",
			req: &ua.CallMethodRequest{
				ObjectID: ua.NewStringNodeID(2, "main"),
				MethodID: ua.NewStringNodeID(2, "even"),
				InputArguments: []*ua.Variant{
					ua.MustVariant(int64(12)),
				},
			},
			resp: &ua.CallMethodResult{
				StatusCode:                   ua.StatusOK,
				InputArgumentResults:         []ua.StatusCode{ua.StatusOK},
				InputArgumentDiagnosticInfos: []*ua.DiagnosticInfo{},
				OutputArguments:              []*ua.Variant{ua.MustVariant(true)},
			},
		},
		{
			name: "square",
			req: &ua.CallMethodRequest{
				ObjectID: ua.NewStringNodeID(2, "main"),
				MethodID: ua.NewStringNodeID(2, "square"),
				InputArguments: []*ua.Variant{
					ua.MustVariant(int64(3)),
				},
			},
			resp: &ua.CallMethodResult{
				StatusCode:                   ua.StatusOK,
				InputArgumentResults:         []ua.StatusCode{ua.StatusOK},
				InputArgumentDiagnosticInfos: []*ua.DiagnosticInfo{},
				OutputArguments:              []*ua.Variant{ua.MustVariant(int64(9))},
			},
		},
		{
			name: "sumOfSquare",
			req: &ua.CallMethodRequest{
				ObjectID: ua.NewStringNodeID(2, "main"),
				MethodID: ua.NewStringNodeID(2, "sumOfSquare"),
				InputArguments: []*ua.Variant{
					ua.MustVariant(ua.NewExtensionObject(&Complex{3, 8})),
				},
			},
			resp: &ua.CallMethodResult{
				StatusCode:                   ua.StatusOK,
				InputArgumentResults:         []ua.StatusCode{ua.StatusOK},
				InputArgumentDiagnosticInfos: []*ua.DiagnosticInfo{},
				OutputArguments:              []*ua.Variant{ua.MustVariant(int64(9 + 64))},
			},
		},
		{
			name: "issue768_array_of_extobjs",
			req: &ua.CallMethodRequest{
				ObjectID:       ua.NewStringNodeID(2, "main"),
				MethodID:       ua.NewStringNodeID(2, "issue768"),
				InputArguments: []*ua.Variant{},
			},
			resp: &ua.CallMethodResult{
				StatusCode:                   ua.StatusOK,
				InputArgumentResults:         []ua.StatusCode{},
				InputArgumentDiagnosticInfos: []*ua.DiagnosticInfo{},
				OutputArguments: []*ua.Variant{ua.MustVariant(
					[]*ua.ExtensionObject{
						ua.NewExtensionObject(&Complex{1, 2}),
						ua.NewExtensionObject(&Complex{3, 4}),
					},
				)},
			},
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
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.Call(ctx, tt.req)
			require.NoError(t, err, "Call failed")
			require.Equal(t, tt.resp, resp, "Response not equal")
		})
	}
}
