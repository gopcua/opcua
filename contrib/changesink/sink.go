// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package changesink stores and publishes node changes of a gopcua server.
//
// When a ClickHouse URL is configured, every change is inserted into a
// ClickHouse table over the HTTP interface (batched). When an MQTT broker is
// configured, every change is published as a JSON message. Both connections
// support authentication and TLS.
//
// Wire it into a server with the server.OnChange hook:
//
//	sink, err := changesink.New(ctx, cfg)
//	srv := server.New(..., server.OnChange(sink.Observe))
//	defer sink.Close(ctx)
//
// Observe never blocks the OPC UA server: each sink has a bounded queue and
// drops (and counts) changes when it is full.
package changesink

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/server"
)

// Config configures the sinks. A sink is disabled when its address is empty.
type Config struct {
	// ServerName is written with every row and used in the default MQTT topic.
	ServerName string
	// QueueSize is the per-sink queue length. Default 10000.
	QueueSize int
	// ClickHouse is disabled when ClickHouse.URL is empty.
	ClickHouse ClickHouseConfig
	// MQTT is disabled when MQTT.Broker is empty.
	MQTT MQTTConfig
	// Logger receives error messages. Default log.Printf.
	Logger func(format string, args ...any)
}

// ClickHouseConfig configures the ClickHouse HTTP sink.
type ClickHouseConfig struct {
	// URL of the ClickHouse HTTP interface, e.g. http://localhost:8123.
	URL string
	// Database, default "default".
	Database string
	// Table, default "opcua_changes".
	Table string
	// User and Password are sent as X-ClickHouse-User / X-ClickHouse-Key.
	User     string
	Password string
	// TLS is used for https URLs (custom CA, client certificates).
	TLS *tls.Config
	// BatchSize flushes when this many rows are buffered. Default 1000.
	BatchSize int
	// FlushInterval flushes buffered rows at least this often. Default 1s.
	FlushInterval time.Duration
	// Timeout per HTTP request. Default 10s.
	Timeout time.Duration
	// CreateTable runs CREATE TABLE IF NOT EXISTS on start.
	CreateTable bool
}

// MQTTConfig configures the MQTT publisher.
type MQTTConfig struct {
	// Broker URL: tcp://, ssl://, tls://, ws:// or wss://.
	Broker string
	// ClientID, default "gopcua-<ServerName>".
	ClientID string
	// Username and Password for broker authentication.
	Username string
	Password string
	// TLS for ssl/tls/wss brokers (custom CA, client certificates).
	TLS *tls.Config
	// TopicPrefix, default "opcua/<ServerName>". The topic is <prefix>/<node id>.
	TopicPrefix string
	// QoS 0 (at most once), 1 (at least once) or 2 (exactly once).
	QoS byte
	// Retain publishes retained messages so subscribers get the last value.
	Retain bool
	// ConnectTimeout for the initial connection. Default 10s.
	ConnectTimeout time.Duration
}

// Stats are cumulative counters of a Sink.
type Stats struct {
	// Received is the number of change events passed to Observe.
	Received uint64
	// ClickHouseWritten is the number of rows inserted into ClickHouse.
	ClickHouseWritten uint64
	// ClickHouseDropped is the number of rows dropped (queue full or failed insert).
	ClickHouseDropped uint64
	// ClickHouseErrors is the number of failed insert requests.
	ClickHouseErrors uint64
	// MQTTPublished is the number of messages published to the broker.
	MQTTPublished uint64
	// MQTTDropped is the number of messages dropped (queue full or failed publish).
	MQTTDropped uint64
	// MQTTErrors is the number of failed publishes.
	MQTTErrors uint64
}

type counters struct {
	received, chWritten, chDropped, chErrors, mqttPublished, mqttDropped, mqttErrors atomic.Uint64
}

// Sink fans out node changes to ClickHouse and MQTT.
type Sink struct {
	cfg  Config
	ch   *clickhouseWriter
	mqtt *mqttPublisher
	c    counters

	closeOnce sync.Once
}

// New creates the configured sinks. If CreateTable is set, the ClickHouse table
// is created. The MQTT connection is established before New returns, so
// authentication errors are reported immediately. With neither address
// configured New returns a no-op sink.
func New(ctx context.Context, cfg Config) (*Sink, error) {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 10000
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Printf
	}
	s := &Sink{cfg: cfg}

	if cfg.ClickHouse.URL != "" {
		w, err := newClickHouseWriter(ctx, cfg, &s.c)
		if err != nil {
			return nil, err
		}
		s.ch = w
	}
	if cfg.MQTT.Broker != "" {
		p, err := newMQTTPublisher(cfg, &s.c)
		if err != nil {
			if s.ch != nil {
				s.ch.close(ctx)
			}
			return nil, err
		}
		s.mqtt = p
	}
	return s, nil
}

// Enabled reports whether at least one sink is configured.
func (s *Sink) Enabled() bool {
	return s != nil && (s.ch != nil || s.mqtt != nil)
}

// Observe enqueues a change event. It never blocks. Use it with server.OnChange.
func (s *Sink) Observe(ev server.ChangeEvent) {
	if !s.Enabled() {
		return
	}
	s.c.received.Add(1)
	r := NewRecord(s.cfg.ServerName, ev)
	if s.ch != nil {
		s.ch.enqueue(r)
	}
	if s.mqtt != nil {
		s.mqtt.enqueue(r)
	}
}

// Stats returns the current counters.
func (s *Sink) Stats() Stats {
	return Stats{
		Received:          s.c.received.Load(),
		ClickHouseWritten: s.c.chWritten.Load(),
		ClickHouseDropped: s.c.chDropped.Load(),
		ClickHouseErrors:  s.c.chErrors.Load(),
		MQTTPublished:     s.c.mqttPublished.Load(),
		MQTTDropped:       s.c.mqttDropped.Load(),
		MQTTErrors:        s.c.mqttErrors.Load(),
	}
}

// Close flushes queued changes and disconnects. ctx bounds the flush.
// Observe must not be called after Close.
func (s *Sink) Close(ctx context.Context) error {
	var errs []error
	s.closeOnce.Do(func() {
		if s.ch != nil {
			errs = append(errs, s.ch.close(ctx))
		}
		if s.mqtt != nil {
			errs = append(errs, s.mqtt.close(ctx))
		}
	})
	return errors.Join(errs...)
}
