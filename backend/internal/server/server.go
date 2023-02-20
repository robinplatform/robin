package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/config"
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
	logger.Print("Starting robin", log.Ctx{
		"projectPath": config.GetProjectPathOrExit(),
		"pid":         os.Getpid(),
	})

	if server.router == nil {
		// TODO: More reasonable defaults?
		server.router = gin.New()
		server.router.Use(gin.Recovery())
		server.router.SetTrustedProxies(nil)

		server.loadRoutes()
	}

	var compiler compile.Compiler

	// Apps
	server.router.GET("/app-resources/html/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")

		markdown, err := compiler.GetClientHtml(id)
		if err != nil {
			ctx.AbortWithStatus(404)
			logger.Err(err, "Ooooops", log.Ctx{
				"id": id,
			})
		} else {
			ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(markdown))
		}
	})

	server.router.GET("/app-resources/js/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")

		logger.Debug("Hello", log.Ctx{})

		markdown, err := compiler.GetClientJs(id)
		if err != nil {
			ctx.AbortWithStatus(404)
			logger.Err(err, "Ooooops "+err.Error(), log.Ctx{
				"id": id,
			})
		} else {
			ctx.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(markdown))
		}
	})

	group := server.router.Group("/api/rpc")
	GetVersion.Register(group)
	GetConfig.Register(group)
	UpdateConfig.Register(group)

	// TODO: Switch to using net/http for the server, and let
	// gin be the router

	fmt.Printf("Starting server ...\r")
	go func() {
		healthCheck := health.HttpHealthCheck{
			Method: "GET",
			Url:    fmt.Sprintf("http://%s", portBinding),
		}
		for !health.CheckHttp(healthCheck) {
			time.Sleep(1 * time.Second)
		}
		logger.Print(fmt.Sprintf("Started robin server on http://%s\n", portBinding), log.Ctx{})
	}()

	if err := server.router.Run(portBinding); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}
