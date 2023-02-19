package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"robinplatform.dev/internal/health"
	"robinplatform.dev/internal/log"
)

type Server struct {
	router *gin.Engine
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

var logger log.Logger = log.New("server")

func (server *Server) Run(portBinding string) error {
	if server.router == nil {
		// TODO: More reasonable defaults?
		server.router = gin.New()
		server.router.Use(gin.Recovery())
		server.router.SetTrustedProxies(nil)

		server.loadRoutes()
	}

	group := server.router.Group("/api/rpc")
	GetVersion.Register(group)
	GetConfig.Register(group)
	UpdateConfig.Register(group)

	// TODO: Switch to using net/http for the server, and let
	// gin be the router

	fmt.Printf("Starting robin ...\r")
	go func() {
		healthCheck := health.HttpHealthCheck{
			Method: "GET",
			Url:    fmt.Sprintf("http://%s", portBinding),
		}
		for !health.CheckHttp(healthCheck) {
			time.Sleep(1 * time.Second)
		}
		logger.Print(fmt.Sprintf("Started robin on http://%s\n", portBinding), log.Ctx{
			"portBinding": portBinding,
		})
	}()

	if err := server.router.Run(portBinding); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}
