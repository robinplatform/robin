package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/config"
)

type AddCommand struct {
	targetApps []string
}

func (cmd *AddCommand) Name() string {
	return "add"
}

func (cmd *AddCommand) Description() string {
	return "Installs a robin app into the active project"
}

func (*AddCommand) ShortUsage() string {
	return "add [apps ...]"
}

func (cmd *AddCommand) Parse(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return err
	}

	cmd.targetApps = flags.Args()
	if len(cmd.targetApps) == 0 {
		return fmt.Errorf("no apps specified to add")
	}

	return nil
}

var eraseEndLine = "\u001B[K"

func (cmd *AddCommand) Run() error {
	projectPath := config.GetProjectPathOrExit()

	existingApps, err := compile.GetAllProjectApps()
	if err != nil {
		return fmt.Errorf("failed to get existing apps: %w", err)
	}

	defer fmt.Printf("%s", eraseEndLine)

	for _, appPath := range cmd.targetApps {
		fmt.Printf("Adding: %s\r", appPath)
		var resolvedAppPath string

		if strings.HasPrefix(appPath, "http:") || strings.HasPrefix(appPath, "https:") {
			resolvedAppPath = appPath
		} else if appPath[0] == '.' || appPath[0] == '/' {
			var err error

			absPath, err := filepath.Abs(appPath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path for %s: %w", appPath, err)
			}

			resolvedAppPath, err = filepath.Rel(projectPath, absPath)
			if err != nil {
				return fmt.Errorf("failed to get relative path for %s: %w", absPath, err)
			}

			resolvedAppPath = "." + string(filepath.Separator) + resolvedAppPath
		} else {
			resolvedAppPath = fmt.Sprintf("https://esm.sh/%s", appPath)
		}

		// Load and verify the app config
		appConfig, err := compile.LoadRobinAppByPath(resolvedAppPath)
		if err != nil {
			return fmt.Errorf("failed to load app config: %w", err)
		}

		// Reload and resave the project config each time, so that if we are slow and the user
		// makes changes, we don't force the user to pick between their changes and ours. Unlike
		// certain programs. *cough* *cough* *npm* *cough* *cough*
		projectConfig := config.RobinProjectConfig{}
		if err := projectConfig.LoadRobinProjectConfig(projectPath); err != nil {
			return fmt.Errorf("failed to load project config: %w", err)
		}

		// Check if the app is already in the project
		isAppInstalled := false
		for _, existingAppPath := range projectConfig.Apps {
			if existingAppPath == resolvedAppPath {
				fmt.Printf("App %s is already in the project\n", resolvedAppPath)
				isAppInstalled = true
				break
			}
		}
		if !isAppInstalled {
			for _, existingApp := range existingApps {
				if existingApp.Id == appConfig.Id {
					fmt.Printf("%s has the same ID as existing app: %s\n", resolvedAppPath, existingApp.ConfigPath)
					isAppInstalled = true
					break
				}
			}
		}

		if !isAppInstalled {
			projectConfig.Apps = append(projectConfig.Apps, resolvedAppPath)

			if err := projectConfig.SaveRobinProjectConfig(); err != nil {
				return fmt.Errorf("failed to save project config: %w", err)
			}

			existingApps = append(existingApps, appConfig)
			fmt.Printf("Added: %s as %s\n", appPath, resolvedAppPath)
		}
	}

	return nil
}
