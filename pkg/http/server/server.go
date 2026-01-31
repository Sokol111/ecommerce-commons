package server

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

// Server defines the HTTP server interface.
type Server interface {
	Serve() error
	ServeWithReadyCallback(onReady func()) error
	Shutdown(ctx context.Context) error
}

type server struct {
	httpSrv *http.Server
	log     *zap.Logger
}

func newServer(log *zap.Logger, conf Config, handler http.Handler) Server {
	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(conf.Port),
		Handler:           handler,
		ReadHeaderTimeout: conf.Connection.ReadHeaderTimeout,
		ReadTimeout:       conf.Connection.ReadTimeout,
		WriteTimeout:      conf.Connection.WriteTimeout,
		IdleTimeout:       conf.Connection.IdleTimeout,
		MaxHeaderBytes:    conf.Connection.MaxHeaderBytes,
	}
	return &server{
		httpSrv: srv,
		log:     log,
	}
}

func (s *server) Serve() error {
	return s.ServeWithReadyCallback(nil)
}

func (s *server) ServeWithReadyCallback(onReady func()) error {
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		s.log.Error("failed to listen", zap.Error(err))
		return err
	}
	s.log.Info("starting HTTP server at", zap.String("addr", s.httpSrv.Addr))

	// Signal that server is ready to accept connections
	if onReady != nil {
		onReady()
	}

	if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
		s.log.Error("HTTP server stopped with error", zap.Error(err))
		return err
	}
	return nil
}

func (s *server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
