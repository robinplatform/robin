package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type RobinAppConfig struct {
	// ConfigPath is the absolute path to the config file that was used to load this config
	ConfigPath string `json:"-"`

	// Id of the app
	Id string `json:"id"`
	// Name of the app
	Name string `json:"name"`
	// PageIcon refers to the path to the icon to use for this app
	PageIcon string `json:"pageIcon"`
	// Page refers to the path to the page to load for this app
	Page string `json:"page"`
}

func (appConfig *RobinAppConfig) validate() error {
	if appConfig.Id == "" {
		return fmt.Errorf("'id' is required")
	}

	if appConfig.Page == "" {
		return fmt.Errorf("'page' is required")
	}
	if !path.IsAbs(appConfig.Page) {
		appConfig.Page = path.Clean(path.Join(path.Dir(appConfig.ConfigPath), appConfig.Page))
	}
	if _, err := os.Stat(appConfig.Page); err != nil {
		return fmt.Errorf("failed to find page '%s': %s", appConfig.Page, err)
	}

	if appConfig.PageIcon == "" {
		return fmt.Errorf("'pageIcon' is required")
	}

	if appConfig.Name == "" {
		return fmt.Errorf("'name' is required")
	}

	return nil
}

type robinProjectConfig struct {
	// Name of the app
	Name string `json:"name"`
	// Apps to load for this project
	Apps []string `json:"apps"`
}

type RobinProjectConfig struct {
	// Name of the app
	Name string
	// Apps to load for this project
	Apps []RobinAppConfig
}

func LoadRobinProjectConfig() (RobinProjectConfig, error) {
	projectPath, err := GetProjectPath()
	if err != nil {
		return RobinProjectConfig{}, err
	}

	storedConfig := robinProjectConfig{}
	parsedConfig := RobinProjectConfig{}
	configPath := path.Join(projectPath, "robin.json")

	buf, err := os.ReadFile(configPath)
	if err != nil {
		return parsedConfig, fmt.Errorf("failed to read robin.json: %s", err)
	}

	err = json.Unmarshal(buf, &storedConfig)
	if err != nil {
		return parsedConfig, fmt.Errorf("failed to parse robin.json: %s", err)
	}

	parsedConfig.Name = storedConfig.Name
	parsedConfig.Apps = make([]RobinAppConfig, len(storedConfig.Apps))
	for i, appConfigPath := range storedConfig.Apps {
		if path.Base(appConfigPath) != "robin.app.json" {
			appConfigPath = path.Join(appConfigPath, "robin.app.json")
		}
		if !path.IsAbs(appConfigPath) {
			appConfigPath = path.Clean(path.Join(projectPath, appConfigPath))
		}

		buf, err := os.ReadFile(appConfigPath)
		if err != nil {
			return parsedConfig, fmt.Errorf("failed to read app config from '%s': %s", appConfigPath, err)
		}

		err = json.Unmarshal(buf, &parsedConfig.Apps[i])
		if err != nil {
			return parsedConfig, fmt.Errorf("failed to parse app config from '%s': %s", appConfigPath, err)
		}

		parsedConfig.Apps[i].ConfigPath = appConfigPath

		if err := parsedConfig.Apps[i].validate(); err != nil {
			return parsedConfig, fmt.Errorf("invalid robin app config in '%s': %s", appConfigPath, err)
		}
	}

	return parsedConfig, nil
}

func LoadRobinAppById(appId string) (RobinAppConfig, error) {
	projectConfig, err := LoadRobinProjectConfig()
	if err != nil {
		return RobinAppConfig{}, err
	}

	for _, appConfig := range projectConfig.Apps {
		if appConfig.Id == appId {
			return appConfig, nil
		}
	}

	return RobinAppConfig{}, fmt.Errorf("failed to find app with id '%s'", appId)
}
