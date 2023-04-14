package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	"github.com/julienschmidt/httprouter"
	"robinplatform.dev/internal/compilerServer"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/project"
)

type Server struct {
	BindAddress string
	Port        int
	EnablePprof bool

	router      *httprouter.Router
	pprofRouter http.Handler
	webRouter   http.Handler
	compiler    compilerServer.Compiler
}

var logger log.Logger = log.New("server")

type RouterGroup struct {
	router *httprouter.Router
	prefix string
}

func (routerGroup *RouterGroup) Handle(method, path string, handler httprouter.Handle) {
	routerGroup.router.Handle(method, routerGroup.prefix+path, handler)
}

type InternalRpcMethod[Input any, Output any] RpcMethod[Input, Output]

func (method *InternalRpcMethod[Input, Output]) Register(server *Server) {
	(*RpcMethod[Input, Output])(method).Register(server, RouterGroup{
		router: server.router,
		prefix: "/api/internal/rpc",
	})
}

type AppsRpcMethod[Input any, Output any] RpcMethod[Input, Output]

func (method *AppsRpcMethod[Input, Output]) Register(server *Server) {
	(*RpcMethod[Input, Output])(method).Register(server, RouterGroup{
		router: server.router,
		prefix: "/api/apps/rpc",
	})
}

func (server *Server) loadRpcMethods() {
	// Internal RPC methods

	GetVersion.Register(server)
	GetConfig.Register(server)
	UpdateConfig.Register(server)

	GetProcessLogs.Register(server)

	GetAppById.Register(server)
	GetApps.Register(server)
	RunAppMethod.Register(server)
	RestartApp.Register(server)
	ListProcesses.Register(server)

	// Apps RPC methods

	GetAppSettingsById.Register(server)
	UpdateAppSettings.Register(server)
	GetTopics.Register(server)
	CreateTopic.Register(server)
	PublishTopic.Register(server)

	StartProcessForApp.Register(server)
	StopProcessForApp.Register(server)
	CheckProcessHealth.Register(server)

	// Streaming methods

	wsHandler := &RpcWebsocket{}
	server.router.GET("/api/websocket", wsHandler.WebsocketHandler(server))

	SubscribeTopic.Register(wsHandler)
	SubscribeAppTopic.Register(wsHandler)
}

func createErrorJs(errMessage string) string {
	return fmt.Sprintf(`
		window.parent.postMessage({
			type: 'appError',
			error: %q,
		}, '*')
	`, errMessage)
}

func (server *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if server.EnablePprof && strings.HasPrefix(req.URL.Path, "/debug/pprof/") {
		server.pprofRouter.ServeHTTP(res, req)
	} else if strings.HasPrefix(req.URL.Path, "/api") {
		server.router.ServeHTTP(res, req)
	} else {
		server.webRouter.ServeHTTP(res, req)
	}
}

func (server *Server) Run() error {
	logger.Print("Starting robin", log.Ctx{
		"projectPath": project.GetProjectPathOrExit(),
		"pid":         os.Getpid(),
	})

	if server.router == nil {
		server.router = httprouter.New()
		server.loadRoutes()
	}
	server.compiler.ServerPort = server.Port

	if server.EnablePprof {
		logger.Print("Running with pprof enabled", log.Ctx{})
		mux := http.NewServeMux()
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		server.pprofRouter = mux
	}

	// TODO: Move the compiler routes to a separate file/into compiler
	// Apps
	server.router.GET("/api/app-resources/:id/base/*filepath", func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		id := params.ByName("id")
		res.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := server.compiler.RenderClient(id, res); err != nil {
			res.Header().Set("X-Cache", "MISS")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte("<script>" + createErrorJs(err.Error()) + "</script>"))
			return
		}
	})

	server.router.GET("/api/app-resources/:id/client.meta.json", func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		id := params.ByName("id")

		metafile, err := server.compiler.GetClientMetaFile(id)
		if err != nil {
			res.Header().Set("Content-Type", "text/plain; charset=utf-8")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(err.Error()))
			return
		}

		metafileJson, err := json.MarshalIndent(metafile, "", "\t")
		if err != nil {
			res.Header().Set("Content-Type", "text/plain; charset=utf-8")
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(err.Error()))
			return
		}

		res.WriteHeader(http.StatusOK)
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		res.Write(metafileJson)
	})

	server.loadRpcMethods()
	portBinding := fmt.Sprintf("%s:%d", server.BindAddress, server.Port)

	fmt.Printf("Starting server ...\r")

	listener, err := net.Listen("tcp", portBinding)
	if err != nil {
		return fmt.Errorf("failed to start server: %s", err)
	}
	logger.Print(fmt.Sprintf("Started robin server on http://%s", portBinding), log.Ctx{})

	httpServer := http.Server{
		Handler: server,
	}
	httpServer.Serve(listener)
	return nil
}
