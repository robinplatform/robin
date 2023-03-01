package compile

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/log"
)

var toolkitInit = sync.Once{}

func DisableEmbeddedToolkit() {
	toolkitFS = nil
	logger.Warn("Embedded toolkit disabled", log.Ctx{})
}

func (appConfig RobinAppConfig) getToolkitPlugins() []es.Plugin {
	toolkitInit.Do(initToolkit)

	if toolkitFS == nil {
		return nil
	}

	resolver := resolve.Resolver{
		FS: toolkitFS,
	}

	// The first set of plugins aim to resolve the toolkit source itself, and immediately give up on any
	// third-party module resolution requests. We try to avoid resolving modules at all, and trust that esbuild
	// or robin-resolver will do a better job.
	return []es.Plugin{
		{
			Name: "resolve-robin-toolkit",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{
					Filter:    "^\\.",
					Namespace: "robin-toolkit",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					resolvedPath, err := resolver.ResolveFrom(args.Importer, args.Path)
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("could not resolve: %s (imported by %s)", args.Path, args.Importer)
					}

					logger.Debug("Resolved toolkit path (source: toolkit)", log.Ctx{
						"args":         args,
						"resolvedPath": resolvedPath,
					})
					return es.OnResolveResult{
						Namespace: "robin-toolkit",
						Path:      resolvedPath,
					}, nil
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

					logger.Debug("Resolved toolkit path (source: external)", log.Ctx{
						"args":         args,
						"resolvedPath": resolvedPath,
					})
					return es.OnResolveResult{
						Namespace: "robin-toolkit",
						Path:      resolvedPath,
					}, nil
				})
			},
		},
		{
			Name: "load-robin-toolkit",
			Setup: func(build es.PluginBuild) {
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

					if strings.HasSuffix(args.Path, ".css") {
						script := fmt.Sprintf(`!function(){
							let style = document.createElement('style')
							style.setAttribute('data-path', '%s')
							style.innerText = %q
							document.body.appendChild(style)
						}()`, args.Path, str)
						return es.OnLoadResult{
							Contents: &script,
							Loader:   es.LoaderJS,
						}, nil
					}

					return es.OnLoadResult{
						Contents:   &str,
						ResolveDir: resolveDir,
						Loader:     es.LoaderTSX,
					}, nil
				})
			},
		},
	}
}
