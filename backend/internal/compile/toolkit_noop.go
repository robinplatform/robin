//go:build !toolkit && prod

package compile

import (
	"io/fs"
)

var toolkitFS fs.FS
var initToolkit = func() {}
