//go:build !prod

package config

func GetRobinVersion() string {
	return "v0.0.0"
}

func GetReleaseChannel() ReleaseChannel {
	return ReleaseChannelDev
}
