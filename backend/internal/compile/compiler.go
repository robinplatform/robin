package compile

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/process"
)

type processMeta struct {
	Port int `json:"port"`
}

var (
	//go:embed client.html
	clientHtmlTemplateRaw string

	clientHtmlTemplate = template.Must(template.New("robinAppClientHtml").Parse(clientHtmlTemplateRaw))

	logger log.Logger = log.New("compiler")

	cacheEnabled = os.Getenv("ROBIN_CACHE") != "false"

	processManager *process.ProcessManager[processMeta]

	// TODO: add something like a write handle to processManager so we don't
	// need to use our own mutex
	daemonProcessMux = &sync.Mutex{}
)

func init() {
	robinPath := config.GetRobinPath()

	var err error
	processManager, err = process.NewProcessManager[processMeta](filepath.Join(
		robinPath,
		"app-processes.db",
	))
	if err != nil {
		panic(fmt.Errorf("failed to initialize compiler: %w", err))
	}
}

type Compiler struct {
	mux      sync.Mutex
	appCache map[string]CompiledApp
}

type CompiledApp struct {
	httpClient *http.Client

	Id string

	// Html holds the HTML to be rendered on the client
	Html string

	// ClientJs holds the compiled JS bundle for the client-side app
	ClientJs string

	// ClientMetafile holds the parsed metafile for the client-side app (useful for debugging)
	ClientMetafile map[string]any

	// ServerJs holds the compiled JS bundle for the server-side app
	ServerJs string

	// Cached is set to true if the app was loaded from the cache
	Cached bool

	// serverExports maps absolute paths of server files to the functions they export
	serverExports map[string][]string
}

func (compiler *Compiler) GetApp(id string) (CompiledApp, error) {
	compiler.mux.Lock()
	defer compiler.mux.Unlock()

	if app, found := compiler.appCache[id]; found {
		return app, nil
	}

	if compiler.appCache == nil && cacheEnabled {
		compiler.appCache = make(map[string]CompiledApp)
	}

	appConfig, err := LoadRobinAppById(id)
	if err != nil {
		return CompiledApp{}, fmt.Errorf("failed to load app config: %w", err)
	}

	htmlOutput := bytes.NewBuffer(nil)
	if err := clientHtmlTemplate.Execute(htmlOutput, map[string]any{
		"AppConfig": appConfig,
		"ScriptURL": fmt.Sprintf("/api/app-resources/%s/bootstrap.js", id),
	}); err != nil {
		return CompiledApp{}, fmt.Errorf("failed to render client html: %w", err)
	}

	app := CompiledApp{
		Id:     id,
		Html:   htmlOutput.String(),
		Cached: true,
	}
	if err := app.buildClientJs(); err != nil {
		return CompiledApp{}, err
	}
	if err := app.buildServerBundle(); err != nil {
		return CompiledApp{}, err
	}

	if compiler.appCache != nil {
		compiler.appCache[id] = app
	}

	app.Cached = false
	return app, nil
}

