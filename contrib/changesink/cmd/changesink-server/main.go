// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Command changesink-server runs a demo gopcua server whose node changes are
// stored in ClickHouse and published to MQTT.
//
// Configuration is read from the environment. A sink is enabled only when its
// address is set:
//
//	OPCUA_HOST, OPCUA_PORT             listen address (default localhost:4840)
//	SINK_SERVER_NAME                   server tag in rows/topics (default hostname)
//
//	CLICKHOUSE_URL                     e.g. http://localhost:8123 (enables ClickHouse)
//	CLICKHOUSE_DATABASE, CLICKHOUSE_TABLE
//	CLICKHOUSE_USER, CLICKHOUSE_PASSWORD
//	CLICKHOUSE_CA_FILE                 PEM CA bundle for https
//	CLICKHOUSE_CREATE_TABLE            "true" to create the table on start
//
//	MQTT_BROKER                        e.g. tcp://localhost:1883 or ssl://host:8883 (enables MQTT)
//	MQTT_USERNAME, MQTT_PASSWORD, MQTT_CLIENT_ID, MQTT_TOPIC_PREFIX
//	MQTT_QOS (default 1), MQTT_RETAIN ("true")
//	MQTT_CA_FILE, MQTT_CERT_FILE, MQTT_KEY_FILE   TLS / mutual TLS
//
// A few demo variables change every second; OPC UA clients can also write
// them (ns=1;s=<name>).
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gopcua/opcua/contrib/changesink"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func tlsConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	if caFile == "" && certFile == "" {
		return nil, nil
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates in %s", caFile)
		}
		cfg.RootCAs = pool
	}
	if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}

func sinkConfigFromEnv() (changesink.Config, error) {
	host, _ := os.Hostname()
	cfg := changesink.Config{
		ServerName: env("SINK_SERVER_NAME", host),
		ClickHouse: changesink.ClickHouseConfig{
			URL:         os.Getenv("CLICKHOUSE_URL"),
			Database:    os.Getenv("CLICKHOUSE_DATABASE"),
			Table:       os.Getenv("CLICKHOUSE_TABLE"),
			User:        os.Getenv("CLICKHOUSE_USER"),
			Password:    os.Getenv("CLICKHOUSE_PASSWORD"),
			CreateTable: os.Getenv("CLICKHOUSE_CREATE_TABLE") == "true",
		},
		MQTT: changesink.MQTTConfig{
			Broker:      os.Getenv("MQTT_BROKER"),
			ClientID:    os.Getenv("MQTT_CLIENT_ID"),
			Username:    os.Getenv("MQTT_USERNAME"),
			Password:    os.Getenv("MQTT_PASSWORD"),
			TopicPrefix: os.Getenv("MQTT_TOPIC_PREFIX"),
			QoS:         1,
			Retain:      os.Getenv("MQTT_RETAIN") == "true",
		},
	}
	if q := os.Getenv("MQTT_QOS"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 0 || n > 2 {
			return cfg, fmt.Errorf("invalid MQTT_QOS %q", q)
		}
		cfg.MQTT.QoS = byte(n)
	}
	var err error
	if cfg.ClickHouse.TLS, err = tlsConfig(os.Getenv("CLICKHOUSE_CA_FILE"), "", ""); err != nil {
		return cfg, fmt.Errorf("clickhouse tls: %w", err)
	}
	if cfg.MQTT.TLS, err = tlsConfig(os.Getenv("MQTT_CA_FILE"), os.Getenv("MQTT_CERT_FILE"), os.Getenv("MQTT_KEY_FILE")); err != nil {
		return cfg, fmt.Errorf("mqtt tls: %w", err)
	}
	return cfg, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := sinkConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	sink, err := changesink.New(ctx, cfg)
	if err != nil {
		log.Fatalf("change sink: %s", err)
	}
	log.Printf("clickhouse sink enabled: %v, mqtt sink enabled: %v", cfg.ClickHouse.URL != "", cfg.MQTT.Broker != "")

	port, err := strconv.Atoi(env("OPCUA_PORT", "4840"))
	if err != nil {
		log.Fatalf("invalid OPCUA_PORT: %s", err)
	}
	srv := server.New(
		server.EndPoint(env("OPCUA_HOST", "localhost"), port),
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.OnChange(sink.Observe),
	)

	ns := server.NewNodeNameSpace(srv, "changesink-demo")
	srv.AddNamespace(ns)
	root, _ := srv.Namespace(0)
	root.Objects().AddRef(ns.Objects(), id.HasComponent, true)

	vars := map[string]any{"temperature": 20.0, "pressure": 1.0, "running": true, "counter": int32(0), "setpoint": 21.0}
	nodes := map[string]*server.Node{}
	for name, v := range vars {
		n := ns.AddNewVariableStringNode(name, v)
		ns.Objects().AddRef(n, id.HasComponent, true)
		nodes[name] = n
	}

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("start server: %s", err)
	}

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var i int32
	for {
		select {
		case <-ctx.Done():
			log.Printf("shutting down")
			srv.Close()
			cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := sink.Close(cctx); err != nil {
				log.Printf("flush sink: %s", err)
			}
			cancel()
			log.Printf("sink stats: %+v", sink.Stats())
			return
		case <-tick.C:
			i++
			t := float64(i)
			set := func(name string, v any) {
				ns.SetAttribute(nodes[name].ID(), ua.AttributeIDValue, server.DataValueFromValue(v))
			}
			set("temperature", 20+5*math.Sin(t/10))
			set("pressure", 1+0.1*math.Cos(t/7))
			set("counter", i)
			if i%10 == 0 {
				set("running", i%20 != 0)
			}
		}
	}
}
