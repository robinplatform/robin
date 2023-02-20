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

func (c *Compiler) GetClientHtml(id string) (string, error) {
	text := strings.Replace(clientHtml, "__APP_ID__", id, -1)

	return text, nil
}

func (c *Compiler) GetClientJs(id string) (string, error) {
	_ = id

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
		for _, message := range result.Errors {
			logger.Debug("OOOOFFEFEFEFE "+message.Text+": "+message.Location.File+":"+message.Location.LineText, log.Ctx{})

		}

		return "", fmt.Errorf("FFUFUFUFUFUU")
	}

	output := result.OutputFiles[0]

	return string(output.Contents), nil
}

func (c *Compiler) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
