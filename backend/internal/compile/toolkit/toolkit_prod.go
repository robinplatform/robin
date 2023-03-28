//go:build toolkit && prod

package toolkit

import (
	"embed"
	"io/fs"
	"path"

	"robinplatform.dev/internal/log"
)

type toolkitFsWrapper embed.FS

func (e toolkitFsWrapper) Open(name string) (fs.File, error) {
	// embed.FS always uses forward slashes
	return embed.FS(e).Open(path.Join("toolkit", name))
}

//go:generate rm -rf toolkit
//go:generate rsync -rv --exclude=node_modules --exclude=robin.servers.json ../../../../toolkit toolkit
//go:embed all:toolkit
var embedToolkitFS embed.FS
var ToolkitFS fs.FS = toolkitFsWrapper(embedToolkitFS)

var initToolkit = func() {
	logger.Debug("Using embedded toolkit", log.Ctx{})
}
