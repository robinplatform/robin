package compile

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/log"
)

var httpClient httpcache.CacheClient
var esmSHResolver *resolve.Resolver

func init() {
	robinPath := config.GetRobinPath()

	var err error
	cacheFilename := filepath.Join(robinPath, "http-cache.json")
	httpClient, err = httpcache.NewClient(cacheFilename, 1024*1024*1024)
	if err != nil {
		httpLogger := log.New("http")
		httpLogger.Debug("Failed to load HTTP cache, will recreate", log.Ctx{
			"error": err,
			"path":  cacheFilename,
		})
	}

	esmSHResolver = resolve.NewHttpResolver(&url.URL{
		Scheme: "https",
		Host:   "esm.sh",
	}, httpClient)
}

func getHttpResolver(importerUrl *url.URL) *resolve.Resolver {
	if importerUrl.Host == "esm.sh" {
		return esmSHResolver
	}
	return resolve.NewHttpResolver(importerUrl, httpClient)
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

var esbuildPluginLoadHttp = es.Plugin{
	Name: "load-http",
	Setup: func(build es.PluginBuild) {
		// If we try to make a request to a URL, assume that the server on the other end will handle resolution for us. We will simply
		// mark this as ready-to-load, and follow redirects.
		build.OnResolve(es.OnResolveOptions{Filter: "^https?://"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			return es.OnResolveResult{
				Path:      args.Path,
				Namespace: "http",
			}, nil
		})

		// When we have an absolute path with a file extension, we want to 'resolve' it by simply forming a proper URL using the importer's
		// schema, host, and port.
		build.OnResolve(es.OnResolveOptions{Filter: "^/", Namespace: "http"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			if path.Ext(args.Path) == "" {
				return es.OnResolveResult{}, nil
			}

			importerUrl, err := url.Parse(args.Importer)
			if err != nil {
				return es.OnResolveResult{}, fmt.Errorf("failed to parse importer url %s: %w", args.Importer, err)
			}

			return es.OnResolveResult{
				Path:      importerUrl.ResolveReference(&url.URL{Path: args.Path}).String(),
				Namespace: "http",
			}, nil
		})

		// For any relative path requests coming from an HTTP resource, we can use the HTTP resolver to resolve the path to a proper URL.
		// This is not scoped to the namespace, because we want to catch imports from the core page of the app, which wouldn't be namespaced.
		build.OnResolve(es.OnResolveOptions{Filter: "^\\."}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			if args.Namespace != "http" && !strings.HasPrefix(args.Importer, "http://") && !strings.HasPrefix(args.Importer, "https://") {
				return es.OnResolveResult{}, nil
			}

			importerUrl, err := url.Parse(args.Importer)
			if err != nil {
				return es.OnResolveResult{}, fmt.Errorf("failed to parse url %s: %w", args.Importer, err)
			}
			if importerUrl.Scheme != "http" && importerUrl.Scheme != "https" {
				return es.OnResolveResult{}, fmt.Errorf("invalid importer url %s", args.Importer)
			}

			resolver := getHttpResolver(importerUrl)
			resolvedPath, err := resolver.ResolveFrom(importerUrl.Path, args.Path)
			if err != nil {
				return es.OnResolveResult{}, fmt.Errorf("failed to resolve %s (imported by %s)", args.Path, args.Importer)
			}

			return es.OnResolveResult{
				Path:      fmt.Sprintf("%s://%s/%s", importerUrl.Scheme, importerUrl.Host, resolvedPath),
				Namespace: "http",
			}, nil
		})

		// Finally, we can load HTTP resources by making a simple request and following redirects.
		build.OnLoad(es.OnLoadOptions{Filter: "^https?://"}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
			targetUrl, err := url.Parse(args.Path)
			if err != nil {
				return es.OnLoadResult{}, fmt.Errorf("failed to parse url %s: %w", args.Path, err)
			}

			if targetUrl.Host == "esm.sh" {
				targetUrl.RawQuery += "target=esnext"
			}

			res, err := httpClient.Get(args.Path)
			if err != nil {
				return es.OnLoadResult{}, fmt.Errorf("failed to load %s: %w", args.Path, err)
			}

			str := res.Body
			if strings.HasSuffix(targetUrl.Path, ".css") {
				str = wrapWithCssLoader(args.Path, str)
			}

			return es.OnLoadResult{
				Contents: &str,
				Loader:   es.LoaderJS,
			}, nil
		})

		build.OnEnd(func(_ *es.BuildResult) (es.OnEndResult, error) {
			go httpClient.Save()
			return es.OnEndResult{}, nil
		})
	},
}
