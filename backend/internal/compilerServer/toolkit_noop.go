//go:build !toolkit && prod

package compilerServer

import (
	"io/fs"
)

var toolkitFS fs.FS
var initToolkit = func() {}
