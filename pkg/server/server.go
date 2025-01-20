package server

import (
	"context"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

type ServerConf struct {
	Port int `mapstructure:"port"`
}

type Server struct {
	httpServer  *http.Server
	baseContext context.Context
	serverConf  *ServerConf
}

func NewServer(conf *ServerConf, baseContext context.Context, handler http.Handler) *Server {
	s := &http.Server{
		Addr:    ":" + strconv.Itoa(conf.Port),
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			return baseContext
		},
	}

	return &Server{s, baseContext, conf}
}

func (s *Server) Start() {
	g, gCtx := errgroup.WithContext(s.baseContext)

	g.Go(func() error {
		slog.Info("listening and serving HTTP", slog.String("addr", s.httpServer.Addr))
		return s.httpServer.ListenAndServe()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return s.httpServer.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		slog.Error("exited", slog.Any("error", err))
	}
}
