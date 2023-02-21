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
var clientHtmlTemplate string

//go:embed client.tsx
var clientJsBootstrap string

//go:embed error.html
var clientErrorHtml string

var logger log.Logger = log.New("compiler")

type Compiler struct {
	mux      sync.Mutex
	appCache map[string]*App
}

type App struct {
	Id          string
	Html        string
	ClientJs    string
	BundleError error
}

func (compiler *Compiler) GetApp(id string) *App {
	compiler.mux.Lock()
	defer compiler.mux.Unlock()

	// TODO: For testing
	if id == "robin-invalid-id" {
		return nil
	}

	if app, found := c.appCache[id]; found {
		logger.Debug("Found existing app bundle", log.Ctx{
			"id": id,
		})

		// TODO: Don't want to cache until updates to the source are propagated
		_ = app
		// return app
	}

	if compiler.appCache == nil {
		compiler.appCache = make(map[string]*App)
	}

	// TODO: Check if ID is valid

	clientHtml := clientHtmlTemplate
	clientHtml = strings.Replace(clientHtml, "__APP_SCRIPT_URL__", "/app-resources/"+id+"/bootstrap.js", -1)
	clientHtml = strings.Replace(clientHtml, "__APP_ID__", id, -1)

	clientJs, err := getClientJs(id)

	// TODO: Make this API actually make sense
	app := &App{
		Id:          id,
		Html:        clientHtml,
		ClientJs:    clientJs,
		BundleError: err,
	}
	compiler.appCache[id] = app

	return app
}

func GetNotFoundHtml(id string) string {
	text := `App not found: "` + id + `" is an invalid ID`
	return strings.Replace(clientErrorHtml, "__ERROR_TEXT__", text, -1)
}

func GetErrorHtml(err error) string {
	return strings.Replace(clientErrorHtml, "__ERROR_TEXT__", err.Error(), -1)
}

func getClientJs(id string) (string, error) {
	projectPath, err := config.GetProjectPath()
	if err != nil {
		return "", err
	}

	result := es.Build(es.BuildOptions{
		Stdin: &es.StdinOptions{
			Contents:   strings.Replace(clientJsBootstrap, "__SCRIPT_PATH__", appConfig.Page, -1),
			ResolveDir: path.Dir(appConfig.Page),
			Loader:     es.LoaderTSX,
		},
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Write:    false,
		Plugins:  []es.Plugin{},
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
			"projectPath": projectPath,
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
