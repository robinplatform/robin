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

func (server *Server) loadRpcMethods(group *gin.RouterGroup) {
	GetVersion.Register(group)
	GetConfig.Register(group)
	UpdateConfig.Register(group)
	GetApps.Register(group)
}

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
	server.router.GET("/app-resources/:id/base.html", func(ctx *gin.Context) {
		id := ctx.Param("id")

		a := compiler.GetApp(id)
		if a == nil {
			ctx.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(compile.GetNotFoundHtml(id)))
			ctx.AbortWithStatus(404)
			return
		}

		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(a.Html))
	})

	server.router.GET("/app-resources/:id/bootstrap.js", func(ctx *gin.Context) {
		id := ctx.Param("id")

		a := compiler.GetApp(id)
		if a == nil {
			ctx.AbortWithStatus(404)
			return
		}

		markdown, err := a.GetClientJs()
		if err != nil {
			ctx.AbortWithStatus(500)
			logger.Err(err, "Failed to get ClientJS", log.Ctx{
				"id":  id,
				"err": err.Error(),
			})
		} else {
			ctx.Data(http.StatusOK, "text/javascript; charset=utf-8", []byte(markdown))
		}
	})

	group := server.router.Group("/api/rpc")
	server.loadRpcMethods(group)

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
		logger.Print(fmt.Sprintf("Started robin server on http://%s", portBinding), log.Ctx{})
	}()

	if err := server.router.Run(portBinding); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}
