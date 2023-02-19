package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

type Command interface {
	Name() string
	Description() string

	// Parse is given an allocated flagSet, and the set of args that are specific to this command.
	// It should parse the args, and return an error if unexpected values were received in the flags.
	Parse(flagSet *pflag.FlagSet, args []string) error

	// Run should run the command, and return an error if something went wrong.
	Run() error
}

var (
	commands = []Command{
		&StartCommand{},
		&VersionCommand{},
	}
)

func showUsageFooter() {
	fmt.Fprintf(os.Stderr, "\n")

	releaseChannel := "unknown"
	projectConfig, err := config.LoadProjectConfig()
	if err == nil {
		releaseChannel = string(projectConfig.ReleaseChannel)
	}

	fmt.Fprintf(
		os.Stderr,
		"robin v%s on %s\n",
		config.GetRobinVersion(),
		releaseChannel,
	)
	fmt.Fprintf(os.Stderr, "\n")
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "Usage: robin [-p $ROBIN_PROJECT_PATH] [command] [options]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "All commands must be run from a valid robin project directory, or a sub-directory of a valid robin project directory.\n")
	fmt.Fprintf(os.Stderr, "You can also override the project using the `-p` flag or the `ROBIN_PROJECT_PATH` environment variable.\n")
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "Available commands:\n\n")

	longestCmdNameLength := 0
	for _, cmd := range commands {
		if len(cmd.Name()) > longestCmdNameLength {
			longestCmdNameLength = len(cmd.Name())
		}
	}

	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "\t%s\t%s%s\n", cmd.Name(), strings.Repeat(" ", 1+longestCmdNameLength-len(cmd.Name())), cmd.Description())
	}

	showUsageFooter()
	os.Exit(1)
}

func showCommandUsage(cmd Command, flagSet *pflag.FlagSet) {
	fmt.Fprintf(os.Stderr, "Usage: robin %s [options]\n", cmd.Name())
	fmt.Fprintf(os.Stderr, "%s\n", cmd.Description())
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "Options:\n\n")
	fmt.Fprintf(os.Stderr, "%s\n", flagSet.FlagUsages())
	fmt.Fprintf(os.Stderr, "\n")

	showUsageFooter()
	os.Exit(1)
}

func main() {
	fmt.Printf("\n")
	args := os.Args[1:]

	logger := log.New("cli")

	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		showUsage()
	}

	// First, allow a project path override to take place
	if args[0] == "--projectPath" || args[0] == "-p" {
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "missing argument for --projectPath flag\n")
			showUsage()
		}

		projectPath := args[1]
		if projectPath[0] != '/' {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to get cwd: %s\n", err)
				os.Exit(1)
			}

			projectPath = cwd + "/" + projectPath
		}

		config.SetProjectPath(projectPath)
		args = args[2:]
	}

	// Next, verify that a command was given
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "missing command\n")
		showUsage()
	}

	commandName := args[0]
	args = args[1:]

	// Make sure this is a real command
	var command Command
	for _, cmd := range commands {
		if cmd.Name() == commandName {
			command = cmd
			break
		}
	}

	logger.Debug("Parsed Args, found command", log.Ctx{
		"command": command.Name(),
	})
	if command == nil {
		fmt.Fprintf(os.Stderr, "unrecognized command: %s\n", commandName)
		showUsage()
	}

	// Perform parsing
	flagSet := pflag.NewFlagSet(commandName, pflag.ExitOnError)
	if err := command.Parse(flagSet, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		showCommandUsage(command, flagSet)
	}

	// Run the command
	startTime := time.Now()
	if err := command.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n\n", err)
		os.Exit(1)
	}

	execDuration := (time.Since(startTime) + time.Millisecond).Truncate(time.Millisecond)
	fmt.Printf("\n⚡️%s completed in %s\n", commandName, execDuration)
}
