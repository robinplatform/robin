package compile

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
)

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

			resolver := resolve.NewHttpResolver(importerUrl)
			resolvedPath, err := resolver.ResolveFrom(importerUrl.Path, args.Path)
			if err != nil {
				return es.OnResolveResult{}, fmt.Errorf("failed to resolve %s (imported by %s)", args.Path, args.Importer)
			}

			return es.OnResolveResult{
				Path:      fmt.Sprintf("%s://%s/%s", importerUrl.Scheme, importerUrl.Host, resolvedPath),
				Namespace: "http",
			}, nil
		})

		// A catch all if we miss anything that was tagged for this namespace
		build.OnResolve(es.OnResolveOptions{Filter: ".", Namespace: "http"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			return es.OnResolveResult{}, fmt.Errorf("unexpected import of %s from http resource %s", args.Path, args.Importer)
		})

		// Finally, we can load HTTP resources by making a simple request and following redirects.
		build.OnLoad(es.OnLoadOptions{Filter: "^https?://"}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
			targetUrl, err := url.Parse(args.Path)
			if err != nil {
				return es.OnLoadResult{}, fmt.Errorf("failed to parse url %s: %w", args.Path, err)
			}

			res, err := http.Get(args.Path)
			if err != nil {
				return es.OnLoadResult{}, fmt.Errorf("failed to load http resource %s: %w", args.Path, err)
			}
			if res.StatusCode != http.StatusOK {
				return es.OnLoadResult{}, fmt.Errorf("failed to load http resource %s: %s", args.Path, res.Status)
			}

			buf, err := io.ReadAll(res.Body)
			if err != nil {
				return es.OnLoadResult{}, fmt.Errorf("failed to load http resource %s: %w", args.Path, err)
			}

			str := string(buf)
			if strings.HasSuffix(targetUrl.Path, ".css") {
				str = wrapWithCssLoader(args.Path, str)
			}

			return es.OnLoadResult{
				Contents: &str,
				Loader:   es.LoaderJS,
			}, nil
		})
	},
}
