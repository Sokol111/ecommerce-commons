package server

import (
	"net/http"
)

// NewServer creates an HTTP server that supports both HTTP/1.1 and H2C
// (unencrypted HTTP/2). H2C is required for native gRPC clients to connect
// without TLS.
func NewServer(conf Config, handler http.Handler) *http.Server {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: conf.Connection.ReadHeaderTimeout,
		ReadTimeout:       conf.Connection.ReadTimeout,
		WriteTimeout:      conf.Connection.WriteTimeout,
		IdleTimeout:       conf.Connection.IdleTimeout,
		MaxHeaderBytes:    conf.Connection.MaxHeaderBytes,
	}
	srv.Protocols = &http.Protocols{}
	srv.Protocols.SetHTTP1(true)
	srv.Protocols.SetHTTP2(true)
	srv.Protocols.SetUnencryptedHTTP2(true)
	return srv
}
