package compile

import (
	_ "embed"
	"fmt"
	"path"
	"strings"
	"sync"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

//go:embed client.html
var clientHtml string

//go:embed client.tsx
var clientJsBootstrap string

//go:embed not-found.html
var clientNotFoundHtml string

var logger log.Logger = log.New("compiler")

type Compiler struct {
	mux      sync.Mutex
	appCache map[string]*App
}

type App struct {
	Id   string
	Html string
}

func (compiler *Compiler) GetApp(id string) *App {
	compiler.mux.Lock()
	defer compiler.mux.Unlock()

	if app, found := compiler.appCache[id]; found {
		return app
	}

	if compiler.appCache == nil {
		compiler.appCache = make(map[string]*App)
	}

	// TODO: Check if ID is valid

	app := &App{
		Id:   id,
		Html: strings.Replace(clientHtml, "__APP_SCRIPT_URL__", "/app-resources/"+id+"/bootstrap.js", -1),
	}
	compiler.appCache[id] = app

	return app
}

func GetNotFoundHtml(id string) string {
	return strings.Replace(clientNotFoundHtml, "__APP_ID__", id, -1)
}

func (app *App) GetClientJs() (string, error) {
	appConfig, err := config.LoadRobinAppById(app.Id)
	if err != nil {
		return "", err
	}

	result := es.Build(es.BuildOptions{
		Stdin: &es.StdinOptions{
			Contents:   clientJsBootstrap,
			ResolveDir: path.Dir(appConfig.Page),
			Loader:     es.LoaderTSX,
		},
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Write:    false,
		Plugins: []es.Plugin{
			{
				Name: "load-robin-app-entrypoint",
				Setup: func(build es.PluginBuild) {
					build.OnResolve(es.OnResolveOptions{Filter: "__robinplatform-app-client-entrypoint__"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
						return es.OnResolveResult{Path: args.Path, Namespace: "robin-app-entrypoint"}, nil
					})
					build.OnLoad(es.OnLoadOptions{Filter: ".", Namespace: "robin-app-entrypoint"}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
						buf, err := appConfig.ReadFile(appConfig.Page)
						if err != nil {
							return es.OnLoadResult{}, err
						}

						return es.OnLoadResult{Contents: &buf, Loader: es.LoaderTSX}, nil
					})
				},
			},
		},
	})

	if len(result.Errors) != 0 {
		errors := make([]string, len(result.Errors))
		for i, err := range result.Errors {
			if err.PluginName == "" {
				errors[i] = err.Text
			} else {
				errors[i] = fmt.Sprintf("%s: %s", err.PluginName, err.Text)
			}
		}

		logger.Warn("Failed to compile extension", log.Ctx{
			"id":         app.Id,
			"scriptPath": appConfig.Page,
			"errors":     errors,
		})

		err := result.Errors[0]

		errMessage := err.Text
		if len(result.Errors) > 1 {
			errMessage = fmt.Sprintf("%s (and %d more errors)", errMessage, len(result.Errors)-1)
		}

		if err.PluginName != "" {
			return "", fmt.Errorf("%s: %s", err.PluginName, errMessage)
		}
		return "", fmt.Errorf("%s", errMessage)
	}

	output := result.OutputFiles[0]
	return string(output.Contents), nil
}

func (app *App) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
