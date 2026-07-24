package server

import (
	"net/http"
)

func NewServer(conf Config, handler http.Handler) *http.Server {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: conf.Connection.ReadHeaderTimeout,
		ReadTimeout:       conf.Connection.ReadTimeout,
		WriteTimeout:      conf.Connection.WriteTimeout,
		IdleTimeout:       conf.Connection.IdleTimeout,
		MaxHeaderBytes:    conf.Connection.MaxHeaderBytes,
	}
	if conf.H2C {
		srv.Protocols = &http.Protocols{}
		srv.Protocols.SetHTTP1(true)
		srv.Protocols.SetUnencryptedHTTP2(true)
	}
	return srv
}
