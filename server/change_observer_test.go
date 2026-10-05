package server

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/ua"
)

type eventRecorder struct {
	mu     sync.Mutex
	events []ChangeEvent
}

func (r *eventRecorder) observe(ev ChangeEvent) {
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
}

func (r *eventRecorder) all() []ChangeEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ChangeEvent{}, r.events...)
}

func TestOnChangeMapNamespace(t *testing.T) {
	rec := &eventRecorder{}
	s := New(OnChange(rec.observe), OnChange(nil))
	ns := NewMapNamespace(s, "test")
	s.AddNamespace(ns)

	// before Start: MonitoredItemService is nil and must not panic
	ns.SetValue("temp", 21.5)

	nid := ua.NewStringNodeID(ns.ID(), "temp")
	st := ns.SetAttribute(nid, ua.AttributeIDValue, DataValueFromValue(22.5))
	require.Equal(t, ua.StatusOK, st)

	ev := rec.all()
	require.Len(t, ev, 2)
	require.Equal(t, nid.String(), ev[0].NodeID.String())
	require.Equal(t, 21.5, ev[0].Value.Value.Value())
	require.Equal(t, 22.5, ev[1].Value.Value.Value())
	require.False(t, ev[1].Time.IsZero())
}

func TestOnChangeNodeNamespace(t *testing.T) {
	rec := &eventRecorder{}
	s := New(OnChange(rec.observe))
	ns := NewNodeNameSpace(s, "test")
	s.AddNamespace(ns)
	n := ns.AddNewVariableStringNode("speed", int32(1))

	st := ns.SetAttribute(n.ID(), ua.AttributeIDValue, DataValueFromValue(int32(7)))
	require.Equal(t, ua.StatusOK, st)

	ev := rec.all()
	require.Len(t, ev, 1)
	require.Equal(t, n.ID().String(), ev[0].NodeID.String())
	require.Equal(t, int32(7), ev[0].Value.Value.Value())
}

func TestOnChangeNoObservers(t *testing.T) {
	s := New()
	ns := NewMapNamespace(s, "test")
	s.AddNamespace(ns)
	ns.SetValue("x", 1) // must not panic without observers or MonitoredItemService
}

func TestOnChangeConcurrent(t *testing.T) {
	rec := &eventRecorder{}
	s := New(OnChange(rec.observe))
	ns := NewMapNamespace(s, "test")
	s.AddNamespace(ns)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ns.SetValue("k", i*j)
			}
		}(i)
	}
	wg.Wait()
	require.Len(t, rec.all(), 400)
}
