package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const defaultHTTPServerShutdownTimeout = 5 * time.Second

type HTTPServerOption func(*HTTPServer)

func WithHTTPServerListener(listener net.Listener) HTTPServerOption {
	return func(server *HTTPServer) {
		server.listener = listener
	}
}

func WithHTTPServerShutdownTimeout(timeout time.Duration) HTTPServerOption {
	return func(server *HTTPServer) {
		if timeout > 0 {
			server.shutdownTimeout = timeout
		}
	}
}

type HTTPServer struct {
	server          *http.Server
	listener        net.Listener
	shutdownTimeout time.Duration
}

func NewHTTPServer(server *http.Server, opts ...HTTPServerOption) *HTTPServer {
	httpServer := &HTTPServer{
		server:          server,
		shutdownTimeout: defaultHTTPServerShutdownTimeout,
	}
	for _, opt := range opts {
		opt(httpServer)
	}

	return httpServer
}

func (s *HTTPServer) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}

	listener := s.listener
	if listener == nil {
		var listenConfig net.ListenConfig
		var err error
		listener, err = listenConfig.Listen(ctx, "tcp", s.server.Addr)
		if err != nil {
			return fmt.Errorf("listen http server: %w", err)
		}
		s.listener = listener
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()

	shutdownErr := s.server.Shutdown(shutdownCtx)
	if shutdownErr == nil {
		return nil
	}

	result := fmt.Errorf("shutdown http server: %w", shutdownErr)
	if !errors.Is(shutdownErr, context.DeadlineExceeded) {
		return result
	}

	result = errors.Join(ErrShutdownTimeout, result)
	closeErr := s.server.Close()
	if closeErr != nil {
		result = errors.Join(result, fmt.Errorf("close http server: %w", closeErr))
	}

	return result
}
