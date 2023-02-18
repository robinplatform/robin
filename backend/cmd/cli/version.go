package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/pflag"
	"robin.dev/internal/config"
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
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}

	if _, err := config.GetProjectPath(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN: Not in a robin project, cannot determine release channel\n")
	} else {
		releaseChannel, err := config.GetReleaseChannel()
		if err != nil {
			return fmt.Errorf("failed to get release channel: %w", err)
		}
		versionInfo["channel"] = string(releaseChannel)
	}

	buf, err := json.MarshalIndent(versionInfo, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal version info: %w", err)
	}

	fmt.Printf("%s\n", buf)
	return nil
}
