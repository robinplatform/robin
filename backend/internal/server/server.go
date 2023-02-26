package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/health"
	"robinplatform.dev/internal/log"
)

type Server struct {
	router    *httprouter.Router
	webRouter http.Handler
	compiler  compile.Compiler
}

var logger log.Logger = log.New("server")

type RouterGroup struct {
	router *httprouter.Router
	prefix string
}

func (routerGroup *RouterGroup) Handle(method, path string, handler httprouter.Handle) {
	routerGroup.router.Handle(method, routerGroup.prefix+path, handler)
}

func (server *Server) loadRpcMethods(group RouterGroup) {
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

func (server *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/api") {
		server.router.ServeHTTP(res, req)
	} else {
		server.webRouter.ServeHTTP(res, req)
	}
}

func (server *Server) Run(portBinding string) error {
	logger.Print("Starting robin", log.Ctx{
		"projectPath": config.GetProjectPathOrExit(),
		"pid":         os.Getpid(),
	})

	if server.router == nil {
		server.router = httprouter.New()
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
	server.router.GET("/api/app-resources/:id/base.html", func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		res.Header().Set("Content-Type", "text/html; charset=utf-8")

		app, err := server.compiler.GetApp(id)
		if err != nil {
			res.Header().Set("X-Cache", "MISS")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte("<script>" + createErrorJs(err.Error()) + "</script>"))
			return
		}

		if app.Cached {
			res.Header().Set("X-Cache", "HIT")
		} else {
			res.Header().Set("X-Cache", "MISS")
		}
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(app.Html))
	})

	server.router.GET("/api/app-resources/:id/client.meta.json", func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		id := params.ByName("id")

		app, err := server.compiler.GetApp(id)
		if err != nil {
			res.Header().Set("X-Cache", "MISS")
			res.Header().Set("Content-Type", "text/plain; charset=utf-8")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(err.Error()))
			return
		}

		metafileJson, err := json.MarshalIndent(app.ClientMetafile, "", "\t")
		if err != nil {
			res.Header().Set("X-Cache", "MISS")
			res.Header().Set("Content-Type", "text/plain; charset=utf-8")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(err.Error()))
			return
		}

		if app.Cached {
			res.Header().Set("X-Cache", "HIT")
		} else {
			res.Header().Set("X-Cache", "MISS")
		}
		res.WriteHeader(http.StatusOK)
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		res.Write(metafileJson)
	})

	server.router.GET("/api/app-resources/:id/bootstrap.js", func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		id := params.ByName("id")

		app, err := server.compiler.GetApp(id)
		if err != nil {
			serializedErr, err := json.Marshal(err.Error())
			if err != nil {
				serializedErr = []byte(`Unknown error occurred`)
			}

			res.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(createErrorJs(string(serializedErr))))
			return
		}

		if app.Cached {
			res.Header().Set("X-Cache", "HIT")
		} else {
			res.Header().Set("X-Cache", "MISS")
		}
		res.WriteHeader(http.StatusOK)
		res.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		res.Write([]byte(app.ClientJs))
	})

	server.loadRpcMethods(RouterGroup{
		router: server.router,
		prefix: "/api/internal/rpc",
	})

	// TODO: Simplify this

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

	if err := http.ListenAndServe(portBinding, server); err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	return nil
}
