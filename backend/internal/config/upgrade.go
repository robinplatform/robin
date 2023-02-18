package config

import (
	"fmt"
)

type ReleaseChannel string

const (
	Stable  ReleaseChannel = "stable"
	Beta    ReleaseChannel = "beta"
	Nightly ReleaseChannel = "nightly"
)

func (channel *ReleaseChannel) UnmarshalJSON(buf []byte) error {
	value := string(buf)

	switch value {
	case `"stable"`:
		*channel = Stable
	case `"beta"`:
		*channel = Beta
	case `"nightly"`:
		*channel = Nightly

	default:
		return fmt.Errorf("invalid release channel: '%s'", value)
	}

	return nil
}
