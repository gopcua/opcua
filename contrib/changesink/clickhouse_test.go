package changesink

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

// fakeClickHouse is a minimal ClickHouse HTTP interface stub.
type fakeClickHouse struct {
	t        *testing.T
	user     string
	password string

	mu      sync.Mutex
	queries []string
	rows    []map[string]any
	inserts int
	block   chan struct{} // when set, inserts block until closed
	fail    bool
}

func (f *fakeClickHouse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("X-ClickHouse-User") != f.user || r.Header.Get("X-ClickHouse-Key") != f.password {
		http.Error(w, "Code: 516. DB::Exception: Authentication failed", http.StatusUnauthorized)
		return
	}
	q := r.URL.Query().Get("query")
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	f.queries = append(f.queries, q)
	block, fail := f.block, f.fail
	f.mu.Unlock()

	if strings.HasPrefix(q, "INSERT") {
		if block != nil {
			<-block
		}
		if fail {
			http.Error(w, "Code: 60. DB::Exception: Table does not exist", http.StatusNotFound)
			return
		}
		require.Equal(f.t, "analytics", r.URL.Query().Get("database"))
		require.Equal(f.t, "application/x-ndjson", r.Header.Get("Content-Type"))
		f.mu.Lock()
		f.inserts++
		sc := bufio.NewScanner(bytes.NewReader(body))
		for sc.Scan() {
			var m map[string]any
			require.NoError(f.t, json.Unmarshal(sc.Bytes(), &m))
			f.rows = append(f.rows, m)
		}
		f.mu.Unlock()
	}
	_, _ = w.Write([]byte("1\n"))
}

func (f *fakeClickHouse) snapshot() (queries []string, rows []map[string]any, inserts int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.queries...), append([]map[string]any{}, f.rows...), f.inserts
}

func newFakeClickHouse(t *testing.T) (*fakeClickHouse, *httptest.Server) {
	f := &fakeClickHouse{t: t, user: "ingest", password: "p@ss"}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, srv
}

func chConfig(url string) Config {
	return Config{
		ServerName: "plant-a",
		QueueSize:  100,
		ClickHouse: ClickHouseConfig{
			URL:           url,
			Database:      "analytics",
			Table:         "opcua_changes",
			User:          "ingest",
			Password:      "p@ss",
			BatchSize:     3,
			FlushInterval: time.Hour, // only size/close flushes unless overridden
			CreateTable:   true,
		},
		Logger: func(string, ...any) {},
	}
}

func event(key string, v any) server.ChangeEvent {
	if i, ok := v.(int); ok {
		v = int64(i) // ua.Variant has no platform int
	}
	return server.ChangeEvent{
		NodeID: ua.NewStringNodeID(2, key),
		Value:  &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(v)},
		Time:   time.Now(),
	}
}

func TestClickHouseBatchAndClose(t *testing.T) {
	f, srv := newFakeClickHouse(t)
	ctx := context.Background()

	s, err := New(ctx, chConfig(srv.URL))
	require.NoError(t, err)
	require.True(t, s.Enabled())

	queries, _, _ := f.snapshot()
	require.Equal(t, "SELECT 1", queries[0])
	require.Contains(t, queries[1], "CREATE TABLE IF NOT EXISTS `analytics`.`opcua_changes`")
	require.Contains(t, queries[1], "ENGINE = MergeTree")

	for i := 0; i < 4; i++ {
		s.Observe(event("temp", float64(i)))
	}
	// batch of 3 is flushed by size
	require.Eventually(t, func() bool { _, _, n := f.snapshot(); return n == 1 }, 5*time.Second, 10*time.Millisecond)

	// the 4th row is flushed on close
	require.NoError(t, s.Close(ctx))
	queries, rows, inserts := f.snapshot()
	require.Equal(t, 2, inserts)
	require.Len(t, rows, 4)
	require.Equal(t, "INSERT INTO `analytics`.`opcua_changes` FORMAT JSONEachRow", queries[len(queries)-1])

	require.Equal(t, "plant-a", rows[0]["server"])
	require.Equal(t, "ns=2;s=temp", rows[0]["node_id"])
	require.Equal(t, float64(2), rows[0]["namespace"])
	require.Equal(t, "0", rows[0]["value_str"])
	require.Equal(t, float64(0), rows[0]["value_num"])
	require.Equal(t, "Double", rows[0]["data_type"])
	require.Regexp(t, `^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d\.\d{3}$`, rows[0]["ts"])
	require.Equal(t, float64(3), rows[3]["value_num"])

	st := s.Stats()
	require.Equal(t, uint64(4), st.Received)
	require.Equal(t, uint64(4), st.ClickHouseWritten)
	require.Zero(t, st.ClickHouseDropped)

	// Close is idempotent
	require.NoError(t, s.Close(ctx))
}

