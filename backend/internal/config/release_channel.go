package config

import (
	"fmt"
	"path"
)

type ReleaseChannel string

const (
	ReleaseChannelStable  ReleaseChannel = "stable"
	ReleaseChannelBeta    ReleaseChannel = "beta"
	ReleaseChannelNightly ReleaseChannel = "nightly"
)

func (channel *ReleaseChannel) UnmarshalJSON(buf []byte) error {
	value := string(buf)

	switch value {
	case `"stable"`:
		*channel = ReleaseChannelStable
	case `"beta"`:
		*channel = ReleaseChannelBeta
	case `"nightly"`:
		*channel = ReleaseChannelNightly

	default:
		return fmt.Errorf("invalid release channel: '%s'", value)
	}

	return nil
}

func (channel *ReleaseChannel) GetPath() string {
	return path.Join(robinPath, string(*channel))
}
