package compilerServer

import (
	"fmt"
	"path/filepath"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/log"
)

var httpClient httpcache.CacheClient

func init() {
	robinPath := config.GetRobinPath()

	var err error
	cacheFilename := filepath.Join(robinPath, "data", "http-cache.json")
	httpClient, err = httpcache.NewClient(cacheFilename, 100*1024*1024)
	if err != nil {
		httpLogger := log.New("http")
		httpLogger.Debug("Failed to load HTTP cache, will recreate", log.Ctx{
			"error": err,
			"path":  cacheFilename,
		})
	}

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
