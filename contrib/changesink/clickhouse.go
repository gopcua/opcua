// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package changesink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// clickhouseWriter batches records and inserts them over the ClickHouse HTTP
// interface with FORMAT JSONEachRow.
type clickhouseWriter struct {
	cfg    ClickHouseConfig
	logf   func(string, ...any)
	c      *counters
	client *http.Client

	queue chan Record
	done  chan struct{}
	stop  chan context.Context
	once  sync.Once
}

func newClickHouseWriter(ctx context.Context, cfg Config, c *counters) (*clickhouseWriter, error) {
	ch := cfg.ClickHouse
	if ch.Database == "" {
		ch.Database = "default"
	}
	if ch.Table == "" {
		ch.Table = "opcua_changes"
	}
	if ch.BatchSize <= 0 {
		ch.BatchSize = 1000
	}
	if ch.FlushInterval <= 0 {
		ch.FlushInterval = time.Second
	}
	if ch.Timeout <= 0 {
		ch.Timeout = 10 * time.Second
	}
	u, err := url.Parse(ch.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("changesink: invalid ClickHouse URL %q", ch.URL)
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	if ch.TLS != nil {
		tr.TLSClientConfig = ch.TLS.Clone()
	}
	w := &clickhouseWriter{
		cfg:    ch,
		logf:   cfg.Logger,
		c:      c,
		client: &http.Client{Transport: tr, Timeout: ch.Timeout},
		queue:  make(chan Record, cfg.QueueSize),
		done:   make(chan struct{}),
		stop:   make(chan context.Context, 1),
	}

	if err := w.ping(ctx); err != nil {
		return nil, err
	}
	if ch.CreateTable {
		if err := w.exec(ctx, w.createTableSQL(), nil); err != nil {
			return nil, fmt.Errorf("changesink: create ClickHouse table: %w", err)
		}
	}

	go w.run()
	return w, nil
}

// quoteIdent backtick-quotes an identifier.
func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func (w *clickhouseWriter) tableIdent() string {
	return quoteIdent(w.cfg.Database) + "." + quoteIdent(w.cfg.Table)
}

func (w *clickhouseWriter) createTableSQL() string {
	return `CREATE TABLE IF NOT EXISTS ` + w.tableIdent() + ` (
    ts DateTime64(3, 'UTC'),
    server LowCardinality(String),
    node_id String,
    namespace UInt16,
    value_str String,
    value_num Nullable(Float64),
    data_type LowCardinality(String),
    status UInt32,
    source_ts DateTime64(3, 'UTC')
) ENGINE = MergeTree
ORDER BY (node_id, ts)`
}

func (w *clickhouseWriter) ping(ctx context.Context) error {
	return w.exec(ctx, "SELECT 1", nil)
}

// exec POSTs query (and optional body) to ClickHouse.
func (w *clickhouseWriter) exec(ctx context.Context, query string, body []byte) error {
	u, _ := url.Parse(w.cfg.URL)
	q := u.Query()
	q.Set("database", w.cfg.Database)
	q.Set("query", query)
	u.RawQuery = q.Encode()

	ctx, cancel := context.WithTimeout(ctx, w.cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if w.cfg.User != "" {
		req.Header.Set("X-ClickHouse-User", w.cfg.User)
	}
	if w.cfg.Password != "" {
		req.Header.Set("X-ClickHouse-Key", w.cfg.Password)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("changesink: clickhouse request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("changesink: clickhouse %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (w *clickhouseWriter) enqueue(r Record) {
	select {
	case w.queue <- r:
	default:
		w.c.chDropped.Add(1)
	}
}

func (w *clickhouseWriter) run() {
	defer close(w.done)
	t := time.NewTicker(w.cfg.FlushInterval)
	defer t.Stop()

	batch := make([]Record, 0, w.cfg.BatchSize)
	flush := func(ctx context.Context) {
		if len(batch) == 0 {
			return
		}
		w.insert(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case r := <-w.queue:
			batch = append(batch, r)
			if len(batch) >= w.cfg.BatchSize {
				flush(context.Background())
			}
		case <-t.C:
			flush(context.Background())
		case ctx := <-w.stop:
			// drain what is queued, then flush
			for {
				select {
				case r := <-w.queue:
					batch = append(batch, r)
					if len(batch) >= w.cfg.BatchSize {
						flush(ctx)
					}
					continue
				default:
				}
				break
			}
			flush(ctx)
			return
		}
	}
}

func (w *clickhouseWriter) insert(ctx context.Context, batch []Record) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, r := range batch {
		if err := enc.Encode(r.clickhouseRow()); err != nil {
			w.c.chDropped.Add(1)
			continue
		}
	}
	query := "INSERT INTO " + w.tableIdent() + " FORMAT JSONEachRow"
	if err := w.exec(ctx, query, buf.Bytes()); err != nil {
		w.c.chErrors.Add(1)
		w.c.chDropped.Add(uint64(len(batch)))
		w.logf("changesink: dropped %d rows: %s", len(batch), err)
		return
	}
	w.c.chWritten.Add(uint64(len(batch)))
}

func (w *clickhouseWriter) close(ctx context.Context) error {
	w.once.Do(func() { w.stop <- ctx })
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
