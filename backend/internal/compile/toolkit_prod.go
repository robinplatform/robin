//go:build toolkit && prod

package compile

import (
	"embed"
	"os"
	"path/filepath"

	es "github.com/evanw/esbuild/pkg/api"
	"robinplatform.dev/internal/log"
)

var toolkitPath = filepath.Join(os.TempDir(), "robin-toolkit")

//go:generate cp -R ../../toolkit toolkit
//go:embed all:toolkit
var toolkitFS embed.FS

func init() {
	logger.Debug("Using embedded toolkit", log.Ctx{})
}
