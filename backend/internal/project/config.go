package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"robinplatform.dev/internal/config"
)

var (
	pathRegex *regexp.Regexp = regexp.MustCompile("[^a-zA-Z0-9]+")
)

func GetProjectName() (string, error) {
	projectConfig, err := LoadFromEnv()
	return projectConfig.Name, err
}

func (projectConfig *RobinProjectConfig) GetProjectAlias() string {
	// Remove all non alphanumeric characters from 'projectName' so it is a safe directory name
	return pathRegex.ReplaceAllString(projectConfig.Name, "")
}

func GetProjectAlias() (string, error) {
	projectConfig, err := LoadFromEnv()
	if err != nil {
		return "", err
	}

	return projectConfig.GetProjectAlias(), nil
}

type RobinConfig struct {
	// Environments is a map of environment names to a map of environment variables
	Environments map[string]map[string]string `json:"environments"`

	// AppSettings is a map of app IDs to the respective app settings
	AppSettings map[string]map[string]any `json:"appSettings"`

	// KeyMappings is a map of key mappings
	KeyMappings map[string]string `json:"keyMappings"`
}

var defaultRobinConfig = RobinConfig{}

func LoadProjectConfig() (RobinConfig, error) {
	alias, err := GetProjectAlias()
	if err != nil {
		return defaultRobinConfig, err
	}

	robinPath := config.GetRobinPath()
	robinConfigPath := filepath.Join(robinPath, "projects", alias, "config.json")

	// Load the config file from robinConfigPath
	configFileBuf, err := os.ReadFile(robinConfigPath)
	if os.IsNotExist(err) {
		return defaultRobinConfig, nil
	}
	if err != nil {
		return defaultRobinConfig, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the config file
	var config RobinConfig
	if err := json.Unmarshal(configFileBuf, &config); err != nil {
		return defaultRobinConfig, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return config, nil
}

func UpdateProjectConfig(projectConfig RobinConfig) error {
	alias, err := GetProjectAlias()
	if err != nil {
		return fmt.Errorf("failed to get project name: %w", err)
	}

	robinPath := config.GetRobinPath()
	robinConfigPath := filepath.Join(robinPath, "projects", alias, "config.json")

	// Marshal the config file
	buf, err := json.Marshal(&projectConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config file: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(robinPath, "projects", alias), 0777); err != nil {
		return fmt.Errorf("failed to create folder for config file: %w", err)
	}

	// Save the file
	if err := os.WriteFile(robinConfigPath, buf, 0755); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
