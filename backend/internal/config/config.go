package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
)

var (
	robinPath       string
	robinConfigPath string

	// This should only be loaded once, since robin will with a target project
	projectName string
	// This is a path-safe version of 'projectName'
	projectAlias string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to get user home directory: %w", err))
	}

	robinPath = path.Join(home, ".robin")

	// If it doesn't exist, create it
	if _, err := os.Stat(robinPath); os.IsNotExist(err) {
		if err := os.Mkdir(robinPath, 0755); err != nil {
			panic(fmt.Errorf("failed to create robin directory: %w", err))
		}
	}
}

func GetProjectName() string {
	if projectName != "" {
		return projectName
	}

	packageJsonPath := path.Join(GetProjectPath(), "package.json")
	packageJson, err := LoadPackageJson(packageJsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load %s: %v", packageJsonPath, err)
		os.Exit(1)
	}
	projectName = packageJson.Name

	// Remove all non alphanumeric characters from 'projectName' so it is a safe directory name
	projectAlias = regexp.MustCompile("[^a-zA-Z0-9]+").ReplaceAllString(projectName, "")

	return projectName
}

func GetProjectAlias() string {
	if projectAlias == "" {
		GetProjectName()
	}
	return projectAlias
}

type RobinConfig struct {
	// ReleaseChannel is the release channel to use for upgrades
	ReleaseChannel ReleaseChannel `json:"releaseChannel"`

	// Environments is a map of environment names to a map of environment variables
	Environments map[string]map[string]string `json:"environments"`

	// Extensions is a map of extension names to a map of extension settings
	Extensions map[string]map[string]interface{} `json:"extensions"`

	// ShowReactQueryDebugger is a flag to show the react-query debugger
	ShowReactQueryDebugger bool `json:"showReactQueryDebugger"`

	// MinifyExtensionClients is a flag to minify extension clients
	MinifyExtensionClients bool `json:"minifyExtensionClients"`

	// KeyMappings is a map of key mappings
	KeyMappings map[string]string `json:"keyMappings"`

	// EnableKeyMappings is a flag to enable key mappings
	EnableKeyMappings bool `json:"enableKeyMappings"`
}

func LoadProjectConfig() (*RobinConfig, error) {
	var config RobinConfig
	robinConfigPath = path.Join(robinPath, GetProjectAlias(), "config.json")

	// Load the config file from robinConfigPath
	configFileBuf, err := ioutil.ReadFile(robinConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the config file
	if err := json.Unmarshal(configFileBuf, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return &config, nil
}
