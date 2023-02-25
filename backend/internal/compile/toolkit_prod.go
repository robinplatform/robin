//go:build toolkit && prod

package compile

import (
	"embed"
	"io/fs"
	"path"

	"robinplatform.dev/internal/log"
)

type toolkitFsWrapper embed.FS

func (e toolkitFsWrapper) Open(name string) (fs.File, error) {
	return embed.FS(e).Open(path.Join("toolkit", name))
}

//go:generate rm -rf toolkit
//go:generate cp -vR ../../../toolkit toolkit
//go:generate rm -rf toolkit/node_modules toolkit/src
//go:embed all:toolkit
var embedToolkitFS embed.FS
var toolkitFS fs.FS = toolkitFsWrapper(embedToolkitFS)

var initToolkit = func() {
	logger.Debug("Using embedded toolkit", log.Ctx{})
}
