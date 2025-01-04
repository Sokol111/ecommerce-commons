package server

import (
	"context"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

type Server struct {
	httpServer  *http.Server
	baseContext context.Context
}

//func NewServer(port int, baseContext context.Context, handlers ...Handler) *Server {
//	engine := gin.Default()
//	engine.Use(errorHandler)
//
//	for _, h := range handlers {
//		h.BindRoutes(engine)
//	}
//
//	s := &http.Server{
//		Addr:    ":" + strconv.Itoa(port),
//		Handler: engine,
//		BaseContext: func(_ net.Listener) context.Context {
//			return baseContext
//		},
//	}
//
//	return &Server{s, baseContext}
//}

func NewServer(port int, baseContext context.Context, handler http.Handler) *Server {
	s := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: handler,
		BaseContext: func(_ net.Listener) context.Context {
			return baseContext
		},
	}

	return &Server{s, baseContext}
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
		slog.Error("exit reason", slog.Any("error", err))
	}
}
