package main

import (
	"fmt"

	"github.com/spf13/pflag"
	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/server"
)

type StartCommand struct {
	port               int
	bindAddress        string
	forceStableToolkit bool
}

func (cmd *StartCommand) Parse(flagSet *pflag.FlagSet, args []string) error {
	flagSet.IntVar(&cmd.port, "port", 9010, "The port to listen on")
	flagSet.StringVar(&cmd.bindAddress, "bind", "[::1]", "The address to bind to")

	releaseChannel, _ := config.GetReleaseChannel()
	if releaseChannel != config.ReleaseChannelStable {
		flagSet.BoolVar(&cmd.forceStableToolkit, "use-stable-toolkit", false, "Force the use of the stable toolkit")
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}
	if cmd.port < 1 || cmd.port > 65535 {
		return fmt.Errorf("invalid port number: %d", cmd.port)
	}

	return nil
}

func (cmd *StartCommand) Name() string {
	return "start"
}

func (cmd *StartCommand) Description() string {
	return "Start the robin server"
}

func (cmd *StartCommand) Run() error {
	if cmd.forceStableToolkit {
		compile.DisableEmbeddedToolkit()
	}

	app := server.Server{}
	return app.Run("[::1]:9010")
}
