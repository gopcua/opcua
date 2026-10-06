package faults

import (
	"time"

	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"
)

// observed is what the recorder held when a fault was armed, so the
// fault's Fired can look only at what happened after it.
type observed struct {
	env         *spectest.Environment
	requests    int
	responses   int
	transport   int
	connections int
}

func observe(env *spectest.Environment) observed {
	return observed{
		env:         env,
		requests:    len(env.Recorder.Requests()),
		responses:   len(env.Recorder.Responses()),
		transport:   len(env.Recorder.Transport()),
		connections: env.Relay.ConnectionCount(),
	}
}

func (o observed) requestsSince() []spectest.ServiceRecord[ua.Request] {
	return o.env.Recorder.Requests()[o.requests:]
}

func (o observed) responsesSince() []spectest.ServiceRecord[ua.Response] {
	return o.env.Recorder.Responses()[o.responses:]
}

func (o observed) transportSince() []spectest.TransportRecord {
	return o.env.Recorder.Transport()[o.transport:]
}

// messageWritten says whether a request of m was written to the
// server, which for a hold DelayAt armed means the held message was
// released.
func (o observed) messageWritten(m message.Message) bool {
	if m == message.HEL {
		for _, record := range o.transportSince() {
			if record.Type == spectest.HEL && !record.WrittenAt.IsZero() {
				return true
			}
		}
		return false
	}
	for _, record := range o.requestsSince() {
		decoded, ok := record.Message()
		if !ok {
			continue
		}
		if named, isNamed := spectest.MessageOf(decoded); isNamed && named == m {
			if _, _, written := o.env.Recorder.TimesOf(record.Order); written {
				return true
			}
		}
	}
	return false
}

// serviceFaultWritten says whether the next request of m was answered
// with a ServiceFault that reached the client.
func (o observed) serviceFaultWritten(m message.Message) bool {
	responses := o.responsesSince()
	for _, request := range o.requestsSince() {
		decoded, ok := request.Message()
		if !ok {
			continue
		}
		if named, isNamed := spectest.MessageOf(decoded); !isNamed || named != m {
			continue
		}
		for i := range responses {
			answer := &responses[i]
			if answer.Connection != request.Connection || answer.RequestID != request.RequestID {
				continue
			}
			answerDecoded, decodedOK := answer.Message()
			if !decodedOK {
				continue
			}
			if _, isFault := answerDecoded.(*ua.ServiceFault); isFault {
				if _, _, written := o.env.Recorder.TimesOf(answer.Order); written {
					return true
				}
			}
		}
	}
	return false
}

// anyStalled says whether the relay recorded a message it read but
// never wrote.
func (o observed) anyStalled() bool {
	for _, record := range o.requestsSince() {
		if record.Fate == spectest.Stalled {
			return true
		}
	}
	for _, record := range o.responsesSince() {
		if record.Fate == spectest.Stalled {
			return true
		}
	}
	for _, record := range o.transportSince() {
		if record.Fate == spectest.Stalled {
			return true
		}
	}
	return false
}

// ackStalled says whether the relay discarded an ACK it owes the
// client.
func (o observed) ackStalled() bool {
	for _, record := range o.transportSince() {
		if record.Type == spectest.ACK && record.Fate == spectest.Stalled {
			return true
		}
	}
	return false
}

// responseWritten says whether the server's next answer reached the
// client.
func (o observed) responseWritten() bool {
	for _, record := range o.responsesSince() {
		if _, _, written := o.env.Recorder.TimesOf(record.Order); written {
			return true
		}
	}
	return false
}

// cutFired says whether the one cut Inject armed has fired.
func cutFired(env *spectest.Environment) func() bool {
	return func() bool {
		return len(env.Relay.ArmedCuts()) == 0
	}
}

// delayBelow holds a request half as long as the client waits for it:
// one second for the transport handshakes the client has no timeout
// for, half the Publish timeout for Publish and half the request
// timeout for every other message.
func delayBelow(env *spectest.Environment, m message.Message) time.Duration {
	switch m {
	case message.HEL, message.OpenSecureChannel:
		return time.Second
	case message.Publish:
		return env.PublishTimeout() / 2
	}
	return env.RequestTimeout() / 2
}

// delayAbove holds a request past what the client waits for it.
func delayAbove(env *spectest.Environment, m message.Message) time.Duration {
	if m == message.Publish {
		return env.PublishTimeout() + 2*time.Second
	}
	return env.RequestTimeout() + 2*time.Second
}

func (f messageFault) Options() []spectest.Option { return nil }

func (f messageFault) Inject(env *spectest.Environment) *spectest.Injected {
	switch f.kind {
	case requestLost:
		env.Relay.CutAt(spectest.BeforeRequestReachesServer, f.msg)
		return spectest.NewInjected(cutFired(env))
	case responseLost:
		env.Relay.CutAt(spectest.ResponseNeverReachesClient, f.msg)
		return spectest.NewInjected(cutFired(env))
	case cutAfterResponse:
		env.Relay.CutAt(spectest.AfterResponseReachesClient, f.msg)
		return spectest.NewInjected(cutFired(env))
	case delayBelowTimeout:
		env.Relay.DelayAt(f.msg, delayBelow(env, f.msg))
	case delayAboveTimeout:
		env.Relay.DelayAt(f.msg, delayAbove(env, f.msg))
	}
	o := observe(env)
	return spectest.NewInjected(func() bool { return o.messageWritten(f.msg) })
}

func (f overloadFault) Options() []spectest.Option { return nil }

func (f overloadFault) Inject(env *spectest.Environment) *spectest.Injected {
	o := observe(env)
	env.UpstreamServer().AnswerNextWith(f.service, f.status.StatusCode())
	return spectest.NewInjected(func() bool { return o.serviceFaultWritten(f.service) })
}

func (f linkFault) Options() []spectest.Option { return nil }

func (f linkFault) Inject(env *spectest.Environment) *spectest.Injected {
	switch f.kind {
	case closedOnAccept:
		o := observe(env)
		env.Relay.CloseNextAccepts(3)
		return spectest.NewInjected(func() bool {
			return env.Relay.ConnectionCount() >= o.connections+3
		})
	case listenerClosed:
		o := observe(env)
		env.Relay.CloseListenerFor(3 * env.ReconnectInterval())
		return spectest.NewInjected(func() bool {
			return env.Relay.ConnectionCount() > o.connections
		})
	case stall:
		o := observe(env)
		env.Relay.Stall()
		return spectest.NewInjected(func() bool { return o.anyStalled() })
	case helUnanswered:
		o := observe(env)
		env.Relay.DiscardNextACK()
		return spectest.NewInjected(func() bool { return o.ackStalled() })
	}
	return nil
}

func (f serverFault) Options() []spectest.Option { return nil }

func (f serverFault) Inject(env *spectest.Environment) *spectest.Injected {
	o := observe(env)
	switch f.kind {
	case pause:
		env.Relay.HoldNextResponse(time.Second)
	case duplicateSequence:
		env.UpstreamServer().DuplicateNextAnswer()
	case skippedSequence:
		env.UpstreamServer().SkipNextAnswer()
	}
	return spectest.NewInjected(func() bool { return o.responseWritten() })
}

func (consumerFault) Options() []spectest.Option {
	return []spectest.Option{
		spectest.WithNotificationBuffer(4),
		spectest.WithSlowConsumer(200 * time.Millisecond),
	}
}

func (consumerFault) Inject(env *spectest.Environment) *spectest.Injected {
	return spectest.NewInjected(func() bool { return env.ConsumerBlocked() })
}
