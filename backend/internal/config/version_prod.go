//go:build prod

package config

import (
	_ "embed"
	"fmt"
)

//go:generate cp ../../package.json .

var (
	//go:embed package.json
	rawPackageJsonRaw []byte
	rootPackageJson   PackageJson
)

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
