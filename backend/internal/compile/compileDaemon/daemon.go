package compileDaemon

import (
	"fmt"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/buildError"
	"robinplatform.dev/internal/compile/plugins"
	"robinplatform.dev/internal/compile/toolkit"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/project"
)

type ServerBundleInput struct {
	AppId           string
	HttpClient      httpcache.CacheClient
	DefineConstants map[string]string
	ServerExports   map[string][]string
}

type ServerBundle struct {
	ServerJS string
}

func BuildServerBundle(input ServerBundleInput) (ServerBundle, error) {
	appConfig, err := project.LoadRobinAppById(input.AppId)
	if err != nil {
		return ServerBundle{}, fmt.Errorf("failed to load app config for %s: %w", input.AppId, err)
	}

	pagePath, _, err := appConfig.ReadFile(input.HttpClient, appConfig.Page)
	if err != nil {
		return ServerBundle{}, err
	}

	// Generate a bundle entrypoint that pulls all the server files into
	// a single file, and re-exports the RPC methods as a consumable map.

	serverRpcMethodsSource := ""
	for serverFile, exports := range input.ServerExports {
		serverRpcMethodsSource += fmt.Sprintf(
			"import { %s } from '%s';\n",
			strings.Join(exports, ", "),
			serverFile,
		)
	}

	serverRpcMethodsSource += "\nexport const serverRpcMethods = {\n"
	for serverFile, exports := range input.ServerExports {
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
		Define:   input.DefineConstants,
		Plugins: concat(
			[]es.Plugin{esbuildPluginMarkBuiltinsAsExternal},
			plugins.LoadHttp(input.HttpClient),
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
			toolkit.Plugins(appConfig),
			plugins.ResolverPlugin(appConfig, input.HttpClient, pagePath),
		),
	})
	if len(result.Errors) != 0 {
		return ServerBundle{}, fmt.Errorf("failed to build server: %w", buildError.BuildError(result))
	}
	if len(result.OutputFiles) != 1 {
		return ServerBundle{}, fmt.Errorf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	bundle := ServerBundle{
		ServerJS: string(result.OutputFiles[0].Contents),
	}
	return bundle, nil
}

var nodeBuiltinModules = []string{
	"assert",
	"buffer",
	"child_process",
	"cluster",
	"console",
	"constants",
	"crypto",
	"dgram",
	"dns",
	"domain",
	"events",
	"fs",
	"http",
	"https",
	"module",
	"net",
	"os",
	"path",
	"perf_hooks",
	"process",
	"punycode",
	"querystring",
	"readline",
	"repl",
	"stream",
	"string_decoder",
	"sys",
	"timers",
	"tls",
	"tty",
	"url",
	"util",
	"v8",
	"vm",
	"zlib",
}

var esbuildPluginMarkBuiltinsAsExternal = es.Plugin{
	Name: "mark-builtins-as-external",
	Setup: func(build es.PluginBuild) {
		filter := fmt.Sprintf("^(%s)$", strings.Join(nodeBuiltinModules, "|"))
		build.OnResolve(es.OnResolveOptions{Filter: filter}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			return es.OnResolveResult{
				Path:      args.Path,
				External:  true,
				Namespace: "builtin",
			}, nil
		})

		// Also mark any imports that start with 'node:' as external, since they are also builtins.
		build.OnResolve(es.OnResolveOptions{Filter: "^node:"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			return es.OnResolveResult{
				Path:      args.Path,
				External:  true,
				Namespace: "builtin",
			}, nil
		})
	},
}

func concat[T any](lists ...[]T) []T {
	concatLen := 0
	for _, list := range lists {
		concatLen += len(list)
	}

	output := make([]T, 0, concatLen)
	for _, list := range lists {
		output = append(output, list...)
	}
	return output
}
