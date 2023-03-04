package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/project"
)

type Command interface {
	Name() string
	Description() string

	// Parse is given an allocated flagSet, and the set of args that are specific to this command.
	// It should parse the args, and return an error if unexpected values were received in the flags.
	Parse(flagSet *flag.FlagSet, args []string) error

	// Run should run the command, and return an error if something went wrong.
	Run() error
}

var (
	commands = []Command{
		&StartCommand{},
		&AddCommand{},
		&RemoveCommand{},
		&CreateCommand{},
		&VersionCommand{},
	}
)

func showUsageFooter() {
	fmt.Fprintf(os.Stderr, "\n")

	releaseChannel := config.GetReleaseChannel()
	fmt.Fprintf(
		os.Stderr,
		"robin %s on %s\n\n",
		config.GetRobinVersion(),
		releaseChannel,
	)
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "\n")
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
		fmt.Fprintf(os.Stderr, "\t%s%s\t%s\n", cmd.Name(), strings.Repeat(" ", longestCmdNameLength-len(cmd.Name())), cmd.Description())
	}

	showUsageFooter()
	os.Exit(1)
}

func showCommandUsage(cmd Command, flagSet *flag.FlagSet) {
	shortUsage := fmt.Sprintf("%s [options]", cmd.Name())

	// allow commands to override the short usage text
	if cmdWithShortUsage, ok := cmd.(interface{ ShortUsage() string }); ok {
		shortUsage = cmdWithShortUsage.ShortUsage()
	}

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Usage: robin %s\n", shortUsage)
	fmt.Fprintf(os.Stderr, "%s\n", cmd.Description())
	fmt.Fprintf(os.Stderr, "\n")

	var hasFlags bool
	flagSet.VisitAll(func(f *flag.Flag) {
		hasFlags = true
	})

	if hasFlags {
		flagSet.PrintDefaults()
	} else {
		fmt.Fprintf(os.Stderr, "This command has no options.\n")
	}

	showUsageFooter()
	os.Exit(1)
}

func main() {
	args := os.Args[1:]

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

		project.SetProjectPath(projectPath)
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

	if command == nil {
		fmt.Fprintf(os.Stderr, "unrecognized command: %s\n\n", commandName)
		showUsage()
	}

	// Perform parsing
	flagSet := flag.NewFlagSet(commandName, flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n")
		showCommandUsage(command, flagSet)
	}
	if err := command.Parse(flagSet, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		showCommandUsage(command, flagSet)
	}

	// Run the command
	fmt.Printf("\n")
	startTime := time.Now()
	if err := command.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n\n", err)
		os.Exit(1)
	}

	execDuration := (time.Since(startTime) + time.Millisecond).Truncate(time.Millisecond)
	fmt.Printf("\n⚡️%s completed in %s\n", commandName, execDuration)
}
