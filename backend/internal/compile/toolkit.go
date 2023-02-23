package compile

import (
	"fmt"
	"path/filepath"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
)

func getToolkitPlugins() []es.Plugin {
	if toolkitFS == nil {
		return nil
	}

	return []es.Plugin{
		{
			Name: "load-robin-toolkit",
			Setup: func(build es.PluginBuild) {
				resolver := resolve.Resolver{
					FS: toolkitFS,
				}

				build.OnResolve(es.OnResolveOptions{
					Namespace: "robin-toolkit",
					Filter:    ".",
				}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					if args.Path[0] != '.' {
						fmt.Printf("refusing to resolve: %#v\n", args)
						return es.OnResolveResult{}, nil
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
					fsPath := (args.PluginData.(map[string]string))["fsPath"]

					contents, ok := resolver.ReadFile(fsPath)
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
