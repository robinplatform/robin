package plugins

import (
	"fmt"
	"net/url"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/toolkit"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/project"
)

var resolverLogger = log.New("plugins.ResolverPlugin")

func ResolverPlugin(appConfig project.RobinAppConfig, httpClient httpcache.CacheClient, pageSourceUrl *url.URL) []es.Plugin {
	if appConfig.ConfigPath.Scheme == "file" {
		return nil
	}

	return []es.Plugin{
		{
			Name: "robin-resolver",
			Setup: func(build es.PluginBuild) {
				build.OnResolve(es.OnResolveOptions{Filter: "^[^/\\.]"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
					// If we're resolving a module from the virtual toolkit, we should assume that the extension
					// itself asked for it
					if args.Namespace == toolkit.Namespace {
						args.Importer = pageSourceUrl.String()
					}

					importerUrl, err := url.Parse(args.Importer)
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("expected source to be a valid URL: %w", err)
					}
					if importerUrl.Scheme != "https" {
						return es.OnResolveResult{}, fmt.Errorf("expected source to be a valid URL: %s", importerUrl)
					}
					if importerUrl.Host != appConfig.ConfigPath.Host {
						return es.OnResolveResult{}, fmt.Errorf("expected all app imports to come from the same host: %s", importerUrl)
					}

					// We want to parse the pathname, which will look something like: `react/jsx-runtime`
					// The output should be the moduleName as `react` (with a possible scope name prefix), and then
					// the rest of the filepath being imported _from_ react, which is `jsx-runtime`.
					pathPieces := strings.Split(args.Path, "/")
					moduleName := pathPieces[0]
					moduleSourceFilePath := ""
					if len(moduleName) == 0 {
						return es.OnResolveResult{}, fmt.Errorf("expected module name to be non-empty in: %s", args.Path)
					}
					if moduleName[0] == '@' {
						moduleName = moduleName + "/" + pathPieces[1]
						moduleSourceFilePath = strings.Join(pathPieces[2:], "/")
					} else {
						moduleSourceFilePath = strings.Join(pathPieces[1:], "/")
					}

					// To load the source of the module, we need to know the relative version of the module.
					//
					// But there is N places that the version of the module might exist. The highest priority is in the package.json
					// of the immediate importer. If that doesn't exist, the node resolution algorithm would actually look up a single
					// parent directory at a time (i.e. if foo imports bar which then imports baz, bar might satisfy a peer dep of baz
					// which is a higher priority than a version of baz in foo).
					//
					// However, `esm.sh` takes care of most of this anyways, so we really just need to perform lookups for modules that
					// are immediately imported by the app. So we'll just look in the package.json of the immediate importer.
					packageJsonPath, rawPackageJson, err := appConfig.ReadFile(httpClient, "package.json")
					if err != nil {
						return es.OnResolveResult{}, err
					}

					var packageJson project.PackageJson
					if err := project.ParsePackageJson(rawPackageJson, &packageJson); err != nil {
						return es.OnResolveResult{}, err
					}

					moduleVersion, found := packageJson.Dependencies[moduleName]
					if !found {
						resolverLogger.Debug("Failed to find module version in package.json", log.Ctx{
							"packageJsonPath": packageJsonPath,
							"packageJson":     packageJson,
							"moduleName":      moduleName,
						})
						return es.OnResolveResult{}, fmt.Errorf("cannot resolve module '%s' (not found in package.json)", moduleName)
					}

					reqPath, _, err := appConfig.ReadFile(httpClient, fmt.Sprintf("/%s@%s/%s", moduleName, moduleVersion, moduleSourceFilePath))
					if err != nil {
						return es.OnResolveResult{}, fmt.Errorf("failed to get module %s@%s/%s: %w", moduleName, moduleVersion, moduleSourceFilePath, err)
					}

					resolverLogger.Debug("Resolved remote module for remote module", log.Ctx{
						"importer":             args.Importer,
						"path":                 args.Path,
						"moduleName":           moduleName,
						"moduleVersion":        moduleVersion,
						"moduleSourceFilePath": moduleSourceFilePath,
						"resolvedPath":         reqPath.String(),
					})
					return es.OnResolveResult{
						Namespace: "http",
						Path:      reqPath.String(),
						PluginData: map[string]string{
							"moduleName": moduleName,
							"version":    moduleVersion,
						},
					}, nil
				})
			},
		},
	}
}
