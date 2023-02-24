package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/config"
)

type RemoveCommand struct {
	targetApps []string
}

func (cmd *RemoveCommand) Name() string {
	return "rm"
}

func (cmd *RemoveCommand) Description() string {
	return "Removes a robin app from the active project"
}

func (*RemoveCommand) ShortUsage() string {
	return "rm [apps ...]"
}

func (cmd *RemoveCommand) Parse(flags *pflag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return err
	}

	cmd.targetApps = flags.Args()
	if len(cmd.targetApps) == 0 {
		return fmt.Errorf("no apps specified to add")
	}

	return nil
}

func (cmd *RemoveCommand) Run() error {
	projectPath := config.GetProjectPathOrExit()

	apps, err := compile.GetAllProjectApps()
	if err != nil {
		return fmt.Errorf("failed to load project apps: %w", err)
	}

	existingApps := make(map[string]compile.RobinAppConfig, len(apps)*3)
	for _, app := range apps {
		existingApps[app.Id] = app
		existingApps[app.Name] = app
		existingApps[app.ConfigPath.String()] = app
	}

	rmTargetIds := make(map[string]bool, len(cmd.targetApps))
	for _, appPattern := range cmd.targetApps {
		if matchingApp, ok := existingApps[appPattern]; ok {
			fmt.Printf("Removing: %s\n", matchingApp.Name)
			rmTargetIds[matchingApp.Id] = true
		} else {
			if appPattern[0] != '.' && appPattern[0] != '/' {
				appPattern = fmt.Sprintf("https://esm.sh/%s", appPattern)
			}

			appConfig, err := compile.LoadRobinAppByPath(appPattern)
			if err != nil {
				return fmt.Errorf("unrecognized app: %s", appPattern)
			}

			if matchingApp, ok := existingApps[appConfig.Id]; ok {
				fmt.Printf("Removing: %s\n", matchingApp.Name)
				rmTargetIds[matchingApp.Id] = true
			} else {
				fmt.Printf("Not installed: %s\n", appConfig.Name)
			}
		}
	}

	projectConfig := config.RobinProjectConfig{}
	if err := projectConfig.LoadRobinProjectConfig(); err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	newApps := make([]string, 0, len(projectConfig.Apps))
	for _, app := range projectConfig.Apps {
		appConfig, err := compile.LoadRobinAppByPath(app)
		if err != nil {
			return fmt.Errorf("failed to load app config: %w", err)
		}

		if _, ok := rmTargetIds[appConfig.Id]; !ok {
			if appConfig.ConfigPath.Scheme == "file" {
				relpath, err := filepath.Rel(projectPath, appConfig.ConfigPath.Path)
				if err != nil {
					return fmt.Errorf("failed to get relative path of %s: %w", appConfig.ConfigPath.Path, err)
				}

				newApps = append(newApps, "."+string(filepath.Separator)+relpath)
			} else {
				newApps = append(newApps, appConfig.ConfigPath.String())
			}
		}
	}
	projectConfig.Apps = newApps

	if err := projectConfig.SaveRobinProjectConfig(); err != nil {
		return fmt.Errorf("failed to save project config: %w", err)
	}

	return nil
}
