package compile

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

//go:embed client.html
var clientHtml string

//go:embed client.tsx
var clientJsBootstrap string

var logger log.Logger = log.New("compiler")

type Compiler struct {
}

type App struct {
	Id string
}

func (c *Compiler) GetApp(id string) (*App, error) {
	return &App{Id: id}, nil
}

func (a *App) GetClientHtml() (string, error) {
	text := strings.Replace(clientHtml, "__APP_ID__", a.Id, -1)

	return text, nil
}

func (a *App) GetClientJs() (string, error) {
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
			Contents: strings.Replace(clientJsBootstrap, "__SCRIPT_PATH__", scriptPath, -1),

			// These are all optional:
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
			"id":          a.Id,
			"projectPath": projectPath,
			"scriptPath":  scriptPath,
			"errors":      result.Errors,
		})

		e := result.Errors[0]
		return "", fmt.Errorf("%s,%s: %s", e.Location.File, e.Location.LineText, e.Text)
	}

	// TODO: Output all files in the case of more crazy bundling things
	output := result.OutputFiles[0]

	return string(output.Contents), nil
}

func (a *App) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
