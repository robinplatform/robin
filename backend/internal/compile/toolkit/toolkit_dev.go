//go:build !prod

package toolkit

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"robinplatform.dev/internal/log"
)

var ToolkitFS fs.FS

var initToolkit = func() {
	_, filename, _, _ := runtime.Caller(0)
	toolkitPath := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "toolkit"))
	ToolkitFS = os.DirFS(toolkitPath)
	logger.Warn("Detected dev mode, using local toolkit", log.Ctx{
		"path": toolkitPath,
	})
}
