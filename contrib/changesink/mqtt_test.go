package changesink

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	mqttsrv "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/stretchr/testify/require"
)

type received struct {
	topic   string
	payload []byte
	retain  bool
	qos     byte
}

// startBroker starts an in-process MQTT broker that only accepts u/p.
func startBroker(t *testing.T) (addr string, msgs func() []received) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr = l.Addr().String()
	require.NoError(t, l.Close())

	b := mqttsrv.New(&mqttsrv.Options{
		InlineClient: true,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, b.AddHook(new(auth.Hook), &auth.Options{
		Ledger: &auth.Ledger{
			Auth: auth.AuthRules{
				{Username: "u", Password: "p", Allow: true},
			},
		},
	}))
	require.NoError(t, b.AddListener(listeners.NewTCP(listeners.Config{ID: "t", Address: addr})))
	require.NoError(t, b.Serve())
	t.Cleanup(func() { b.Close() })

	var mu sync.Mutex
	var got []received
	require.NoError(t, b.Subscribe("opcua/#", 1, func(_ *mqttsrv.Client, _ packets.Subscription, pk packets.Packet) {
		mu.Lock()
		got = append(got, received{topic: pk.TopicName, payload: append([]byte{}, pk.Payload...), retain: pk.FixedHeader.Retain, qos: pk.FixedHeader.Qos})
		mu.Unlock()
	}))
	return "tcp://" + addr, func() []received {
		mu.Lock()
		defer mu.Unlock()
		return append([]received{}, got...)
	}
}

func mqttConfig(broker string) Config {
	return Config{
		ServerName: "plant/a",
		QueueSize:  100,
		MQTT: MQTTConfig{
			Broker:         broker,
			Username:       "u",
			Password:       "p",
			QoS:            1,
			Retain:         true,
			ConnectTimeout: 5 * time.Second,
		},
		Logger: func(string, ...any) {},
	}
}

func TestMQTTPublish(t *testing.T) {
	broker, msgs := startBroker(t)
	ctx := context.Background()

	s, err := New(ctx, mqttConfig(broker))
	require.NoError(t, err)

	s.Observe(event("line1/temp", 21.5))
	s.Observe(event("pump#1", true))

	require.Eventually(t, func() bool { return len(msgs()) == 2 }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, s.Close(ctx))

	got := msgs()
	byTopic := map[string]received{}
	for _, m := range got {
		byTopic[m.topic] = m
	}
	m, ok := byTopic["opcua/plant_a/ns=2;s=line1_temp"]
	require.True(t, ok, "topics: %v", got)
	require.True(t, m.retain)
	require.Equal(t, byte(1), m.qos)

	var p map[string]any
	require.NoError(t, json.Unmarshal(m.payload, &p))
	require.Equal(t, "ns=2;s=line1/temp", p["node_id"])
	require.Equal(t, "21.5", p["value_str"])
	require.Equal(t, 21.5, p["value_num"])
	require.Equal(t, "Double", p["data_type"])
	require.Equal(t, "plant/a", p["server"])
	require.NotEmpty(t, p["ts"])

	_, ok = byTopic["opcua/plant_a/ns=2;s=pump_1"]
	require.True(t, ok, "topics: %v", got)

	st := s.Stats()
	require.Equal(t, uint64(2), st.MQTTPublished)
	require.Zero(t, st.MQTTDropped)
}

func TestMQTTCustomTopicPrefix(t *testing.T) {
	broker, msgs := startBroker(t)
	cfg := mqttConfig(broker)
	cfg.MQTT.TopicPrefix = "opcua/custom/"
	cfg.MQTT.QoS = 0
	cfg.MQTT.Retain = false
	s, err := New(context.Background(), cfg)
	require.NoError(t, err)
	s.Observe(event("x", int32(1)))
	require.Eventually(t, func() bool { return len(msgs()) == 1 }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, s.Close(context.Background()))
	require.Equal(t, "opcua/custom/ns=2;s=x", msgs()[0].topic)
	require.False(t, msgs()[0].retain)
}

func TestMQTTAuthFailure(t *testing.T) {
	broker, _ := startBroker(t)
	cfg := mqttConfig(broker)
	cfg.MQTT.Password = "wrong"
	_, err := New(context.Background(), cfg)
	require.Error(t, err)
	require.ErrorContains(t, err, "mqtt connect")
}

func TestMQTTInvalidQoS(t *testing.T) {
	cfg := mqttConfig("tcp://127.0.0.1:1")
	cfg.MQTT.QoS = 3
	_, err := New(context.Background(), cfg)
	require.ErrorContains(t, err, "QoS")
}

func TestBothSinks(t *testing.T) {
	broker, msgs := startBroker(t)
	f, srv := newFakeClickHouse(t)
	cfg := chConfig(srv.URL)
	cfg.MQTT = mqttConfig(broker).MQTT

	s, err := New(context.Background(), cfg)
	require.NoError(t, err)
	s.Observe(event("a", 1.0))
	require.NoError(t, s.Close(context.Background()))

	_, rows, _ := f.snapshot()
	require.Len(t, rows, 1)
	require.Eventually(t, func() bool { return len(msgs()) == 1 }, 5*time.Second, 10*time.Millisecond)
}

func TestMQTTFailureClosesClickHouse(t *testing.T) {
	_, srv := newFakeClickHouse(t)
	cfg := chConfig(srv.URL)
	cfg.MQTT = MQTTConfig{Broker: "tcp://127.0.0.1:1", ConnectTimeout: time.Second}
	_, err := New(context.Background(), cfg)
	require.Error(t, err)
}
