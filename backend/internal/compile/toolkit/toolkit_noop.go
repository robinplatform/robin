//go:build !toolkit && prod

package toolkit

import (
	"io/fs"
)

var ToolkitFS fs.FS
var initToolkit = func() {}
