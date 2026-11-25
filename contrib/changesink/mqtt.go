// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package changesink

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// mqttPublisher publishes records as JSON messages to an MQTT broker.
type mqttPublisher struct {
	cfg    MQTTConfig
	logf   func(string, ...any)
	c      *counters
	client mqtt.Client

	queue chan Record
	done  chan struct{}
	stop  chan context.Context
	once  sync.Once
}

func newMQTTPublisher(cfg Config, c *counters) (*mqttPublisher, error) {
	m := cfg.MQTT
	if m.ClientID == "" {
		m.ClientID = "gopcua-" + cfg.ServerName
	}
	if m.TopicPrefix == "" {
		name := cfg.ServerName
		if name == "" {
			name = "server"
		}
		m.TopicPrefix = "opcua/" + sanitizeTopicLevel(name)
	}
	m.TopicPrefix = strings.TrimRight(m.TopicPrefix, "/")
	if m.QoS > 2 {
		return nil, fmt.Errorf("changesink: invalid MQTT QoS %d", m.QoS)
	}
	if m.ConnectTimeout <= 0 {
		m.ConnectTimeout = 10 * time.Second
	}

	opts := mqtt.NewClientOptions().
		AddBroker(m.Broker).
		SetClientID(m.ClientID).
		SetUsername(m.Username).
		SetPassword(m.Password).
		SetAutoReconnect(true).
		SetConnectRetry(false).
		SetConnectTimeout(m.ConnectTimeout).
		SetOrderMatters(false).
		SetCleanSession(true)
	if m.TLS != nil {
		opts.SetTLSConfig(m.TLS.Clone())
	}
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		cfg.Logger("changesink: mqtt connection lost: %s", err)
	})

	client := mqtt.NewClient(opts)
	tok := client.Connect()
	if !tok.WaitTimeout(m.ConnectTimeout) {
		client.Disconnect(0)
		return nil, fmt.Errorf("changesink: mqtt connect to %s timed out", m.Broker)
	}
	if err := tok.Error(); err != nil {
		return nil, fmt.Errorf("changesink: mqtt connect to %s: %w", m.Broker, err)
	}

	p := &mqttPublisher{
		cfg:    m,
		logf:   cfg.Logger,
		c:      c,
		client: client,
		queue:  make(chan Record, cfg.QueueSize),
		done:   make(chan struct{}),
		stop:   make(chan context.Context, 1),
	}
	go p.run()
	return p, nil
}

// sanitizeTopicLevel makes s safe to use inside an MQTT topic: wildcards and
// separators are replaced so a node id always maps to a single topic level.
func sanitizeTopicLevel(s string) string {
	r := strings.NewReplacer("/", "_", "+", "_", "#", "_", "\x00", "")
	s = r.Replace(s)
	if s == "" {
		return "_"
	}
	return s
}

// Topic returns the topic a record is published on.
func (p *mqttPublisher) topic(r Record) string {
	return p.cfg.TopicPrefix + "/" + sanitizeTopicLevel(r.NodeID)
}

func (p *mqttPublisher) enqueue(r Record) {
	select {
	case p.queue <- r:
	default:
		p.c.mqttDropped.Add(1)
	}
}

func (p *mqttPublisher) run() {
	defer close(p.done)
	for {
		select {
		case r := <-p.queue:
			p.publish(r)
		case <-p.stop:
			for {
				select {
				case r := <-p.queue:
					p.publish(r)
					continue
				default:
				}
				return
			}
		}
	}
}

func (p *mqttPublisher) publish(r Record) {
	payload, err := json.Marshal(r)
	if err != nil {
		p.c.mqttDropped.Add(1)
		return
	}
	tok := p.client.Publish(p.topic(r), p.cfg.QoS, p.cfg.Retain, payload)
	if !tok.WaitTimeout(p.cfg.ConnectTimeout) {
		p.c.mqttErrors.Add(1)
		p.c.mqttDropped.Add(1)
		p.logf("changesink: mqtt publish to %s timed out", p.topic(r))
		return
	}
	if err := tok.Error(); err != nil {
		p.c.mqttErrors.Add(1)
		p.c.mqttDropped.Add(1)
		p.logf("changesink: mqtt publish to %s: %s", p.topic(r), err)
		return
	}
	p.c.mqttPublished.Add(1)
}

func (p *mqttPublisher) close(ctx context.Context) error {
	p.once.Do(func() { p.stop <- ctx })
	var err error
	select {
	case <-p.done:
	case <-ctx.Done():
		err = ctx.Err()
	}
	p.client.Disconnect(250)
	return err
}
