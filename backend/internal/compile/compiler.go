package compile

import (
	_ "embed"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/config"
)

//go:embed client.html
var clientHtml string

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
		scriptPath, err = filepath.Rel(projectPath, scriptPath)
		if err != nil {
			return "", err
		}
	}

	result := es.Build(es.BuildOptions{
		EntryPoints: []string{scriptPath},
		Bundle:      true,
		Platform:    es.PlatformBrowser,
		Write:       false,
		Plugins:     []es.Plugin{},
	})

	if len(result.Errors) != 0 {
		return "", fmt.Errorf("FFUFUFUFUFUU")
	}

	output := result.OutputFiles[0]

	return string(output.Contents), nil
}

func (c *Compiler) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
