package spectest

import "github.com/gopcua/opcua/ua"

// RecordedRequest returns a service record of a client request for a
// hand-built environment: its wire order, the connection it rode, the
// request id it carried, the fate the relay gave it and its decoded
// message, so a rule that reads the recorder can run against traffic no
// live client produced.
func RecordedRequest(order, connection int, requestID uint32, fate Fate, message ua.Request) ServiceRecord[ua.Request] {
	return ServiceRecord[ua.Request]{
		Order:      order,
		Connection: connection,
		RequestID:  requestID,
		Fate:       fate,
		message:    message,
	}
}

// RecordedResponse returns a service record of a server response for a
// hand-built environment, as RecordedRequest builds one of a request.
func RecordedResponse(order, connection int, requestID uint32, fate Fate, message ua.Response) ServiceRecord[ua.Response] {
	return ServiceRecord[ua.Response]{
		Order:      order,
		Connection: connection,
		RequestID:  requestID,
		Fate:       fate,
		message:    message,
	}
}

// RecordedEnvironment returns an Environment whose recorder holds the
// given requests and responses and whose delivered values are the given
// ones, so a rule can run its Check against hand-built traffic. Nothing
// in it is live: the recorder raises no harness fault and no relay,
// server or client exists, so a Check reading it must not touch those.
func RecordedEnvironment(t T, requests []ServiceRecord[ua.Request], responses []ServiceRecord[ua.Response], received []int32) *Environment {
	recorder := newRecorder(t)
	recorder.mu.Lock()
	recorder.requests = requests
	recorder.responses = responses
	recorder.mu.Unlock()
	return &Environment{
		t:        t,
		Recorder: recorder,
		received: received,
	}
}
