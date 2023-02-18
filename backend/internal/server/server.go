package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func (server *Server) loadRoutes() {
}

func (server *Server) Run(portBinding string) {
	if server.router == nil {
		// TODO: More reasonable defaults?
		server.router = gin.New()
		server.router.Use(gin.Recovery())
		server.router.SetTrustedProxies(nil)
		server.loadRoutes()
	}

	fmt.Printf("Starting robin on http://%s\n", portBinding)
	server.router.Run(portBinding)
}
