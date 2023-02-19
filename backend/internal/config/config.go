package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
)

var (
	pathRegex *regexp.Regexp = regexp.MustCompile("[^a-zA-Z0-9]+")

	robinPath string

	// This should only be loaded once, since robin will start up with a target project
	projectName string
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to get user home directory: %w", err))
	}

	robinPath = path.Join(home, ".robin")

	// If it doesn't exist, create it
	if _, err := os.Stat(robinPath); os.IsNotExist(err) {
		if err := os.MkdirAll(robinPath, 0777); err != nil {
			panic(fmt.Errorf("failed to create robin directory: %w", err))
		}
	}

	for _, rc := range []ReleaseChannel{ReleaseChannelStable, ReleaseChannelBeta, ReleaseChannelNightly} {
		rcPath := path.Join(robinPath, string(rc))
		if err := os.MkdirAll(rcPath, 0777); err != nil {
			panic(fmt.Errorf("failed to create robin directory: %w", err))
		}
	}
}

func GetRobinPath() string {
	return robinPath
}

func GetProjectName() (string, error) {
	if projectName != "" {
		return projectName, nil
	}

	projectPath, err := GetProjectPath()
	if err != nil {
		return "", err
	}

	packageJsonPath := path.Join(projectPath, "package.json")
	var packageJson PackageJson
	if err := LoadPackageJson(packageJsonPath, &packageJson); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load %s: %v", packageJsonPath, err)
		os.Exit(1)
	}
	projectName = packageJson.Name

	return projectName, nil
}

func GetProjectAlias() (string, error) {
	if projectName == "" {
		_, err := GetProjectName()
		if err != nil {
			return "", err
		}
	}

	// Remove all non alphanumeric characters from 'projectName' so it is a safe directory name
	return pathRegex.ReplaceAllString(projectName, ""), nil
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

var defaultRobinConfig = RobinConfig{
	ReleaseChannel:         ReleaseChannelStable,
	ShowReactQueryDebugger: false,
	MinifyExtensionClients: true,
	EnableKeyMappings:      true,
}

func LoadProjectConfig() (RobinConfig, error) {
	alias, err := GetProjectAlias()
	if err != nil {
		return defaultRobinConfig, err
	}

	robinConfigPath := path.Join(robinPath, "projects", alias, "config.json")

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

	robinConfigPath := path.Join(robinPath, "projects", alias, "config.json")

	// Marshal the config file
	buf, err := json.Marshal(&projectConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config file: %w", err)
	}

	if err := os.MkdirAll(path.Join(robinPath, "projects", alias), 0777); err != nil {
		return fmt.Errorf("failed to create folder for config file: %w", err)
	}

	// Save the file
	if err := os.WriteFile(robinConfigPath, buf, 0755); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
