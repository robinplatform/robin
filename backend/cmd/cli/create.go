package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"robinplatform.dev/internal/project"
)

//go:embed all:app-template
var appTemplate embed.FS

type CreateCommand struct {
	targetPath     string
	appId          string
	appName        string
	skipInstall    bool
	packageManager string
}

func (cmd *CreateCommand) Name() string {
	return "create"
}

func (cmd *CreateCommand) Description() string {
	return "Create a new robin app"
}

func (*CreateCommand) ShortUsage() string {
	return "create [path to your app]"
}

func (cmd *CreateCommand) Parse(flags *flag.FlagSet, args []string) error {
	flags.StringVar(&cmd.appId, "id", "", "the id of your app (defaults to the name of the directory)")
	flags.StringVar(&cmd.appName, "name", "", "the name of your app (defaults to the same as ID)")
	flags.BoolVar(&cmd.skipInstall, "skip-install", false, "skip installing dependencies")
	flags.StringVar(&cmd.packageManager, "package-manager", "", "the package manager to use (defaults to yarn if available, otherwise npm)")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if len(flags.Args()) != 1 {
		return fmt.Errorf("you must specify an app path")
	}
	cmd.targetPath = flags.Arg(0)

	if cmd.appId == "" {
		cmd.appId = filepath.Base(cmd.targetPath)
	}
	if cmd.appName == "" {
		cmd.appName = cmd.appId
	}

	return nil
}

func (cmd *CreateCommand) installDeps(args ...string) error {
	cmdArgs := []string{"add", "--exact"}
	if cmd.packageManager == "npm" {
		cmdArgs = []string{"install", "--save", "--exact"}
	}
	cmdArgs = append(cmdArgs, args...)

	cmdInstall := exec.Command(cmd.packageManager, cmdArgs...)
	cmdInstall.Dir = cmd.targetPath
	cmdInstall.Stdout = os.Stdout
	cmdInstall.Stderr = os.Stderr
	if err := cmdInstall.Run(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}
	return nil
}

func (cmd *CreateCommand) Run() error {
	if !filepath.IsAbs(cmd.targetPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		cmd.targetPath = filepath.Join(cwd, cmd.targetPath)
	}

	var excitingEmojis = []string{"👋", "🎉", "🎊", "🎈", "🎁", "🎀", "🚀"}
	var randomEmoji = excitingEmojis[rand.Intn(len(excitingEmojis))]

	templateData := project.RobinAppConfig{
		Id:       cmd.appId,
		Name:     cmd.appName,
		PageIcon: randomEmoji,
		Page:     "./src/app.tsx",
	}

	// Make sure that the target path doesn't already exist
	if _, err := os.Stat(cmd.targetPath); os.IsNotExist(err) {
		// this is good
	} else if err != nil {
		return fmt.Errorf("failed to check if target path exists: %w", err)
	} else {
		return fmt.Errorf("target path already exists")
	}

	// Create the target path
	if err := os.MkdirAll(cmd.targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target path: %w", err)
	}

	err := fs.WalkDir(appTemplate, "app-template", func(templateFilePath string, dirEntry fs.DirEntry, err error) error {
		outputFilePath := filepath.Join(cmd.targetPath, filepath.FromSlash(strings.TrimPrefix(templateFilePath, "app-template/")))

		if err != nil {
			return fmt.Errorf("failed to walk template directory: %w", err)
		}
		if templateFilePath == "app-template" {
			return nil
		}
		if dirEntry.IsDir() {
			return os.MkdirAll(outputFilePath, 0755)
		}

		fd, err := appTemplate.Open(templateFilePath)
		if err != nil {
			return fmt.Errorf("failed to open template file: %w", err)
		}

		buf, err := io.ReadAll(fd)
		if err != nil {
			return fmt.Errorf("failed to read template file: %w", err)
		}

		tmpl, err := template.New(templateFilePath).Parse(string(buf))
		if err != nil {
			return fmt.Errorf("failed to parse template file: %w", err)
		}

		output, err := os.Create(outputFilePath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		if err := tmpl.Execute(output, templateData); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !cmd.skipInstall {
		if cmd.packageManager == "" {
			cmd.packageManager = "yarn"
			if _, err := exec.LookPath("yarn"); err != nil {
				cmd.packageManager = "npm"
			}
		}

		// install dependencies live, so we get pinned versions
		if err := cmd.installDeps("react", "react-dom", "@robinplatform/toolkit"); err != nil {
			return err
		}
		if err := cmd.installDeps("-D", "@types/node", "@types/react", "rome", "typescript"); err != nil {
			return err
		}
	}

	fmt.Printf("\nCreated new app in: %s\n", cmd.targetPath)
	return nil
}
