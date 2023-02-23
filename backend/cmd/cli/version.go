package main

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/pflag"
	"robinplatform.dev/internal/config"
)

type VersionCommand struct{}

func (c *VersionCommand) Name() string {
	return "version"
}

func (c *VersionCommand) Description() string {
	return "Print the version info of robin"
}

func (c *VersionCommand) Parse(flagSet *pflag.FlagSet, args []string) error {
	return nil
}

func (c *VersionCommand) Run() error {
	versionInfo := map[string]string{
		"version": config.GetRobinVersion(),
		"channel": string(config.GetReleaseChannel()),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}

	buf, err := json.MarshalIndent(versionInfo, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal version info: %w", err)
	}

	fmt.Printf("%s\n", buf)
	return nil
}
