package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

func DisableEmbeddedToolkit() {
	toolkitFS = nil
	logger.Warn("Embedded toolkit disabled", log.Ctx{})
}

func getToolkitPlugins(appConfig config.RobinAppConfig) []es.Plugin {
	if toolkitFS == nil {
		return nil
	}

	resolver := resolve.Resolver{
		FS: toolkitFS,
	}

	projectPath := config.GetProjectPathOrExit()
	moduleResolver := resolve.Resolver{
		FS:              os.DirFS(projectPath),
		EnableDebugLogs: resolver.EnableDebugLogs,
	}

	return []es.Plugin{
		{
			Name: "load-robin-toolkit",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{
					Namespace: "robin-toolkit",
					Filter:    ".",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					if args.Path[0] != '.' {
						logger.Debug("Resolving module", log.Ctx{
							"args":      args,
							"appConfig": appConfig,
						})

						resolvedPath, err := moduleResolver.ResolveFrom(
							strings.TrimPrefix(appConfig.Page, projectPath+string(filepath.Separator)),
							args.Path,
						)

						// We don't want to namespace this, since it is a regular node module and can
						// be loaded by esbuild
						return es.OnResolveResult{
							Path: filepath.Join(projectPath, resolvedPath),
						}, err
					}

					resolvedPath, err := resolver.Resolve(args.Path)
					return es.OnResolveResult{
						Namespace: "robin-toolkit",
						Path:      filepath.Join(toolkitPath, resolvedPath),
						PluginData: map[string]string{
							"fsPath": resolvedPath,
						},
					}, err
				})

				build.OnResolve(es.OnResolveOptions{
					Filter: "@robinplatform/toolkit",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					if !strings.HasPrefix(args.Path, "@robinplatform/toolkit") {
						return es.OnResolveResult{}, nil
					}

					// Update the path to be relative to the resolver's FS root
					sourcePath := "." + strings.TrimPrefix(args.Path, "@robinplatform/toolkit")
					resolvedPath, err := resolver.Resolve(sourcePath)

					return es.OnResolveResult{
						Namespace: "robin-toolkit",
						Path:      filepath.Join(toolkitPath, resolvedPath),
						PluginData: map[string]string{
							"fsPath": resolvedPath,
						},
					}, err
				})

				build.OnLoad(es.OnLoadOptions{
					Filter:    ".",
					Namespace: "robin-toolkit",
				}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
					pluginData, ok := args.PluginData.(map[string]string)

					// this should never happen, since we are scoped to the namespace
					if !ok {
						return es.OnLoadResult{}, fmt.Errorf("could not find pluginData for %s", args.Path)
					}

					contents, ok := resolver.ReadFile(pluginData["fsPath"])
					if !ok {
						return es.OnLoadResult{}, fmt.Errorf("could not read file %s", args.Path)
					}

					str := string(contents)
					return es.OnLoadResult{
						Contents:   &str,
						ResolveDir: filepath.Dir(args.Path),
						Loader:     es.LoaderTSX,
					}, nil
				})
			},
		},
	}
}
