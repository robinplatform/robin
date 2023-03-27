package compilerServer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/buildError"
	"robinplatform.dev/internal/compile/toolkit"
	"robinplatform.dev/internal/identity"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/process"
	"robinplatform.dev/internal/project"
)

var (
	//go:embed client.html
	clientHtmlTemplateRaw string

	clientHtmlTemplate = template.Must(template.New("robinAppClientHtml").Parse(clientHtmlTemplateRaw))

	logger log.Logger = log.New("compile")

	CacheEnabled = os.Getenv("ROBIN_CACHE") != "false"
)

type Compiler struct {
	ServerPort int

	mux      sync.RWMutex
	appCache map[string]CompiledApp
}

type CompiledApp struct {
	httpClient       *http.Client
	compiler         *Compiler
	keepAliveRunning *int64
	builderMux       *sync.RWMutex

	Id        string
	ProcessId process.ProcessId

	// Html holds the HTML to be rendered on the client
	Html string

	// ClientJs holds the compiled JS bundle for the client-side app
	ClientJs string

	// ClientMetafile holds the parsed metafile for the client-side app (useful for debugging)
	ClientMetafile map[string]any

	// ServerJs holds the compiled JS bundle for the server-side app
	ServerJs string

	// serverExports maps absolute paths of server files to the functions they export
	serverExports map[string][]string
}

func (compiler *Compiler) ResetAppCache(id string) {
	compiler.mux.Lock()
	delete(compiler.appCache, id)
	compiler.mux.Unlock()
}

func (compiler *Compiler) GetApp(id string) (CompiledApp, bool, error) {
	compiler.mux.Lock()
	defer compiler.mux.Unlock()

	if app, found := compiler.appCache[id]; found {
		return app, true, nil
	}

	if compiler.appCache == nil && CacheEnabled {
		compiler.appCache = make(map[string]CompiledApp)
	}

	appConfig, err := project.LoadRobinAppById(id)
	if err != nil {
		return CompiledApp{}, false, fmt.Errorf("failed to load app config: %w", err)
	}

	projectName, err := project.GetProjectName()
	if err != nil {
		// This should have been resolved long before.
		panic(err)
	}

	processId := process.ProcessId{
		Category: identity.Category("app", projectName),
		Key:      appConfig.Id,
	}

	app := CompiledApp{
		compiler:         compiler,
		keepAliveRunning: new(int64),
		builderMux:       &sync.RWMutex{},

		Id:        id,
		ProcessId: processId,
	}

	// TODO: add something to invalidate the cache if the app's source is changed, instead of just disabling cache
	if compiler.appCache != nil && appConfig.ConfigPath.Scheme != "file" {
		compiler.appCache[id] = app
	}

	return app, false, nil
}

func (compiler *Compiler) Precompile(id string) {
	if !CacheEnabled {
		return
	}

	app, _, err := compiler.GetApp(id)
	if err != nil {
		return
	}

	go app.buildClientJs()

	if app.IsAlive() && atomic.CompareAndSwapInt64(app.keepAliveRunning, 0, 1) {
		go app.keepAlive()
	}
}

func (compiler *Compiler) RenderClient(id string, res http.ResponseWriter) error {
	app, cached, err := compiler.GetApp(id)
	if err != nil {
		return err
	}

	if app.ClientJs == "" {
		app.builderMux.Lock()
		defer app.builderMux.Unlock()
	}

	if app.ClientJs == "" {
		if err := app.buildClientJs(); err != nil {
			return err
		}

		appConfig, err := project.LoadRobinAppById(id)
		if err != nil {
			return fmt.Errorf("failed to load app config: %w", err)
		}

		htmlOutput := bytes.NewBuffer(nil)
		if err := clientHtmlTemplate.Execute(htmlOutput, map[string]any{
			"AppConfig":    appConfig,
			"ScriptSource": app.ClientJs,
		}); err != nil {
			return fmt.Errorf("failed to render client html: %w", err)
		}
		app.Html = htmlOutput.String()
	}

	if res != nil {
		if cached {
			res.Header().Set("X-Cache", "HIT")
		} else {
			res.Header().Set("X-Cache", "MISS")
		}

		res.Write([]byte(app.Html))
	}

	return nil
}

func (compiler *Compiler) GetClientMetaFile(id string) (map[string]any, error) {
	if err := compiler.RenderClient(id, nil); err != nil {
		return nil, err
	}

	app := compiler.appCache[id]
	return app.ClientMetafile, nil
}

