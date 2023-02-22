//go:build prod

package server

import (
	"embed"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:generate cp -R ../../../frontend/out ./web
//go:embed all:web
var nextBuild embed.FS

var mimetypes = map[string]string{
	".css": "text/css",
	".js":  "text/javascript",
}

func (server *Server) loadRoutes() {
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

			server.router.GET(route, func(c *gin.Context) {
				file, err := nextBuild.Open(assetPath)
				if err != nil {
					c.AbortWithStatus(500)
				} else {
					c.DataFromReader(200, -1, mimetype, file, nil)
				}
			})
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
