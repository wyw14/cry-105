package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Server struct {
	http     *http.Server
	listener net.Listener
}

func Listen(address string, runtime *Runtime) (*Server, error) {
	handler, err := NewRouter(runtime)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}
	server := &Server{http: &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}, listener: listener}
	go func() { _ = server.http.Serve(listener) }()
	return server, nil
}

func (s *Server) Address() string                 { return s.listener.Addr().String() }
func (s *Server) Close(ctx context.Context) error { return s.http.Shutdown(ctx) }
