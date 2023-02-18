package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"robin.dev/internal/config"
	"robin.dev/internal/server"
)

type StartCommand struct {
	port        int
	bindAddress string
}

func (cmd *StartCommand) Parse(flagSet *pflag.FlagSet, args []string) error {
	flagSet.IntVar(&cmd.port, "port", 9010, "The port to listen on")
	flagSet.StringVar(&cmd.bindAddress, "bind", "[::1]", "The address to bind to")

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
	fmt.Printf("Project path: %s\n", config.GetProjectPathOrExit())
	fmt.Printf("Robin PID: %d\n", os.Getpid())

	app := server.Server{}
	return app.Run("[::1]:9010")
}