func getResolverPlugins(pageSourceUrl *url.URL, appConfig RobinAppConfig, plugins []es.Plugin) []es.Plugin {
	if appConfig.ConfigPath.Scheme == "file" {
		return plugins
	}

	resolverPlugins := []es.Plugin{
		{
			Name: "robin-resolver",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{Filter: "."}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					// If the request is to resolve a resource over HTTP/HTTPS, we should adopt a Just Do It approach
					if strings.HasPrefix(args.Path, "http:") || strings.HasPrefix(args.Path, "https:") {
						reqPath, contents, err := appConfig.ReadFile(args.Path)
						if err != nil {
							return es.OnResolveResult{}, fmt.Errorf("could not read '%s': %w", args.Path, err)
						}

						logger.Debug("Loaded remote module", log.Ctx{
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

					// Resolve local file paths relative to the remote importer's path
					// The logic for this almost entirely lives in the resolve package
					if args.Path[0] == '.' || args.Path[0] == '/' {
						resolver := resolve.NewHttpResolver(importerUrl)
						resolved, err := resolver.ResolveFrom(importerUrl.Path, args.Path)
						if err != nil {
							return es.OnResolveResult{}, fmt.Errorf("could not resolve '%s' (imported by %s): %w", args.Path, args.Importer, err)
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
					if len(moduleName) == 0 {
						return es.OnResolveResult{}, fmt.Errorf("expected module name to be non-empty in: %s", args.Path)
					}
					if moduleName[0] == '@' {
						moduleName = moduleName + "/" + pathPieces[1]
					}
					moduleSourceFilePath := strings.Join(pathPieces[1:], "/")

					// To load the source of the module, we need to know the relative version of the module.
					//
					// But there is N places that the version of the module might exist. The highest priority is in the package.json
					// of the immediate importer. If that doesn't exist, the node resolution algorithm would actually look up a single
					// parent directory at a time (i.e. if foo imports bar which then imports baz, bar might satisfy a peer dep of baz
					// which is a higher priority than a version of baz in foo).
					//
					// However, `esm.sh` takes care of most of this anyways, so we really just need to perform lookups for modules that
					// are immediately imported by the app. So we'll just look in the package.json of the immediate importer.
					_, rawPackageJson, err := appConfig.ReadFile("package.json")
					if err != nil {
						return es.OnResolveResult{}, err
					}

					var packageJson config.PackageJson
					if err := config.ParsePackageJson(rawPackageJson, &packageJson); err != nil {
						return es.OnResolveResult{}, err
					}

					moduleVersion, found := packageJson.Dependencies[moduleName]
					if !found {
						return es.OnResolveResult{}, fmt.Errorf("cannot resolve module '%s' (not found in package.json)", moduleName)
					}

					reqPath, contents, err := appConfig.ReadFile(fmt.Sprintf("/%s@%s/%s", moduleName, moduleVersion, moduleSourceFilePath))
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
	return append(plugins, resolverPlugins...)
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
		return nil, fmt.Errorf("failed to build: %w", BuildError(result))
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

func (app *CompiledApp) buildClientJs() error {
	appConfig, err := LoadRobinAppById(app.Id)
	if err != nil {
		return err
	}

	pagePath, content, err := appConfig.ReadFile(appConfig.Page)
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

		// Instead of using `append()`, this API style allows the plugin to decide its own precendence.
		// For instance, toolkit plugins are broken down and wrap the resolver plugins.
		Plugins: getToolkitPlugins(appConfig, getResolverPlugins(pagePath, appConfig, []es.Plugin{
			{
				Name: "extract-server-ts",
				Setup: func(build es.PluginBuild) {
					build.OnLoad(es.OnLoadOptions{
						Filter: "\\.server\\.ts$",
					}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
						var source []byte
						var err error

						if strings.HasPrefix(args.Path, "http://") || strings.HasPrefix(args.Path, "https://") {
							_, source, err = appConfig.ReadFile(args.Path)
						} else {
							source, err = os.ReadFile(args.Path)
						}
						if err != nil {
							return es.OnLoadResult{}, fmt.Errorf("failed to read server file %s: %w", args.Path, err)
						}

						exports, err := getFileExports(&es.StdinOptions{
							Contents:   string(source),
							Sourcefile: args.Path,
							Loader:     es.LoaderTS,
						})
						if err != nil {
							return es.OnLoadResult{}, fmt.Errorf("failed to get exports for %s: %w", args.Path, err)
						}

						serverPolyfill := "import { createRpcMethod } from '@robinplatform/toolkit/internal/rpc';\n\n"
						for _, export := range exports {
							serverPolyfill += fmt.Sprintf(
								"export const %s = createRpcMethod(%q, %q, %q);\n",
								export,
								appConfig.Id,
								args.Path,
								export,
							)
						}

						app.serverExports[args.Path] = exports

						return es.OnLoadResult{
							Contents: &serverPolyfill,
							Loader:   es.LoaderJS,
						}, nil
					})
				},
			},
			{
				Name: "load-css",
				Setup: func(build es.PluginBuild) {
					build.OnLoad(es.OnLoadOptions{
						Filter: "\\.css(\\?bundle)?$",
					}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
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

						cssEscaped, err := json.Marshal(string(css))
						if err != nil {
							return es.OnLoadResult{}, fmt.Errorf("failed to escape css file %s: %w", args.Path, err)
						}

						script := fmt.Sprintf(`!function(){
							let style = document.createElement('style')
							style.setAttribute('data-path', '%s')
							style.innerText = %s
							document.body.appendChild(style)
						}()`, args.Path, cssEscaped)
						return es.OnLoadResult{
							Contents: &script,
							Loader:   es.LoaderJS,
						}, nil
					})
				},
			},
		})),
	})

	if len(result.Errors) != 0 {
		return fmt.Errorf("failed to build client: %w", BuildError(result))
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
	appConfig, err := LoadRobinAppById(app.Id)
	if err != nil {
		return fmt.Errorf("failed to load app config for %s: %w", app.Id, err)
	}

	pagePath, _, err := appConfig.ReadFile(appConfig.Page)
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
		Plugins: getToolkitPlugins(appConfig, getResolverPlugins(pagePath, appConfig, []es.Plugin{
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
		})),
	})
	if len(result.Errors) != 0 {
		return BuildError(result)
	}
	if len(result.OutputFiles) != 1 {
		return fmt.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	app.ServerJs = string(result.OutputFiles[0].Contents)
	return nil
}

func (app *CompiledApp) getProcessId() process.ProcessId {
	return process.ProcessId{
		Namespace:    process.NamespaceExtensionDaemon,
		NamespaceKey: "app-daemon",
		Key:          app.Id,
	}
}

func (app *CompiledApp) StartServer() error {
	daemonProcessMux.Lock()
	defer daemonProcessMux.Unlock()

	// Figure out a temporary file name to write the entrypoint to
	tmpFileName := ""
	for {
		tmpDir := os.TempDir()
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}

		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
		}
		tmpFileName = filepath.Join(tmpDir, fmt.Sprintf("robin-app-server-%s-%x%s", app.Id, buf, ext))

		if _, err := os.Stat(tmpFileName); os.IsNotExist(err) {
			break
		} else if !os.IsExist(err) {
			return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
		}
	}

	// Write the entrypoint to the temporary file
	if err := os.WriteFile(tmpFileName, []byte(app.ServerJs), 0755); err != nil {
		return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
	}

	// Find a free port to listen on
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find free port: %w", err)
	}
	portAvailable := listener.Addr().(*net.TCPAddr).Port
	strPortAvailable := strconv.FormatInt(int64(portAvailable), 10)
	listener.Close()

	// Extract the daemon runner onto disk
	daemonRunnerSourceFile, err := toolkitFS.Open("internal/app-daemon.js")
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find daemon runner: %w", err)
	}
	daemonRunnerSource, err := io.ReadAll(daemonRunnerSourceFile)
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find daemon runner: %w", err)
	}

	daemonRunnerFilePath := filepath.Join(os.TempDir(), "robin-daemon-runner.js")
	daemonRunnerFile, err := os.Create(daemonRunnerFilePath)
	if err != nil {
		return fmt.Errorf("failed to start app server: could not create daemon runner: %w", err)
	}
	if _, err := daemonRunnerFile.Write(daemonRunnerSource); err != nil {
		return fmt.Errorf("failed to start app server: could not create daemon runner: %w", err)
	}
	if err := daemonRunnerFile.Close(); err != nil {
		return fmt.Errorf("failed to start app server: could not create daemon runner: %w", err)
	}

	// Start the app server process
	projectPath := config.GetProjectPathOrExit()
	serverProcess, err := processManager.SpawnPath(process.ProcessConfig[processMeta]{
		Id: process.ProcessId{
			Namespace:    process.NamespaceExtensionDaemon,
			NamespaceKey: "app-daemon",
			Key:          app.Id,
		},
		Command: "node",
		Args:    []string{daemonRunnerFilePath},
		WorkDir: projectPath,
		Env: map[string]string{
			"ROBIN_PROCESS_TYPE":  "daemon",
			"ROBIN_DAEMON_TARGET": tmpFileName,
			"PORT":                strPortAvailable,
		},
		Meta: processMeta{
			Port: portAvailable,
		},
	})
	if err != nil && !errors.Is(err, process.ErrProcessAlreadyExists) {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	// Wait for process to become ready
	for i := 0; i < 10; i++ {
		// Make sure the process is still running
		if !serverProcess.IsAlive() {
			return fmt.Errorf("failed to start app server: process died")
		}

		// Send a ping to the process
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", serverProcess.Meta.Port))
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		if resp == nil {
			logger.Debug("Failed to ping app server", log.Ctx{
				"appId": app.Id,
				"pid":   serverProcess.Pid,
				"err":   err,
			})
		} else {
			logger.Debug("Failed to ping app server", log.Ctx{
				"appId":  app.Id,
				"pid":    serverProcess.Pid,
				"err":    err,
				"status": resp.StatusCode,
			})
		}

		// Wait a bit
		time.Sleep(1 * time.Second)
	}

	if err := app.StopServer(); err != nil {
		logger.Warn("Failed to stop unhealthy app server", log.Ctx{
			"appId": app.Id,
			"pid":   serverProcess.Pid,
			"err":   err,
		})
	}

	return fmt.Errorf("failed to start app server: process did not become ready")
}

