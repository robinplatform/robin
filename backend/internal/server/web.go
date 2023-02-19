package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"robin.dev/internal/log"
)

func ReverseProxy() gin.HandlerFunc {
	remote, err := url.Parse("http://localhost:9001")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

	return func(c *gin.Context) {
		logger.Debug(fmt.Sprintf("Req proxied to dev server with path=%s", c.Request.URL.Path), log.Ctx{
			"uri": c.Request.URL.Path,
		})

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
