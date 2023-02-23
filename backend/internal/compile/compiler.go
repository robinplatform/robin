package compile

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"text/template"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

//go:embed client.html
var clientHtmlTemplateRaw string

var clientHtmlTemplate = template.Must(template.New("robinAppClientHtml").Parse(clientHtmlTemplateRaw))

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

// TODO: The cache accesses here are not thread safe

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

	appConfig, err := config.LoadRobinAppById(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	// TODO: this is stupid, we should just expose methods that render directly
	// onto the response
	htmlOutput := bytes.NewBuffer(nil)
	if err := clientHtmlTemplate.Execute(htmlOutput, map[string]any{
		"AppConfig": appConfig,
		"ScriptURL": fmt.Sprintf("/app-resources/%s/bootstrap.js", id),
	}); err != nil {
		return nil, fmt.Errorf("failed to render client html: %w", err)
	}

	// TODO: If we are going to render the JS at the same time, might as well inline it
	clientJs, err := getClientJs(id)
	if err != nil {
		return nil, err
	}

	// TODO: Make this API actually make sense
	app := &App{
		Id:       id,
		Html:     htmlOutput.String(),
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
					if importerUrl.Scheme != "https" {
						return es.OnResolveResult{}, fmt.Errorf("expected source to be a valid URL: %s", importerUrl)
					}
					if importerUrl.Host != appConfig.ConfigPath.Host {
						return es.OnResolveResult{}, fmt.Errorf("expected all app imports to come from the same host: %s", importerUrl)
					}

					// Path will be something like: '/@foo/bar@version/src/index.js'
					// where the '@version' is optional. We need to extract the module name (+ version), so
					// we can look up its package.json
					importerUrlPath := strings.Split(importerUrl.Path, "/")
					importerModuleName := importerUrlPath[1]
					if importerModuleName[0] == '@' {
						importerModuleName = importerModuleName + "/" + importerUrlPath[2]
					}

					// Resolve local file paths relative to the remote importer's path
					// The logic for this almost entirely lives in the resolve package
					if args.Path[0] == '.' {
						resolver := resolve.NewHttpResolver(importerUrl)
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

					// We want to parse the pathname, which will look something like: `react/jsx-runtime`
					// The output should be the moduleName as `react` (with a possible scope name prefix), and then
					// the rest of the filepath being imported _from_ react, which is `jsx-runtime`.
					pathPieces := strings.Split(args.Path, "/")
					moduleName := pathPieces[0]
					if moduleName[0] == '@' {
						moduleName = moduleName + "/" + pathPieces[1]
					}
					moduleSourceFilePath := strings.Join(pathPieces[1:], "/")

					// To load the source of the module, we need to know the relative version of the module (unpkg supports
					// semver ranges, and will resolve the "latest" version that matches the range).
					//
					// But there is N places that the version of the module might exist. The highest priority is in the package.json
					// of the immediate importer. If that doesn't exist, the node resolution algorithm would actually look up a single
					// parent directory at a time (i.e. if foo imports bar which then imports baz, bar might satisfy a peer dep of baz
					// which is a higher priority than a version of baz in foo).
					//
					// However, it is now 4 AM and I'm not going to implement that. Instead, we will limit the search to the immediate
					// importer, and then finally hope that the robin app itself has the module as a dependency.
					//
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

					var reqPath *url.URL
					var contents []byte
					// If we're given a request to load the module itself, we can first try to use unpkg's beta module
					// export feature. Unfortunately it does not work with TS files, so we can't use it to load the entire package.
					// It also refuses to work for CJS, so tons of popular packages like react also don't work with it :D
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

				// The fake loader will just return the contents of the module that we loaded in the resolver.
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

		// Instead of using `append()`, this API style allows the plugin to decide its own precendence.
		// For instance, toolkit plugins are broken down and wrap the resolver plugins.
		Plugins: getToolkitPlugins(appConfig, getResolverPlugins(pagePath, appConfig)),
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
	return string(output.Contents), nil
}

func (app *App) GetServerJs(id string) (string, error) {
	_ = id

	return "", nil
}
