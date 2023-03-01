package main

import (
	"flag"
	"fmt"
	"runtime"

	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/server"
	"robinplatform.dev/internal/upgrade"
)

type StartCommand struct {
	port               int
	bindAddress        string
	forceStableToolkit bool
}

func (cmd *StartCommand) Name() string {
	return "start"
}

func (cmd *StartCommand) Description() string {
	return "Start the robin server"
}

func (cmd *StartCommand) Parse(flagSet *flag.FlagSet, args []string) error {
	flagSet.IntVar(&cmd.port, "port", 9010, "The port to listen on")
	flagSet.StringVar(&cmd.bindAddress, "bind", "[::1]", "The address to bind to")

	if config.GetReleaseChannel() != config.ReleaseChannelStable {
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

func (cmd *StartCommand) Run() error {
	releaseChannel := config.GetReleaseChannel()
	if runtime.GOOS != "windows" && releaseChannel != config.ReleaseChannelDev {
		go upgrade.WatchForUpdates()
	}

	if cmd.forceStableToolkit {
		compile.DisableEmbeddedToolkit()
	}

	app := server.Server{
		BindAddress: cmd.bindAddress,
		Port:        cmd.port,
	}
	return app.Run()
}
