package compile

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
)

func (appConfig *RobinAppConfig) getCssLoaderPlugins(plugins []es.Plugin) []es.Plugin {
	plugins = append(plugins, es.Plugin{
		Name: "load-css",
		Setup: func(build es.PluginBuild) {
			build.OnLoad(es.OnLoadOptions{
				Filter: "\\.css(\\?bundle)?$",
			}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
				if args.Namespace == "robin-toolkit" {
					return es.OnLoadResult{}, nil
				}

				var css []byte
				var err error

				if strings.HasPrefix(args.Path, "http://") || strings.HasPrefix(args.Path, "https://") {
					_, css, err = appConfig.ReadFile(args.Path)
				} else {
					css, err = os.ReadFile(args.Path)
				}
				if err != nil {
					return es.OnLoadResult{}, fmt.Errorf("failed to read css file %s: %w", args.Path, err)
				}

				script := fmt.Sprintf(`!function(){
							let style = document.createElement('style')
							style.setAttribute('data-path', '%s')
							style.innerText = %q
							document.body.appendChild(style)
						}()`, args.Path, string(css))
				return es.OnLoadResult{
					Contents: &script,
					Loader:   es.LoaderJS,
				}, nil
			})
		},
	}, es.Plugin{
		Name: "load-scss",
		Setup: func(build es.PluginBuild) {
			build.OnLoad(es.OnLoadOptions{
				Filter: "\\.scss(\\?bundle)?$",
			}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
				if args.Namespace == "robin-toolkit" {
					return es.OnLoadResult{}, nil
				}

				var sass []byte
				var err error

				if strings.HasPrefix(args.Path, "http://") || strings.HasPrefix(args.Path, "https://") {
					_, sass, err = appConfig.ReadFile(args.Path)
				} else {
					sass, err = os.ReadFile(args.Path)
				}
				if err != nil {
					return es.OnLoadResult{}, fmt.Errorf("failed to read sass file %s: %w", args.Path, err)
				}

				script, err := buildSass(args.Path, string(sass))
				if err != nil {
					return es.OnLoadResult{}, fmt.Errorf("failed to build sass file %s: %w", args.Path, err)
				}

				return es.OnLoadResult{
					Contents: &script,
					Loader:   es.LoaderJS,
				}, nil
			})
		},
	})
	return plugins
}

func buildSass(srcPath, sass string) (string, error) {
	result := es.Build(es.BuildOptions{
		Stdin: &es.StdinOptions{
			Contents: fmt.Sprintf(`
				import sass from 'https://esm.sh/sass@1.58.3?bundle&target=esnext'

				let style = document.createElement('style')
				style.setAttribute('data-path', '%s')
				style.innerText = sass.compileString(%q).css
				document.body.appendChild(style)
			`, srcPath, string(sass)),
			Loader: es.LoaderTS,
		},
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Format:   es.FormatIIFE,
		Write:    false,
		Define: map[string]string{
			"process.stdout.isTTY": "false",
		},
		Plugins: []es.Plugin{
			esbuildPluginLoadHttp,
		},
	})
	if len(result.Errors) > 0 {
		return "", BuildError(result)
	}
	if len(result.OutputFiles) != 1 {
		return "", fmt.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	buf := bytes.ReplaceAll(result.OutputFiles[0].Contents, []byte("process.stdout.isTTY"), []byte("false"))
	return string(buf), nil
}
