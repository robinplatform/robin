package server

import (
	"io"
	stdlog "log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"robinplatform.dev/internal/log"
)

func ReverseProxy() gin.HandlerFunc {
	remote, err := url.Parse("http://localhost:9001")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ErrorLog = stdlog.New(io.Discard, "", 0)

	return func(c *gin.Context) {
		defer func() {
			// https://github.com/gin-gonic/gin/issues/1714
			if err := recover(); err != nil && err != http.ErrAbortHandler {
				logger.Err(nil, "Error during proxying", log.Ctx{
					"err": err,
				})
			}
		}()

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (server *Server) loadRoutes() {
	//Create a catchall route
	server.router.NoRoute(ReverseProxy())
}
