package compile

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

//go:embed client.html
var clientHtmlTemplate string

var logger log.Logger = log.New("compiler")

var cacheEnabled = os.Getenv("ROBIN_CACHE") != "false"

type Compiler struct {
	mux      sync.Mutex
	appCache map[string]*App
}

type App struct {
	Id       string
	Html     string
	ClientJs string
}

func (compiler *Compiler) GetApp(id string) (*App, error) {
	compiler.mux.Lock()
	defer compiler.mux.Unlock()

	if app, found := compiler.appCache[id]; found {
		logger.Debug("Found existing app bundle", log.Ctx{
			"id": id,
		})
		return app, nil
	}

	if compiler.appCache == nil && cacheEnabled {
		compiler.appCache = make(map[string]*App)
	}

	// TODO: Check if ID is valid
	appConfig, err := config.LoadRobinAppById(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	clientHtml := clientHtmlTemplate
	clientHtml = strings.Replace(clientHtml, "__APP_SCRIPT_URL__", "/app-resources/"+id+"/bootstrap.js", -1)
	clientHtml = strings.Replace(clientHtml, "__APP_NAME__", appConfig.Name, -1)

	clientJs, err := getClientJs(id)
	if err != nil {
		return nil, err
	}

	// TODO: Make this API actually make sense
	app := &App{
		Id:       id,
		Html:     clientHtml,
		ClientJs: clientJs,
	}
	if compiler.appCache != nil {
		compiler.appCache[id] = app
	}

	return app, nil
}

func getClientJs(id string) (string, error) {
	appConfig, err := config.LoadRobinAppById(id)
	if err != nil {
		return "", err
	}

	if !path.IsAbs(appConfig.Page) {
		appConfig.Page = path.Clean(path.Join(path.Dir(appConfig.ConfigPath.Path), appConfig.Page))
	}
	if _, err := os.Stat(appConfig.Page); err != nil {
		return "", fmt.Errorf("failed to find page '%s': %s", appConfig.Page, err)
	}

	result := es.Build(es.BuildOptions{
		EntryPoints: []string{appConfig.Page},
		Bundle:      true,
		Platform:    es.PlatformBrowser,
		Write:       false,
		Plugins:     append([]es.Plugin{}, getToolkitPlugins(appConfig)...),
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
			"id":         id,
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
	errorHandler := `window.addEventListener('error', (event) => {
		window.parent.postMessage({
			type: 'appError',
			error: String(event.error),
		}, '*')
	});`

	return errorHandler + string(output.Contents), nil
}

func (app *App) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
