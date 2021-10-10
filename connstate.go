package opcua

// ConnState is the ua client connection state
type ConnState uint8

const (
	// Closed, the Connection is currently closed
	Closed ConnState = iota
	// Connected, the Connection is currently connected
	Connected
	// Connecting, the Connection is currently connecting to a server for the first time
	Connecting
	// Disconnected, the Connection is currently disconnected
	Disconnected
	// Reconnecting, the Connection is currently attempting to reconnect to a server it was previously connected to
	Reconnecting
)