func getResolverPlugins(appConfig project.RobinAppConfig, pageSourceUrl *url.URL) []es.Plugin {
	if appConfig.ConfigPath.Scheme == "file" {
		return nil
	}

	return []es.Plugin{
		{
			Name: "robin-resolver",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{Filter: "^[^/\\.]"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					// If we're resolving a module from the virtual toolkit, we should assume that the extension
					// itself asked for it
					if args.Namespace == toolkit.Namespace {
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

					// We want to parse the pathname, which will look something like: `react/jsx-runtime`
					// The output should be the moduleName as `react` (with a possible scope name prefix), and then
					// the rest of the filepath being imported _from_ react, which is `jsx-runtime`.
					pathPieces := strings.Split(args.Path, "/")
					moduleName := pathPieces[0]
					moduleSourceFilePath := ""
					if len(moduleName) == 0 {
						return es.OnResolveResult{}, fmt.Errorf("expected module name to be non-empty in: %s", args.Path)
					}
					if moduleName[0] == '@' {
						moduleName = moduleName + "/" + pathPieces[1]
						moduleSourceFilePath = strings.Join(pathPieces[2:], "/")
					} else {
						moduleSourceFilePath = strings.Join(pathPieces[1:], "/")
					}

					// To load the source of the module, we need to know the relative version of the module.
					//
					// But there is N places that the version of the module might exist. The highest priority is in the package.json
					// of the immediate importer. If that doesn't exist, the node resolution algorithm would actually look up a single
					// parent directory at a time (i.e. if foo imports bar which then imports baz, bar might satisfy a peer dep of baz
					// which is a higher priority than a version of baz in foo).
					//
					// However, `esm.sh` takes care of most of this anyways, so we really just need to perform lookups for modules that
					// are immediately imported by the app. So we'll just look in the package.json of the immediate importer.
					packageJsonPath, rawPackageJson, err := appConfig.ReadFile(&httpClient, "package.json")
					if err != nil {
						return es.OnResolveResult{}, err
					}

					var packageJson project.PackageJson
					if err := project.ParsePackageJson(rawPackageJson, &packageJson); err != nil {
						return es.OnResolveResult{}, err
					}

					moduleVersion, found := packageJson.Dependencies[moduleName]
					if !found {
						logger.Debug("Failed to find module version in package.json", log.Ctx{
							"packageJsonPath": packageJsonPath,
							"packageJson":     packageJson,
							"moduleName":      moduleName,
						})
						return es.OnResolveResult{}, fmt.Errorf("cannot resolve module '%s' (not found in package.json)", moduleName)
					}

					reqPath, _, err := appConfig.ReadFile(&httpClient, fmt.Sprintf("/%s@%s/%s", moduleName, moduleVersion, moduleSourceFilePath))
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("failed to get module %s@%s/%s: %w", moduleName, moduleVersion, moduleSourceFilePath, err)
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
						Namespace: "http",
						Path:      reqPath.String(),
						PluginData: map[string]string{
							"moduleName": moduleName,
							"version":    moduleVersion,
						},
					}, nil
				})
			},
		},
	}
}

func getFileExports(input *es.StdinOptions) ([]string, error) {
	result := es.Build(es.BuildOptions{
		Stdin:    input,
		Platform: es.PlatformNeutral,
		Target:   es.ESNext,
		Write:    false,
		Metafile: true,
	})
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("failed to build: %w", buildError.BuildError(result))
	}

	var metafile struct {
		Outputs map[string]struct {
			Exports []string
		}
	}
	if err := json.Unmarshal([]byte(result.Metafile), &metafile); err != nil {
		return nil, fmt.Errorf("failed to analyze %s: %w", input.Sourcefile, err)
	}
	if len(metafile.Outputs) != 1 {
		return nil, fmt.Errorf("failed to analyze %s: expected exactly one output, got %d", input.Sourcefile, len(metafile.Outputs))
	}

	for _, meta := range metafile.Outputs {
		return meta.Exports, nil
	}

	panic(fmt.Errorf("unreachable code"))
}

func (app *CompiledApp) GetConfig() (project.RobinAppConfig, error) {
	return project.LoadRobinAppById(app.Id)
}

func (app *CompiledApp) getEnvConstants() map[string]string {
	return map[string]string{
		"process.env.ROBIN_SERVER_PORT": strconv.FormatInt(int64(app.compiler.ServerPort), 10),
		"process.env.ROBIN_APP_ID":      `"` + app.Id + `"`,
	}
}

