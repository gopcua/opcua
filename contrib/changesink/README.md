# changesink

Stores node changes of a [gopcua](https://github.com/gopcua/opcua) server in
**ClickHouse** and publishes them to an **MQTT** broker.

It is a separate Go module so the core `github.com/gopcua/opcua` module stays
free of MQTT dependencies. It hooks into the server through the dependency-free
`server.OnChange` option, so every change is captured inside the OPC UA server
process:

* writes from OPC UA clients (`Write` service)
* application side changes: `MapNamespace.SetValue`, `NodeNameSpace.SetAttribute`,
  `ChangeNotification(...)`

Both sinks are optional. Each one is enabled only when its address is set, and
both support authentication and TLS.

```go
sink, err := changesink.New(ctx, changesink.Config{
	ServerName: "plant-a",
	ClickHouse: changesink.ClickHouseConfig{
		URL:         "http://clickhouse:8123", // "" disables ClickHouse
		Database:    "default",
		Table:       "opcua_changes",
		User:        "ingest",
		Password:    os.Getenv("CLICKHOUSE_PASSWORD"),
		CreateTable: true,
	},
	MQTT: changesink.MQTTConfig{
		Broker:   "ssl://broker:8883", // "" disables MQTT
		Username: "opcua",
		Password: os.Getenv("MQTT_PASSWORD"),
		TLS:      tlsCfg, // optional: custom CA / client cert
		QoS:      1,
	},
})
if err != nil {
	log.Fatal(err) // includes authentication failures
}
defer sink.Close(ctx) // flushes queued changes

srv := server.New(
	// ... endpoints, security, auth ...
	server.OnChange(sink.Observe),
)
```

## Behavior

* `New` checks both connections up front (`SELECT 1` against ClickHouse, a
  CONNECT to the broker), so a wrong address or password fails at startup.
* `Observe` never blocks the OPC UA server. Each sink has a bounded queue
  (`QueueSize`, default 10000). When a queue is full, the change is dropped and
  counted.
* ClickHouse rows are batched: a batch is flushed when it reaches `BatchSize`
  rows (default 1000) or every `FlushInterval` (default 1s), whichever comes
  first. If an insert fails, the batch is logged, counted as dropped, and not
  retried. Changes are only buffered in memory, never on disk.
* MQTT reconnects automatically. A publish that fails or times out is counted
  as dropped.
* `Close` drains the queues, flushes the last batch, and disconnects.
* `Stats()` exposes the counters `Received`, `ClickHouseWritten`,
  `ClickHouseDropped`, `ClickHouseErrors`, `MQTTPublished`, `MQTTDropped` and
  `MQTTErrors`.

## ClickHouse

Rows are inserted over the HTTP interface with `FORMAT JSONEachRow`, sending the
`X-ClickHouse-User` / `X-ClickHouse-Key` headers. `https://` URLs use
`ClickHouseConfig.TLS`. With `CreateTable: true`, the table below is created if
it doesn't exist:

```sql
CREATE TABLE IF NOT EXISTS `default`.`opcua_changes` (
    ts         DateTime64(3, 'UTC'),        -- server time of the change
    server     LowCardinality(String),      -- Config.ServerName
    node_id    String,                      -- e.g. ns=1;s=temperature
    namespace  UInt16,
    value_str  String,                      -- value as text (JSON for arrays)
    value_num  Nullable(Float64),           -- numeric/bool values, else NULL
    data_type  LowCardinality(String),      -- OPC UA built-in type, e.g. Double, Int32[]
    status     UInt32,                      -- OPC UA status code
    source_ts  DateTime64(3, 'UTC')         -- source timestamp (falls back to ts)
) ENGINE = MergeTree
ORDER BY (node_id, ts)
```

Minimal grants for the ingest user: `INSERT, SELECT ON db.opcua_changes`, plus
`CREATE TABLE` if `CreateTable` is used.

## MQTT

* Topic: `<TopicPrefix>/<node id>`. The default prefix is `opcua/<ServerName>`.
  MQTT wildcard characters and separators (`/ + #`) in a node id are replaced
  with `_`, so each node maps to exactly one topic level. For example,
  `opcua/plant-a/ns=1;s=temperature`.
* Payload (JSON):

```json
{"server":"plant-a","node_id":"ns=1;s=temperature","namespace":1,
 "value_str":"20.5","value_num":20.5,"data_type":"Double","status":0,
 "ts":"2026-10-05T09:27:14.495643Z"}
```

* Brokers: `tcp://`, `ssl://`/`tls://` (uses `MQTTConfig.TLS`, including mutual
  TLS), `ws://`, `wss://`.
* `Retain: true` lets new subscribers receive the last value of each node
  immediately.

## Demo server

`cmd/changesink-server` runs a gopcua server with a few variables that change
every second (OPC UA clients can also write them). It is configured through
environment variables, listed in the package doc of `main.go`.

```bash
# throwaway ClickHouse + Mosquitto with auth
cat > mosquitto.conf <<'EOF'
listener 1883
allow_anonymous false
password_file /mosquitto/config/passwd
EOF
docker run --rm -v $PWD:/mosquitto/config eclipse-mosquitto:2 \
  sh -c 'touch /mosquitto/config/passwd && chmod 0700 /mosquitto/config/passwd && mosquitto_passwd -b /mosquitto/config/passwd opc mqttpass'
docker run -d --name cs-ch -p 8123:8123 -e CLICKHOUSE_USER=ingest -e CLICKHOUSE_PASSWORD=chpass clickhouse/clickhouse-server
docker run -d --name cs-mq -p 1883:1883 -v $PWD:/mosquitto/config eclipse-mosquitto:2

CLICKHOUSE_URL=http://localhost:8123 CLICKHOUSE_USER=ingest CLICKHOUSE_PASSWORD=chpass CLICKHOUSE_CREATE_TABLE=true \
MQTT_BROKER=tcp://localhost:1883 MQTT_USERNAME=opc MQTT_PASSWORD=mqttpass \
go run ./cmd/changesink-server

# watch
docker exec cs-mq mosquitto_sub -u opc -P mqttpass -t 'opcua/#' -v
curl -s -H 'X-ClickHouse-User: ingest' -H 'X-ClickHouse-Key: chpass' localhost:8123 \
  --data-binary 'SELECT node_id, count(), avg(value_num) FROM opcua_changes GROUP BY node_id'
```

## Tests

```bash
cd contrib/changesink
go test -race ./...
```

No external services are needed. The tests use an `httptest` ClickHouse stub
and an in-process MQTT broker ([mochi-mqtt](https://github.com/mochi-mqtt/server))
with username/password auth. `e2e_test.go` drives a real OPC UA client write
through the server into both sinks.
