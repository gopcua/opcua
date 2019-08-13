// +build integration

package uatest

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// Server runs a python test server.
type Server struct {
	// Path is the path to the Python server.
	Path string

	// Endpoint is the endpoint address which will be set
	// after the server has started.
	Endpoint string

	// Opts contains the client options required to connect to the server.
	// They are valid after the server has been started.
	Opts []opcua.Option

	cmd *exec.Cmd
}

// NewServer creates a test server and starts it. The function
// panics if the server cannot be started.
func NewServer(path string) *Server {
	s := &Server{Path: path}
	if err := s.Run(); err != nil {
		panic(err)
	}
	return s
}

func (s *Server) Run() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, s.Path)
	s.cmd = exec.Command("python3", path)
	s.Endpoint = "opc.tcp://127.0.0.1:4840"
	s.Opts = []opcua.Option{opcua.SecurityMode(ua.MessageSecurityModeNone)}
	if err := s.cmd.Start(); err != nil {
		return err
	}

	// wait until endpoint is available
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", "127.0.0.1:4840")
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		c.Close()
		return nil
	}
	return fmt.Errorf("timeout")
}

func (s *Server) Close() error {
	if s.cmd == nil {
		return fmt.Errorf("not running")
	}
	go func() { s.cmd.Process.Kill() }()
	return s.cmd.Wait()
}
