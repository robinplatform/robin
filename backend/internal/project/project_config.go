package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type RobinProjectConfig struct {
	// Path to the project folder
	ProjectPath string `json:"-"`
	// Name of the app
	Name string `json:"name,omitempty"`
	// Apps to load for this project
	Apps []string `json:"apps,omitempty"`
}

func LoadFromEnv() (RobinProjectConfig, error) {
	projectPath, err := GetProjectPath()
	if err != nil {
		return RobinProjectConfig{}, err
	}

	var projectConfig RobinProjectConfig
	err = projectConfig.LoadRobinProjectConfig(projectPath)
	return projectConfig, err
}

func (projectConfig *RobinProjectConfig) LoadRobinProjectConfig(projectPath string) error {
	var err error

	projectConfig.ProjectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectConfig.ProjectPath, "robin.json")

	buf, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read robin.json: %s", err)
	}

	err = json.Unmarshal(buf, &projectConfig)
	if err != nil {
		return fmt.Errorf("failed to parse robin.json: %s", err)
	}

	if projectConfig.Name == "" {
		return fmt.Errorf("robin.json is missing the 'name' field")
	}

	return nil
}

func (projectConfig *RobinProjectConfig) SaveRobinProjectConfig() error {
	configPath := filepath.Join(projectConfig.ProjectPath, "robin.json")

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
