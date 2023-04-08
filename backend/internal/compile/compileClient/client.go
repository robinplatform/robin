package compileClient

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"text/template"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/compile/buildError"
	"robinplatform.dev/internal/compile/plugins"
	"robinplatform.dev/internal/compile/toolkit"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/project"
)

var (
	//go:embed client.html
	clientHtmlTemplateRaw string

	clientHtmlTemplate = template.Must(template.New("robinAppClientHtml").Parse(clientHtmlTemplateRaw))
)

type ClientJSInput struct {
	AppId           string
	HttpClient      httpcache.CacheClient
	DefineConstants map[string]string
}

type ClientBundle struct {
	Metafile      map[string]any
	Html          string
	ServerExports map[string][]string
}

func BuildClientBundle(input ClientJSInput) (ClientBundle, error) {
	appConfig, err := project.LoadRobinAppById(input.AppId)
	if err != nil {
		return ClientBundle{}, err
	}

	pagePath, content, err := appConfig.ReadFile(input.HttpClient, appConfig.Page)
	if err != nil {
		return ClientBundle{}, err
	}

	stdinOptions := es.StdinOptions{
		Contents:   string(content),
		Sourcefile: pagePath.String(),
		Loader:     es.LoaderTSX,
	}
	if pagePath.Scheme == "file" {
		stdinOptions.ResolveDir = path.Dir(pagePath.Path)
	}

	serverExports := make(map[string][]string)
	result := es.Build(es.BuildOptions{
		Stdin:    &stdinOptions,
		Bundle:   true,
		Platform: es.PlatformBrowser,
		Target:   es.ESNext,
		Write:    false,
		Loader: map[string]es.Loader{
			".png":  es.LoaderBase64,
			".jpg":  es.LoaderBase64,
			".jpeg": es.LoaderBase64,
		},
		Metafile: true,
		Define:   input.DefineConstants,
		Plugins: concat(
			getExtractServerPlugins(appConfig, input.HttpClient, serverExports),
			toolkit.Plugins(appConfig),
			plugins.LoadHttp(input.HttpClient),
			plugins.ResolverPlugin(appConfig, input.HttpClient, pagePath),
			plugins.LoadCSS(appConfig, input.HttpClient),

			[]es.Plugin{
				{
					Name: "catch-all",
					Setup: func(build es.PluginBuild) {
						// A catch all if we miss anything
						build.OnResolve(es.OnResolveOptions{Filter: ".", Namespace: "http"}, func(args es.OnResolveArgs) (es.OnResolveResult, error) {
							return es.OnResolveResult{}, fmt.Errorf("unexpected import of %s from http resource %s", args.Path, args.Importer)
						})
					},
				},
			},
		),
	})

	if len(result.Errors) != 0 {
		return ClientBundle{}, fmt.Errorf("failed to build client: %w", buildError.BuildError(result))
	}

	var metafile map[string]any
	if err := json.Unmarshal([]byte(result.Metafile), &metafile); err != nil {
		metafile = map[string]any{
			"error": err.Error(),
		}
	}

	output := result.OutputFiles[0]

	js := string(output.Contents)
	htmlOutput := bytes.NewBuffer(nil)
	if err := clientHtmlTemplate.Execute(htmlOutput, map[string]any{
		"AppConfig":    appConfig,
		"ScriptSource": js,
	}); err != nil {
		return ClientBundle{}, fmt.Errorf("failed to render client html: %w", err)
	}

	bundle := ClientBundle{
		Metafile:      metafile,
		Html:          htmlOutput.String(),
		ServerExports: serverExports,
	}
	return bundle, nil
}

func getExtractServerPlugins(appConfig project.RobinAppConfig, httpClient httpcache.CacheClient, serverExports map[string][]string) []es.Plugin {
	m := sync.Mutex{}

	return []es.Plugin{
		{
			Name: "extract-server-ts",
			Setup: func(build es.PluginBuild) {
				build.OnLoad(es.OnLoadOptions{
					Filter: "\\.server\\.[jt]s$",
				}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
					var source []byte
					var err error

					if strings.HasPrefix(args.Path, "http://") || strings.HasPrefix(args.Path, "https://") {
						_, source, err = appConfig.ReadFile(httpClient, args.Path)
					} else {
						source, err = os.ReadFile(args.Path)
					}
					if err != nil {
						return es.OnLoadResult{}, fmt.Errorf("failed to read server file %s: %w", args.Path, err)
					}

					exports, err := getFileExports(&es.StdinOptions{
						Contents:   string(source),
						Sourcefile: args.Path,
						Loader:     es.LoaderTS,
					})
					if err != nil {
						return es.OnLoadResult{}, fmt.Errorf("failed to get exports for %s: %w", args.Path, err)
					}

					serverPolyfill := "import { createRpcMethod } from '@robinplatform/toolkit/internal/rpc';\n\n"
					for _, export := range exports {
						serverPolyfill += fmt.Sprintf(
							"export const %s = createRpcMethod(%q, %q, %q);\n",
							export,
							appConfig.Id,
							args.Path,
							export,
						)
					}

					m.Lock()
					serverExports[args.Path] = exports
					m.Unlock()

					return es.OnLoadResult{
						Contents: &serverPolyfill,
						Loader:   es.LoaderJS,
					}, nil
				})
			},
		},
	}
}

func getFileExports(input *es.StdinOptions) ([]string, error) {
	result := es.Build(es.BuildOptions{
		Stdin:    input,
		Platform: es.PlatformNeutral,
		Target:   es.ESNext,
		Write:    false,
		Metafile: true,
	})
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("failed to build: %w", buildError.BuildError(result))
	}

	var metafile struct {
		Outputs map[string]struct {
			Exports []string
		}
	}
	if err := json.Unmarshal([]byte(result.Metafile), &metafile); err != nil {
		return nil, fmt.Errorf("failed to analyze %s: %w", input.Sourcefile, err)
	}
	if len(metafile.Outputs) != 1 {
		return nil, fmt.Errorf("failed to analyze %s: expected exactly one output, got %d", input.Sourcefile, len(metafile.Outputs))
	}

	for _, meta := range metafile.Outputs {
		return meta.Exports, nil
	}

	panic(fmt.Errorf("unreachable code"))
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
