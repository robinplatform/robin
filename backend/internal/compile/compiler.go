package compile

import (
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
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

func getResolverPlugins(pageSourceUrl *url.URL, appConfig config.RobinAppConfig) []es.Plugin {
	if appConfig.ConfigPath.Scheme == "file" {
		return nil
	}

	return []es.Plugin{
		{
			Name: "robin-resolver",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{Filter: "."}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					// If we're resolving a module from the virtual toolkit, we should assume that the extension
					// itself asked for it
					if args.Namespace == "robin-toolkit" {
						args.Importer = pageSourceUrl.String()
					}

					importerUrl, err := url.Parse(args.Importer)
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("expected source to be a valid URL: %w", err)
					}
					if importerUrl.Scheme != "http" && importerUrl.Scheme != "https" {
						return es.OnResolveResult{}, fmt.Errorf("expected source to be a valid URL: %s", importerUrl)
					}

					// Path will be something like: '/@foo/bar@version/src/index.js'
					// where the '@version' is optional. We need to extract the module name (+ version), so
					// we can look up its package.json
					importerUrlPath := strings.Split(importerUrl.Path, "/")
					importerModuleName := importerUrlPath[1]
					if importerModuleName[0] == '@' {
						importerModuleName = importerModuleName + "/" + importerUrlPath[2]
					}

					// Resolve local files
					if args.Path[0] == '.' {
						resolver := resolve.NewHttpResolver(importerUrl)
						resolver.EnableDebugLogs = true

						resolved, err := resolver.ResolveFrom(importerUrl.Path, args.Path)
						if err != nil {
							return es.OnResolveResult{}, fmt.Errorf("could not resolve '%s': %w", args.Path, err)
						}

						reqPath, contents, err := appConfig.ReadFile("/" + resolved)
						if err != nil {
							return es.OnResolveResult{}, fmt.Errorf("could not read '%s': %w", resolved, err)
						}

						logger.Debug("Resolved local file for remote module", log.Ctx{
							"importer":     args.Importer,
							"path":         args.Path,
							"resolvedPath": reqPath.String(),
						})
						return es.OnResolveResult{
							Namespace: "robin-resolver",
							Path:      reqPath.String(),
							PluginData: map[string]string{
								"contents": string(contents),
							},
						}, nil
					}

					// Resolve modules

					pathPieces := strings.Split(args.Path, "/")
					moduleName := pathPieces[0]
					if moduleName[0] == '@' {
						moduleName = moduleName + "/" + pathPieces[1]
					}
					moduleSourceFilePath := strings.Join(pathPieces[1:], "/")

					var moduleVersion string

					for _, packageJsonPath := range []string{fmt.Sprintf("/%s/package.json", importerModuleName), "package.json"} {
						_, rawPackageJson, err := appConfig.ReadFile(packageJsonPath)
						if err != nil {
							return es.OnResolveResult{}, err
						}

						var packageJson config.PackageJson
						if err := config.ParsePackageJson(rawPackageJson, &packageJson); err != nil {
							return es.OnResolveResult{}, err
						}

						var found bool
						moduleVersion, found = packageJson.Dependencies[moduleName]
						if found {
							break
						}
					}
					if moduleVersion == "" {
						return es.OnResolveResult{}, fmt.Errorf("cannot resolve module '%s' (not found in package.json)", moduleName)
					}

					// TODO: This is really silly, but it works for now
					var reqPath *url.URL
					var contents []byte
					if moduleSourceFilePath == "" {
						reqPath, contents, err = appConfig.ReadFile(fmt.Sprintf("/%s@%s?module", moduleName, moduleVersion))
						if err != nil {
							reqPath, contents, err = appConfig.ReadFile(fmt.Sprintf("/%s@%s", moduleName, moduleVersion))
						}
					} else {
						reqPath, contents, err = appConfig.ReadFile(fmt.Sprintf("/%s@%s/%s", moduleName, moduleVersion, moduleSourceFilePath))
					}
					if err != nil {
						return es.OnResolveResult{}, err
					}

					logger.Debug("Resolved remote module for remote module", log.Ctx{
						"importer":             args.Importer,
						"path":                 args.Path,
						"moduleName":           moduleName,
						"moduleVersion":        moduleVersion,
						"moduleSourceFilePath": moduleSourceFilePath,
						"resolvedPath":         reqPath.String(),
					})
					return es.OnResolveResult{
						Namespace: "robin-resolver",
						Path:      reqPath.String(),
						PluginData: map[string]string{
							"moduleName": moduleName,
							"version":    moduleVersion,
							"contents":   string(contents),
						},
					}, nil
				})

				build.OnLoad(es.OnLoadOptions{
					Namespace: "robin-resolver",
					Filter:    ".",
				}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
					pluginData, ok := args.PluginData.(map[string]string)
					if !ok {
						return es.OnLoadResult{}, fmt.Errorf("invalid plugin data")
					}

					contents := pluginData["contents"]
					return es.OnLoadResult{
						Contents: &contents,
					}, nil
				})
			},
		},
	}
}

func getClientJs(id string) (string, error) {
	appConfig, err := config.LoadRobinAppById(id)
	if err != nil {
		return "", err
	}

	pagePath, content, err := appConfig.ReadFile(appConfig.Page)
	if err != nil {
		return "", err
	}

	stdinOptions := es.StdinOptions{
		Contents:   string(content),
		Sourcefile: pagePath.String(),
		Loader:     es.LoaderTSX,
	}
	if appConfig.ConfigPath.Scheme == "file" {
		stdinOptions.ResolveDir = path.Dir(appConfig.ConfigPath.Path)
	}

	result := es.Build(es.BuildOptions{
		Stdin:    &stdinOptions,
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Write:    false,
		Plugins:  getToolkitPlugins(appConfig, getResolverPlugins(pagePath, appConfig)),
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
