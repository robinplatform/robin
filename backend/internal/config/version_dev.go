//go:build !prod

package config

func GetRobinVersion() string {
	return "0.0.0"
}

func GetReleaseChannel() (ReleaseChannel, error) {
	return ReleaseChannelDev, nil
}