func (app *CompiledApp) buildClientJs() error {
	appConfig, err := project.LoadRobinAppById(app.Id)
	if err != nil {
		return err
	}

	pagePath, content, err := appConfig.ReadFile(&httpClient, appConfig.Page)
	if err != nil {
		return err
	}

	stdinOptions := es.StdinOptions{
		Contents:   string(content),
		Sourcefile: pagePath.String(),
		Loader:     es.LoaderTSX,
	}
	if pagePath.Scheme == "file" {
		stdinOptions.ResolveDir = path.Dir(pagePath.Path)
	}

	app.serverExports = make(map[string][]string)
	result := es.Build(es.BuildOptions{
		Stdin:    &stdinOptions,
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Target:   es.ESNext,
		Write:    false,
		Loader: map[string]es.Loader{
			".png":  es.LoaderBase64,
			".jpg":  es.LoaderBase64,
			".jpeg": es.LoaderBase64,
		},
		Metafile: true,
		Define:   app.getEnvConstants(),
		Plugins: concat(
			getExtractServerPlugins(appConfig, app),
			toolkit.Plugin(appConfig),
			[]es.Plugin{esbuildPluginLoadHttp},
			getResolverPlugins(appConfig, pagePath),
			getCssLoaderPlugins(appConfig),

			[]es.Plugin{
				{
					Name: "catch-all",
					Setup: func(build es.PluginBuild) {
						// A catch all if we miss anything
						build.OnResolve(es.OnResolveOptions{Filter: ".", Namespace: "http"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
							return es.OnResolveResult{}, fmt.Errorf("unexpected import of %s from http resource %s", args.Path, args.Importer)
						})
					},
				},
			},
		),
	})

	if len(result.Errors) != 0 {
		return fmt.Errorf("failed to build client: %w", buildError.BuildError(result))
	}

	var metafile map[string]any
	if err := json.Unmarshal([]byte(result.Metafile), &metafile); err != nil {
		metafile = map[string]any{
			"error": err.Error(),
		}
	}

	output := result.OutputFiles[0]

	app.ClientJs = string(output.Contents)
	app.ClientMetafile = metafile
	return nil
}

func (app *CompiledApp) buildServerBundle() error {
	appConfig, err := project.LoadRobinAppById(app.Id)
	if err != nil {
		return fmt.Errorf("failed to load app config for %s: %w", app.Id, err)
	}

	pagePath, _, err := appConfig.ReadFile(&httpClient, appConfig.Page)
	if err != nil {
		return err
	}

	// Generate a bundle entrypoint that pulls all the server files into
	// a single file, and re-exports the RPC methods as a consumable map.

	serverRpcMethodsSource := ""
	for serverFile, exports := range app.serverExports {
		serverRpcMethodsSource += fmt.Sprintf(
			"import { %s } from '%s';\n",
			strings.Join(exports, ", "),
			serverFile,
		)
	}

	serverRpcMethodsSource += "\nexport const serverRpcMethods = {\n"
	for serverFile, exports := range app.serverExports {
		serverRpcMethodsSource += fmt.Sprintf(
			"\t'%s': {\n",
			serverFile,
		)
		for _, export := range exports {
			serverRpcMethodsSource += fmt.Sprintf(
				"\t\t%s,\n",
				export,
			)
		}
		serverRpcMethodsSource += "\t},\n"
	}
	serverRpcMethodsSource += "};\n"

	// Build the bundle via esbuild
	// TODO: Maybe support 'external' packages somehow, or all external packages. It'll speed up builds, but
	// more importantly, it is necessary to support native deps.
	result := es.Build(es.BuildOptions{
		Stdin: &es.StdinOptions{
			Contents:   serverRpcMethodsSource,
			Sourcefile: "server-rpc-methods.ts",
			Loader:     es.LoaderJS,
		},
		Platform: es.PlatformNode,
		Format:   es.FormatCommonJS,
		Bundle:   true,
		Write:    false,
		Define:   app.getEnvConstants(),
		Plugins: concat(
			[]es.Plugin{esbuildPluginMarkBuiltinsAsExternal},
			[]es.Plugin{esbuildPluginLoadHttp},
			[]es.Plugin{
				{
					Name: "resolve-abs-paths",
					Setup: func(build es.PluginBuild) {
						build.OnResolve(es.OnResolveOptions{
							Filter: "^/",
						}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
							return es.OnResolveResult{
								Path: args.Path,
							}, nil
						})
					},
				},
			},
			toolkit.Plugin(appConfig),
			getResolverPlugins(appConfig, pagePath),
		),
	})
	if len(result.Errors) != 0 {
		return fmt.Errorf("failed to build server: %w", buildError.BuildError(result))
	}
	if len(result.OutputFiles) != 1 {
		return fmt.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	app.ServerJs = string(result.OutputFiles[0].Contents)
	return nil
}
