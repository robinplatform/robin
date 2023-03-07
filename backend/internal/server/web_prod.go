//go:build prod

package server

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/julienschmidt/httprouter"
	"robinplatform.dev/internal/log"
)

//go:generate rm -rf web
//go:generate cp -R ../../../frontend/out ./web
//go:embed all:web
var nextBuild embed.FS

var mimetypes = map[string]string{
	".css": "text/css",
	".js":  "text/javascript",
}

func (server *Server) loadRoutes() {
	router := httprouter.New()
	err := fs.WalkDir(nextBuild, ".", func(assetPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			routes := []string{strings.TrimPrefix(assetPath, "web")}

			if strings.HasSuffix(assetPath, ".html") {
				wildcardStartIndex := strings.Index(routes[0], "[[...")
				if wildcardStartIndex != -1 {
					wildcardName := routes[0][wildcardStartIndex+5:]
					wildcardName = wildcardName[:len(wildcardName)-7]

					routes = []string{
						routes[0][:wildcardStartIndex-1],
						routes[0][:wildcardStartIndex] + "*" + wildcardName,
					}
				}

				for idx := range routes {
					routes[idx] = regexp.MustCompile(`\[([^\]]+)\]`).ReplaceAllString(routes[idx], ":$1")
					routes[idx] = strings.TrimSuffix(routes[idx], ".html")

					if strings.HasSuffix(routes[0], "/index") {
						routes[idx] = strings.TrimSuffix(routes[idx], "index")
					}
				}
			}

			mimetype, ok := mimetypes[path.Ext(assetPath)]
			if !ok {
				mimetype = "text/html"
			}

			for _, route := range routes {
				logger.Debug("Registering route", log.Ctx{
					"route": route,
					"path":  assetPath,
				})

				router.GET(route, func(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
					file, err := nextBuild.Open(assetPath)
					if os.IsNotExist(err) {
						// TODO: serve 404 page
						res.WriteHeader(404)
						res.Write([]byte("404 - Not Found"))
					} else if err != nil {
						res.WriteHeader(500)
						res.Write([]byte(err.Error()))
					} else {
						res.Header().Set("Content-Type", mimetype)
						io.Copy(res, file)
					}
				})
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	server.webRouter = router
}
