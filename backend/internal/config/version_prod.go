//go:build prod

package config

import (
	"fmt"
)

// robinVersion is the version of the running robin instance. The value is set
// by the go linker during the build process.
var robinVersion string

// builtReleaseChannel is the release channel that the robin binary was built
// for. The value is set by the go linker during the build process.
var builtReleaseChannel string
var activeReleaseChannel ReleaseChannel

func init() {
	if robinVersion == "" || builtReleaseChannel == "" {
		panic("robinVersion and activeReleaseChannel must be set by the go linker")
	}

	if err := activeReleaseChannel.Parse(builtReleaseChannel); err != nil {
		panic(fmt.Errorf("robin compiled with invalid release channel: %w", err))
	}
}

func GetRobinVersion() string {
	return robinVersion
}

func GetReleaseChannel() ReleaseChannel {
	return activeReleaseChannel
}
