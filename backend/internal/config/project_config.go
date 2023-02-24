package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

type RobinProjectConfig struct {
	// Name of the app
	Name string `json:"name"`
	// Apps to load for this project
	Apps []string `json:"apps"`
}

func (projectConfig *RobinProjectConfig) LoadRobinProjectConfig() error {
	projectPath, err := GetProjectPath()
	if err != nil {
		return err
	}

	configPath := path.Join(projectPath, "robin.json")

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
