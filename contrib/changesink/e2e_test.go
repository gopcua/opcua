package changesink_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	mqttsrv "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/contrib/changesink"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestServerWriteReachesClickHouseAndMQTT drives a real OPC UA write through
// the gopcua server and checks that the change is inserted into ClickHouse and
// published on MQTT, both with authentication.
func TestServerWriteReachesClickHouseAndMQTT(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ClickHouse stub
	var chMu sync.Mutex
	var chBodies []string
	ch := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ClickHouse-User") != "ingest" || r.Header.Get("X-ClickHouse-Key") != "chpass" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		if strings.HasPrefix(r.URL.Query().Get("query"), "INSERT") {
			b, _ := io.ReadAll(r.Body)
			chMu.Lock()
			chBodies = append(chBodies, string(b))
			chMu.Unlock()
		}
	}))
	defer ch.Close()

	// MQTT broker with auth
	brokerAddr := fmt.Sprintf("127.0.0.1:%d", freePort(t))
	b := mqttsrv.New(&mqttsrv.Options{InlineClient: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	require.NoError(t, b.AddHook(new(auth.Hook), &auth.Options{Ledger: &auth.Ledger{
		Auth: auth.AuthRules{{Username: "opc", Password: "mqttpass", Allow: true}},
	}}))
	require.NoError(t, b.AddListener(listeners.NewTCP(listeners.Config{ID: "t", Address: brokerAddr})))
	require.NoError(t, b.Serve())
	defer b.Close()
	var mqMu sync.Mutex
	var mqMsgs []packets.Packet
	require.NoError(t, b.Subscribe("opcua/#", 1, func(_ *mqttsrv.Client, _ packets.Subscription, pk packets.Packet) {
		mqMu.Lock()
		mqMsgs = append(mqMsgs, pk)
		mqMu.Unlock()
	}))

	sink, err := changesink.New(ctx, changesink.Config{
		ServerName: "e2e",
		ClickHouse: changesink.ClickHouseConfig{
			URL: ch.URL, User: "ingest", Password: "chpass",
			FlushInterval: 20 * time.Millisecond,
		},
		MQTT: changesink.MQTTConfig{
			Broker: "tcp://" + brokerAddr, Username: "opc", Password: "mqttpass", QoS: 1,
		},
	})
	require.NoError(t, err)

	// OPC UA server
	port := freePort(t)
	srv := server.New(
		server.EndPoint("localhost", port),
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.OnChange(sink.Observe),
	)
	ns := server.NewNodeNameSpace(srv, "e2e")
	srv.AddNamespace(ns)
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)
	n := ns.AddNewVariableStringNode("setpoint", float64(1))
	ns.Objects().AddRef(n, id.HasComponent, true)
	require.NoError(t, srv.Start(ctx))
	defer srv.Close()

	addr := fmt.Sprintf("opc.tcp://localhost:%d", port)
	var c *opcua.Client
	require.Eventually(t, func() bool {
		c, err = opcua.NewClient(addr, opcua.SecurityMode(ua.MessageSecurityModeNone))
		return err == nil && c.Connect(ctx) == nil
	}, 10*time.Second, 50*time.Millisecond)
	defer c.Close(ctx)

	resp, err := c.Write(ctx, &ua.WriteRequest{NodesToWrite: []*ua.WriteValue{{
		NodeID:      n.ID(),
		AttributeID: ua.AttributeIDValue,
		Value:       &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(42.5)},
	}}})
	require.NoError(t, err)
	require.Equal(t, ua.StatusOK, resp.Results[0])

	require.Eventually(t, func() bool {
		chMu.Lock()
		defer chMu.Unlock()
		return len(chBodies) > 0
	}, 5*time.Second, 20*time.Millisecond, "no ClickHouse insert")
	require.Eventually(t, func() bool {
		mqMu.Lock()
		defer mqMu.Unlock()
		return len(mqMsgs) > 0
	}, 5*time.Second, 20*time.Millisecond, "no MQTT message")

	require.NoError(t, sink.Close(ctx))

	chMu.Lock()
	var row map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.SplitN(chBodies[0], "\n", 2)[0]), &row))
	chMu.Unlock()
	require.Equal(t, "e2e", row["server"])
	require.Equal(t, n.ID().String(), row["node_id"])
	require.Equal(t, 42.5, row["value_num"])

	mqMu.Lock()
	pk := mqMsgs[0]
	mqMu.Unlock()
	require.Equal(t, "opcua/e2e/"+n.ID().String(), pk.TopicName)
	var msg map[string]any
	require.NoError(t, json.Unmarshal(pk.Payload, &msg))
	require.Equal(t, "42.5", msg["value_str"])

	st := sink.Stats()
	require.Equal(t, uint64(1), st.Received)
	require.Equal(t, uint64(1), st.ClickHouseWritten)
	require.Equal(t, uint64(1), st.MQTTPublished)
}
