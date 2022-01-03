// Package stats provides instrumentation for the gopcua library via expvar.
//
// The API is experimental and might change.
package stats

import (
	"expvar"
	"io"
	"reflect"

	"github.com/gopcua/opcua/ua"
)

// stats is the global statistics counter.
var stats = NewStats()

func init() {
	expvar.Publish("gopcua", expvar.Func(func() interface{} { return stats }))
}

// Stats collects gopcua statistics via expvar.
type Stats struct {
	Client       *expvar.Map
	Error        *expvar.Map
	Subscription *expvar.Map
}

func NewStats() *Stats {
	return &Stats{
		Client:       &expvar.Map{},
		Error:        &expvar.Map{},
		Subscription: &expvar.Map{},
	}
}

// Reset resets all counters to zero.
func (s *Stats) Reset() {
	s.Client.Init()
	s.Error.Init()
	s.Subscription.Init()
}

// RecordError updates the metric for an error by one.
func (s *Stats) RecordError(err error) {
	if err == nil {
		return
	}
	switch err {
	case io.EOF:
		s.Error.Add("io.EOF", 1)
	case ua.StatusOK:
		s.Error.Add("ua.StatusOK", 1)
	case ua.StatusBad:
		s.Error.Add("ua.StatusBad", 1)
	case ua.StatusUncertain:
		s.Error.Add("ua.StatusUncertain", 1)
	default:
		switch x := err.(type) {
		case ua.StatusCode:
			s.Error.Add("ua."+ua.StatusCodes[x].Name, 1)
		default:
			s.Error.Add(reflect.TypeOf(err).String(), 1)
		}
	}
}

// convenience functions for the global statistics

// Reset resets all counters to zero.
func Reset() {
	stats.Reset()
}

// Client is the global client statistics map.
func Client() *expvar.Map {
	return stats.Client
}

// Error is the global error statistics map.
func Error() *expvar.Map {
	return stats.Error
}

// Subscription is the global subscription statistics map.
func Subscription() *expvar.Map {
	return stats.Subscription
}

func RecordError(err error) {
	stats.RecordError(err)
}