func TestClickHouseFlushInterval(t *testing.T) {
	f, srv := newFakeClickHouse(t)
	cfg := chConfig(srv.URL)
	cfg.ClickHouse.BatchSize = 1000
	cfg.ClickHouse.FlushInterval = 20 * time.Millisecond
	s, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer s.Close(context.Background())

	s.Observe(event("a", "x"))
	require.Eventually(t, func() bool { _, rows, _ := f.snapshot(); return len(rows) == 1 }, 5*time.Second, 10*time.Millisecond)
}

func TestClickHouseAuthFailure(t *testing.T) {
	_, srv := newFakeClickHouse(t)
	cfg := chConfig(srv.URL)
	cfg.ClickHouse.Password = "wrong"
	_, err := New(context.Background(), cfg)
	require.ErrorContains(t, err, "401")
	require.ErrorContains(t, err, "Authentication failed")
}

func TestClickHouseInvalidURL(t *testing.T) {
	for _, u := range []string{"localhost:8123", "ftp://x", "http://"} {
		_, err := New(context.Background(), chConfig(u))
		require.Error(t, err, u)
	}
}

func TestClickHouseInsertErrorCountsDrops(t *testing.T) {
	f, srv := newFakeClickHouse(t)
	s, err := New(context.Background(), chConfig(srv.URL))
	require.NoError(t, err)
	f.mu.Lock()
	f.fail = true
	f.mu.Unlock()

	for i := 0; i < 3; i++ {
		s.Observe(event("a", i))
	}
	require.NoError(t, s.Close(context.Background()))
	st := s.Stats()
	require.Equal(t, uint64(1), st.ClickHouseErrors)
	require.Equal(t, uint64(3), st.ClickHouseDropped)
	require.Zero(t, st.ClickHouseWritten)
}

func TestClickHouseQueueFullDropsWithoutBlocking(t *testing.T) {
	f, srv := newFakeClickHouse(t)
	cfg := chConfig(srv.URL)
	cfg.QueueSize = 5
	cfg.ClickHouse.BatchSize = 1
	s, err := New(context.Background(), cfg)
	require.NoError(t, err)

	block := make(chan struct{})
	f.mu.Lock()
	f.block = block
	f.mu.Unlock()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			s.Observe(event("a", i))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Observe blocked on a slow ClickHouse")
	}
	require.Greater(t, s.Stats().ClickHouseDropped, uint64(90))

	close(block)
	require.NoError(t, s.Close(context.Background()))
	st := s.Stats()
	require.Equal(t, uint64(100), st.ClickHouseWritten+st.ClickHouseDropped)
}

func TestNoopSink(t *testing.T) {
	s, err := New(context.Background(), Config{})
	require.NoError(t, err)
	require.False(t, s.Enabled())
	s.Observe(event("a", 1))
	require.Zero(t, s.Stats().Received)
	require.NoError(t, s.Close(context.Background()))
}

func TestQuoteIdent(t *testing.T) {
	require.Equal(t, "`a``b`", quoteIdent("a`b"))
}
