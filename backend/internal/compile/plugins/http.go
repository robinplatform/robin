package plugins

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
	"robinplatform.dev/internal/httpcache"
)

func LoadHttp(httpClient httpcache.CacheClient) []es.Plugin {

	return []es.Plugin{{
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

				resolver := resolve.NewHttpResolver(importerUrl, httpClient)
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
	},
	}
}