func (app *CompiledApp) StopServer() error {
	daemonProcessMux.Lock()
	defer daemonProcessMux.Unlock()

	if err := processManager.Kill(app.getProcessId()); err != nil && !errors.Is(err, process.ErrProcessNotFound) {
		return fmt.Errorf("failed to stop app server: %w", err)
	}
	return nil
}

func (app *CompiledApp) Request(ctx context.Context, method string, reqPath string, body map[string]any) (any, error) {
	if app.httpClient == nil {
		app.httpClient = &http.Client{}
	}

	serverProcess, err := processManager.FindById(app.getProcessId())
	if err != nil {
		return nil, fmt.Errorf("failed to make app request: %w", err)
	}

	serializedBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize app request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		fmt.Sprintf("http://localhost:%d%s", serverProcess.Meta.Port, reqPath),
		bytes.NewReader(serializedBody),
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to create app request: %w", err)
	}

	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make app request: %w", err)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read app response: %w (http status %d)", err, resp.StatusCode)
	}

	var respBody struct {
		Type   string
		Error  string
		Result any
	}
	if err := json.Unmarshal(buf, &respBody); err != nil {
		return nil, fmt.Errorf("failed to deserialize app response: %w (http status %d)", err, resp.StatusCode)
	}

	if respBody.Type == "error" {
		return nil, fmt.Errorf("failed to make app request: %s", respBody.Error)
	}

	return respBody.Result, nil
}
