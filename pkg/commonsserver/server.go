package commonsserver

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

type ServerInterface interface {
	Serve() error
	Shutdown(ctx context.Context) error
}

type Server struct {
	httpSrv *http.Server
	log     *zap.Logger
}

func NewServer(log *zap.Logger, conf ServerConf, handler http.Handler) *Server {
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(conf.Port),
		Handler: handler,
	}
	return &Server{
		httpSrv: srv,
		log:     log,
	}
}

func (s *Server) Serve() error {
	ln, err := net.Listen("tcp", s.httpSrv.Addr)
	if err != nil {
		s.log.Error("failed to listen", zap.Error(err))
		return err
	}
	s.log.Info("starting HTTP server at", zap.String("addr", s.httpSrv.Addr))
	return s.httpSrv.Serve(ln)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}
