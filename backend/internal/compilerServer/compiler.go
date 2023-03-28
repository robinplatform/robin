package compilerServer

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"robinplatform.dev/internal/compile/compileClient"
	"robinplatform.dev/internal/compile/compileDaemon"
	"robinplatform.dev/internal/identity"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/process"
	"robinplatform.dev/internal/project"
)

var (
	logger log.Logger = log.New("compile")

	CacheEnabled = os.Getenv("ROBIN_CACHE") != "false"
)

type Compiler struct {
	ServerPort int

	mux      sync.RWMutex
	appCache map[string]CompiledApp
}

type CompiledApp struct {
	httpClient       *http.Client
	compiler         *Compiler
	keepAliveRunning *int64
	builderMux       *sync.RWMutex

	Id        string
	ProcessId process.ProcessId

	// Html holds the HTML to be rendered on the client
	Html string

	// ClientJs holds the compiled JS bundle for the client-side app
	ClientJs string

	// ClientMetafile holds the parsed metafile for the client-side app (useful for debugging)
	ClientMetafile map[string]any

	// ServerJs holds the compiled JS bundle for the server-side app
	ServerJs string

	// serverExports maps absolute paths of server files to the functions they export
	serverExports map[string][]string
}

func (compiler *Compiler) ResetAppCache(id string) {
	compiler.mux.Lock()
	delete(compiler.appCache, id)
	compiler.mux.Unlock()
}

func (compiler *Compiler) GetApp(id string) (CompiledApp, bool, error) {
	compiler.mux.Lock()
	defer compiler.mux.Unlock()

	if app, found := compiler.appCache[id]; found {
		return app, true, nil
	}

	if compiler.appCache == nil && CacheEnabled {
		compiler.appCache = make(map[string]CompiledApp)
	}

	appConfig, err := project.LoadRobinAppById(id)
	if err != nil {
		return CompiledApp{}, false, fmt.Errorf("failed to load app config: %w", err)
	}

	projectName, err := project.GetProjectName()
	if err != nil {
		// This should have been resolved long before.
		panic(err)
	}

	processId := process.ProcessId{
		Category: identity.Category("app", projectName),
		Key:      appConfig.Id,
	}

	app := CompiledApp{
		compiler:         compiler,
		keepAliveRunning: new(int64),
		builderMux:       &sync.RWMutex{},

		Id:        id,
		ProcessId: processId,
	}

	// TODO: add something to invalidate the cache if the app's source is changed, instead of just disabling cache
	if compiler.appCache != nil && appConfig.ConfigPath.Scheme != "file" {
		compiler.appCache[id] = app
	}

	return app, false, nil
}

func (compiler *Compiler) Precompile(id string) {
	if !CacheEnabled {
		return
	}

	app, _, err := compiler.GetApp(id)
	if err != nil {
		return
	}

	go app.buildClient()

	if app.IsAlive() && atomic.CompareAndSwapInt64(app.keepAliveRunning, 0, 1) {
		go app.keepAlive()
	}
}

func (compiler *Compiler) RenderClient(id string, res http.ResponseWriter) error {
	app, cached, err := compiler.GetApp(id)
	if err != nil {
		return err
	}

	if err := app.buildClient(); err != nil {
		return err
	}

	if res != nil {
		if cached {
			res.Header().Set("X-Cache", "HIT")
		} else {
			res.Header().Set("X-Cache", "MISS")
		}

		res.Write([]byte(app.Html))
	}

	return nil
}

func (compiler *Compiler) GetClientMetaFile(id string) (map[string]any, error) {
	if err := compiler.RenderClient(id, nil); err != nil {
		return nil, err
	}

	app := compiler.appCache[id]
	return app.ClientMetafile, nil
}

func (app *CompiledApp) GetConfig() (project.RobinAppConfig, error) {
	return project.LoadRobinAppById(app.Id)
}

func (app *CompiledApp) getEnvConstants() map[string]string {
	return map[string]string{
		"process.env.ROBIN_SERVER_PORT": strconv.FormatInt(int64(app.compiler.ServerPort), 10),
		"process.env.ROBIN_APP_ID":      `"` + app.Id + `"`,
	}
}

func (app *CompiledApp) buildClient() error {
	app.builderMux.Lock()
	defer app.builderMux.Unlock()

	if app.ClientJs != "" {
		return nil
	}

	input := compileClient.ClientJSInput{
		AppId:           app.Id,
		HttpClient:      httpClient,
		DefineConstants: app.getEnvConstants(),
	}
	bundle, err := compileClient.BuildClientBundle(input)
	if err != nil {
		return err
	}

	app.ClientJs = bundle.JS
	app.ClientMetafile = bundle.Metafile
	app.Html = bundle.Html
	app.serverExports = bundle.ServerExports

	return nil
}

func (app *CompiledApp) buildServerBundle() error {
	app.builderMux.Lock()
	defer app.builderMux.Unlock()

	if app.ServerJs != "" {
		return nil
	}

	bundle, err := compileDaemon.BuildServerBundle(compileDaemon.ServerBundleInput{
		AppId:           app.Id,
		HttpClient:      httpClient,
		DefineConstants: app.getEnvConstants(),
		ServerExports:   app.serverExports,
	})

	if err != nil {
		return err
	}

	app.ServerJs = bundle.ServerJS

	return nil
}
