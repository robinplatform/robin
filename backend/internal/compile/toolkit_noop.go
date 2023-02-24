//go:build !toolkit && prod

package compile

import (
	"io/fs"
)

var toolkitPath = ""
var toolkitFS fs.FS
var initToolkit = func() {}
