package server

import (
	"encoding/json"
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
	router   *gin.Engine
	compiler compile.Compiler
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

var logger log.Logger = log.New("server")

func (server *Server) loadRpcMethods(group *gin.RouterGroup) {
	GetVersion.Register(server, group)
	GetConfig.Register(server, group)
	UpdateConfig.Register(server, group)

	// Apps
	GetApps.Register(server, group)
	RunAppMethod.Register(server, group)
	RestartApp.Register(server, group)
}

func createErrorJs(errMessage string) string {
	errJson, err := json.Marshal(errMessage)
	if err != nil {
		errMessage = "Unknown error occurred"
	} else {
		errMessage = string(errJson)
	}

	return fmt.Sprintf(`
		window.parent.postMessage({
			type: 'appError',
			error: %s,
		}, '*')
	`, errMessage)
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

	// Start precompiling apps, and ignore the errors for now
	// The errors will get handled when the app is requested
	go func() {
		apps, err := compile.GetAllProjectApps()
		if err != nil {
			return
		}

		for _, app := range apps {
			go server.compiler.GetApp(app.Id)
		}
	}()

	// TODO: Move the compiler routes to a separate file/into compiler
	// Apps
	server.router.GET("/app-resources/:id/base.html", func(ctx *gin.Context) {
		id := ctx.Param("id")

		app, err := server.compiler.GetApp(id)
		if err != nil {
			ctx.Header("X-Cache", "MISS")
			ctx.Data(
				http.StatusInternalServerError,
				"text/html; charset=utf-8",
				[]byte("<script>"+createErrorJs(err.Error())+"</script>"))
			return
		}

		if app.Cached {
			ctx.Header("X-Cache", "HIT")
		} else {
			ctx.Header("X-Cache", "MISS")
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(app.Html))
	})

	server.router.GET("/app-resources/:id/client.meta.json", func(ctx *gin.Context) {
		id := ctx.Param("id")

		app, err := server.compiler.GetApp(id)
		if err != nil {
			ctx.Header("X-Cache", "MISS")
			ctx.Data(
				http.StatusInternalServerError,
				"text/plain; charset=utf-8",
				[]byte(err.Error()))
			return
		}

		metafileJson, err := json.MarshalIndent(app.ClientMetafile, "", "\t")
		if err != nil {
			ctx.Header("X-Cache", "MISS")
			ctx.Data(
				http.StatusInternalServerError,
				"text/plain; charset=utf-8",
				[]byte(err.Error()))
			return
		}

		if app.Cached {
			ctx.Header("X-Cache", "HIT")
		} else {
			ctx.Header("X-Cache", "MISS")
		}
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(metafileJson))
	})

	server.router.GET("/app-resources/:id/bootstrap.js", func(ctx *gin.Context) {
		id := ctx.Param("id")

		app, err := server.compiler.GetApp(id)
		if err != nil {
			serializedErr, err := json.Marshal(err.Error())
			if err != nil {
				serializedErr = []byte(`Unknown error occurred`)
			}

			ctx.Header("X-Cache", "MISS")
			ctx.Data(
				http.StatusInternalServerError,
				"application/javascript; charset=utf-8",
				[]byte(createErrorJs(string(serializedErr))))
			return
		}

		if app.Cached {
			ctx.Header("X-Cache", "HIT")
		} else {
			ctx.Header("X-Cache", "MISS")
		}
		ctx.Data(http.StatusOK, "application/javascript; charset=utf-8", []byte(app.ClientJs))
	})

	group := server.router.Group("/api/internal/rpc")
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
