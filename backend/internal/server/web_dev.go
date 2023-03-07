//go:build !prod

package server

import (
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"syscall"
)

func (server *Server) loadRoutes() {
	remote, err := url.Parse("http://localhost:9001")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ErrorLog = stdlog.New(io.Discard, "", 0)
	proxy.ErrorHandler = func(res http.ResponseWriter, req *http.Request, err error) {
		res.WriteHeader(http.StatusBadGateway)

		if errors.Is(err, syscall.ECONNREFUSED) {
			res.Write([]byte("Next.js dev server is not running."))
		} else {
			res.Write([]byte(fmt.Sprintf("Could not reach Next.js dev server: %v", err)))
		}
	}
	server.webRouter = proxy
}
