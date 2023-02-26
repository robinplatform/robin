//go:build !prod

package server

import (
	"io"
	stdlog "log"
	"net/http/httputil"
	"net/url"
)

func (server *Server) loadRoutes() {
	remote, err := url.Parse("http://localhost:9001")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ErrorLog = stdlog.New(io.Discard, "", 0)
	server.webRouter = proxy
}
