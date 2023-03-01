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
			route := strings.TrimPrefix(assetPath, "web")
			if strings.HasSuffix(assetPath, ".html") {
				route = regexp.MustCompile(`\[([^\]]+)\]`).ReplaceAllString(route, ":$1")
				route = strings.TrimSuffix(route, ".html")

				if strings.HasSuffix(route, "/index") {
					route = strings.TrimSuffix(route, "index")
				}
			}

			mimetype, ok := mimetypes[path.Ext(assetPath)]
			if !ok {
				mimetype = "text/html"
			}

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
		return nil
	})
	if err != nil {
		panic(err)
	}
	server.webRouter = router
}
