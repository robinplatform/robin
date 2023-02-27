package compile

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/resolve"
)

var esbuildPluginLoadHttp = es.Plugin{
	Name: "load-http",
	Setup: func(build es.PluginBuild) {
		build.OnResolve(es.OnResolveOptions{Filter: "^https?://"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			return es.OnResolveResult{
				Path:      args.Path,
				Namespace: "http",
			}, nil
		})
		build.OnResolve(es.OnResolveOptions{Filter: "^[./]", Namespace: "http"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
			importerUrl, err := url.Parse(args.Importer)
			if err != nil {
				return es.OnResolveResult{}, fmt.Errorf("failed to parse url %s: %w", args.Importer, err)
			}
			if importerUrl.Scheme != "http" && importerUrl.Scheme != "https" {
				return es.OnResolveResult{}, fmt.Errorf("invalid importer url %s", args.Importer)
			}

			resolver := resolve.NewHttpResolver(importerUrl)
			resolvedPath, err := resolver.ResolveFrom(args.Importer, args.Path)
			if err != nil {
				return es.OnResolveResult{}, fmt.Errorf("failed to resolve %s (imported by %s)", args.Path, args.Importer)
			}

			return es.OnResolveResult{
				Path:      fmt.Sprintf("%s://%s/%s", importerUrl.Scheme, importerUrl.Host, resolvedPath),
				Namespace: "http",
			}, nil
		})
		build.OnLoad(es.OnLoadOptions{Filter: "^https?://", Namespace: "http"}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
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

			// TODO: This could've been a CSS file

			str := string(buf)
			return es.OnLoadResult{
				Contents: &str,
				Loader:   es.LoaderJS,
			}, nil
		})
	},
}
