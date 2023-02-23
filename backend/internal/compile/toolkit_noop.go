//go:build !toolkit && prod

package compile

import (
	"io/fs"
	"os"
	"path/filepath"
)

var toolkitPath = filepath.Join(os.TempDir(), "robin-toolkit")
var toolkitFS fs.FS
