package project

import (
	"os"
	"path/filepath"
	"testing"
)

func createProjectStructure(projectPath string, paths map[string]string) error {
	projectPath = filepath.FromSlash(projectPath)
	for path, contents := range paths {
		path = filepath.FromSlash(filepath.Join(projectPath, path))

		if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
			return err
		}

		if err := os.WriteFile(path, []byte(contents), 0755); err != nil {
			return err
		}
	}

	return nil
}

func TestConfigLoad(t *testing.T) {
	projectPath := t.TempDir()
	err := createProjectStructure(projectPath, map[string]string{
		"robin.json": `{
			"name": "robin",
			"apps": ["app1", "app2"]
		}`,
	})

	if err != nil {
		t.Fatal(err)
	}

	var projectConfig RobinProjectConfig
	if err := projectConfig.LoadRobinProjectConfig(projectPath); err != nil {
		t.Fatal(err)
	}

	if projectConfig.Name != "robin" {
		t.Errorf("Expected 'robin', got '%s'", projectConfig.Name)
	}

	expected := []string{"app1", "app2"}
	if len(projectConfig.Apps) != len(expected) {
		t.Errorf("Expected %d apps, got %d", len(expected), len(projectConfig.Apps))
	}

	for idx, appPath := range projectConfig.Apps {
		if appPath != expected[idx] {
			t.Errorf("Expected app %d to be '%s', got '%s'", idx, expected[idx], appPath)
		}
	}
}
