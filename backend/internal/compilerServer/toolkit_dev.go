//go:build !prod

package compilerServer

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"robinplatform.dev/internal/log"
)

var toolkitFS fs.FS

var initToolkit = func() {
	_, filename, _, _ := runtime.Caller(0)
	toolkitPath := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "toolkit"))
	toolkitFS = os.DirFS(toolkitPath)
	logger.Warn("Detected dev mode, using local toolkit", log.Ctx{
		"path": toolkitPath,
	})
}
