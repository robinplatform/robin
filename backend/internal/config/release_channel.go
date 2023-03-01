package config

import (
	"fmt"
	"path/filepath"
)

type ReleaseChannel string

const (
	ReleaseChannelStable  ReleaseChannel = "stable"
	ReleaseChannelBeta    ReleaseChannel = "beta"
	ReleaseChannelNightly ReleaseChannel = "nightly"
	ReleaseChannelDev     ReleaseChannel = "dev"
)

func (channel *ReleaseChannel) Parse(value string) error {
	switch value {
	case "stable":
		*channel = ReleaseChannelStable
	case "beta":
		*channel = ReleaseChannelBeta
	case "nightly":
		*channel = ReleaseChannelNightly

	default:
		return fmt.Errorf("invalid release channel: '%s'", value)
	}

	return nil
}

func (channel *ReleaseChannel) UnmarshalJSON(buf []byte) error {
	// Assume that it is formatted as a JSON string, and remove the quotes.
	value := string(buf)
	value = value[1 : len(value)-1]
	return channel.Parse(value)
}

func (channel ReleaseChannel) GetPath() string {
	return filepath.Join(robinPath, string(channel))
}

func (channel ReleaseChannel) String() string {
	return string(channel)
}
