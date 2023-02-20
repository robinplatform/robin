package compile

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
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
	m        sync.Mutex
	appCache map[string]*App
}

type App struct {
	Id          string
	Html        string
	ClientJs    string
	BundleError error
}

func (c *Compiler) GetApp(id string) *App {
	c.m.Lock()
	defer c.m.Unlock()

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

	if c.appCache == nil {
		c.appCache = make(map[string]*App)
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
	c.appCache[id] = app

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

	packageJsonPath := path.Join(projectPath, "package.json")
	var packageJson config.PackageJson
	if err := config.LoadPackageJson(packageJsonPath, &packageJson); err != nil {
		return "", err
	}

	scriptPath := packageJson.Robin
	if !filepath.IsAbs(scriptPath) {
		scriptPath = path.Clean(path.Join(projectPath, scriptPath))
	}

	result := es.Build(es.BuildOptions{
		Stdin: &es.StdinOptions{
			Contents:   strings.Replace(clientJsBootstrap, "__SCRIPT_PATH__", scriptPath, -1),
			ResolveDir: path.Dir(scriptPath),
			Loader:     es.LoaderTSX,
		},
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Write:    false,
		Plugins:  []es.Plugin{},
	})

	if len(result.Errors) != 0 {
		logger.Info("Failed to compile extension", log.Ctx{
			"id":          id,
			"projectPath": projectPath,
			"scriptPath":  scriptPath,
			"errors":      result.Errors,
		})

		e := result.Errors[0]
		return "", fmt.Errorf("%v,%v: %v", e.Location.File, e.Location.Line, e.Text)
	}

	// TODO: Output all files in the case of more crazy bundling things
	output := result.OutputFiles[0]

	return string(output.Contents), nil
}

func (a *App) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
