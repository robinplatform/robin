//go:build prod

package config

// robinVersion is the version of the running robin instance. The value is set
// by the go linker during the build process.
var robinVersion string

func GetRobinVersion() string {
	return robinVersion
}

func GetReleaseChannel() (ReleaseChannel, error) {
	projectConfig, err := LoadProjectConfig()
	if err != nil {
		return ReleaseChannelStable, err
	}
	return projectConfig.ReleaseChannel, nil
}
