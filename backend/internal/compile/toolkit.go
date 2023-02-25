package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

var toolkitInit = sync.Once{}

func DisableEmbeddedToolkit() {
	toolkitFS = nil
	logger.Warn("Embedded toolkit disabled", log.Ctx{})
}

func getToolkitPlugins(appConfig RobinAppConfig, plugins []es.Plugin) []es.Plugin {
	toolkitInit.Do(initToolkit)

	if toolkitFS == nil {
		return plugins
	}

	resolver := resolve.Resolver{
		FS: toolkitFS,
	}

	projectPath := config.GetProjectPathOrExit()
	moduleResolver := resolve.Resolver{
		FS:              os.DirFS(projectPath),
		EnableDebugLogs: resolver.EnableDebugLogs,
	}

	// The first set of plugins aim to resolve the toolkit source itself, and immediately give up on any
	// third-party module resolution requests. This gives a chance for robin-resolver to attempt to resolve
	// the module. This is important so we don't end up loading two different versions of react.
	pluginsStart := []es.Plugin{
		{
			Name: "resolve-robin-toolkit",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{
					Filter:    "^\\.",
					Namespace: "robin-toolkit",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					resolvedPath, err := resolver.Resolve(args.Path)
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("could not resolve: %s (imported by %s)", args.Path, args.Importer)
					}

					return es.OnResolveResult{
						Namespace: "robin-toolkit",
						Path:      resolvedPath,
					}, err
				})

				build.OnResolve(es.OnResolveOptions{
					Filter: "^@robinplatform/toolkit",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					// Update the path to be relative to the resolver's FS root
					sourcePath := "." + strings.TrimPrefix(args.Path, "@robinplatform/toolkit")
					resolvedPath, err := resolver.Resolve(sourcePath)
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("could not resolve: %s (imported by %s)", args.Path, args.Importer)
					}

					return es.OnResolveResult{
						Namespace: "robin-toolkit",
						Path:      resolvedPath,
					}, err
				})
			},
		},
	}

	// The second set of plugins try to resolve everything that was never resolved, and then load the
	// toolkit sources.
	pluginsEnd := []es.Plugin{
		{
			Name: "load-robin-toolkit",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{
					Namespace: "robin-toolkit",
					Filter:    ".",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					resolvedPath, err := moduleResolver.ResolveFrom(
						strings.TrimPrefix(appConfig.Page, projectPath+string(filepath.Separator)),
						args.Path,
					)

					resultPath := filepath.Join(projectPath, resolvedPath)
					logger.Debug("Resolved module", log.Ctx{
						"args":      args,
						"appConfig": appConfig,
						"result":    resultPath,
					})

					// We don't want to namespace this, since it is a regular node module and can
					// be loaded by esbuild
					return es.OnResolveResult{
						Path: resultPath,
					}, err
				})

				build.OnLoad(es.OnLoadOptions{
					Filter:    ".",
					Namespace: "robin-toolkit",
				}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
					contents, ok := resolver.ReadFile(args.Path)
					if !ok {
						return es.OnLoadResult{}, fmt.Errorf("could not read file %s", args.Path)
					}

					resolveDir := ""
					if appConfig.ConfigPath.Scheme == "file" {
						resolveDir = filepath.Dir(appConfig.ConfigPath.Path)
					}

					logger.Debug("Loaded module", log.Ctx{
						"args":       args,
						"resolveDir": resolveDir,
					})

					str := string(contents)
					return es.OnLoadResult{
						Contents:   &str,
						ResolveDir: resolveDir,
						Loader:     es.LoaderTSX,
					}, nil
				})
			},
		},
	}

	plugins = append(pluginsStart, plugins...)
	plugins = append(plugins, pluginsEnd...)
	return plugins
}
