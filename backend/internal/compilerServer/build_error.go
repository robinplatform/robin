package compilerServer

import (
	"fmt"
	"strings"

	es "github.com/evanw/esbuild/pkg/api"
)

type BuildError es.BuildResult

func getEsbuildErrorString(err es.Message) string {
	errMessage := err.Text

	if err.PluginName != "" {
		errMessage = fmt.Sprintf("%s: %s", err.PluginName, errMessage)
	}

	if err.Location != nil {
		pluginNameEnd := strings.Index(err.Location.File, ":")
		if pluginNameEnd == -1 {
			errMessage = fmt.Sprintf("%s in %s:%d:%d", errMessage, err.Location.File, err.Location.Line, err.Location.Column)
		} else {
			errMessage = fmt.Sprintf("%s in %s:%d:%d", errMessage, err.Location.File[pluginNameEnd+1:], err.Location.Line, err.Location.Column)
		}
	}

	return errMessage
}

func (beResult BuildError) Error() string {
	result := es.BuildResult(beResult)
	if len(result.Errors) == 0 {
		return "Unknown error"
	}

	errMessage := fmt.Sprintf("Build failed with %d errors:\n", len(result.Errors))

	for _, err := range result.Errors {
		errMessage += "\n - " + getEsbuildErrorString(err)
	}

	return errMessage
}
