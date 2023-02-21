//go:build prod

package config

import (
	_ "embed"
	"fmt"
)

//go:generate cp ../../package.json .
//go:embed package.json
var rawPackageJsonRaw []byte

// rootPackageJson is the parsed version of `rawPackageJsonRaw`, but we will
// probably get rid of all this sillyness once we figure out stable releases.
var rootPackageJson PackageJson

func init() {
	if err := ParsePackageJson(rawPackageJsonRaw, &rootPackageJson); err != nil {
		panic(fmt.Errorf("failed to load package.json: %w", err))
	}
}

func GetRobinVersion() string {
	return rootPackageJson.Version
}

func GetReleaseChannel() (ReleaseChannel, error) {
	projectConfig, err := LoadProjectConfig()
	if err != nil {
		return ReleaseChannelStable, err
	}
	return projectConfig.ReleaseChannel, nil
}
