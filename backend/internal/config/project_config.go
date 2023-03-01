package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type RobinProjectConfig struct {
	// Name of the app
	Name string `json:"name,omitempty"`
	// Apps to load for this project
	Apps []string `json:"apps,omitempty"`
}

func (projectConfig *RobinProjectConfig) LoadRobinProjectConfig() error {
	projectPath, err := GetProjectPath()
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectPath, "robin.json")

	buf, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read robin.json: %s", err)
	}

	err = json.Unmarshal(buf, &projectConfig)
	if err != nil {
		return fmt.Errorf("failed to parse robin.json: %s", err)
	}

	return nil
}

func (projectConfig *RobinProjectConfig) SaveRobinProjectConfig() error {
	projectPath, err := GetProjectPath()
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectPath, "robin.json")

	// Let's indent the config so it is easily readable
	buf, err := json.MarshalIndent(projectConfig, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal robin.json: %s", err)
	}

	err = os.WriteFile(configPath, buf, 0777)
	if err != nil {
		return fmt.Errorf("failed to write robin.json: %s", err)
	}

	return nil
}
