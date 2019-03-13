package opcua

// Server is a high-level OPC-UA Server
type Server struct {
	EndpointURL string

	// conn *uasc.ServerConn
}

func (a *Server) Open() error {
	return nil
}

func (a *Server) Close() error {
	return nil
}
