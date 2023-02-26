//go:build prod

package server

import (
	"embed"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

//go:generate cp -R ../../../frontend/out ./web
//go:embed all:web
var nextBuild embed.FS

var mimetypes = map[string]string{
	".css": "text/css",
	".js":  "text/javascript",
}

func (server *Server) loadRoutes() {
	router := httprouter.New()
	router.ServeFiles("/*filepath", http.FS(nextBuild))
	server.webRouter = router
}
