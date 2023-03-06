package project

import (
	"net/url"
	"path/filepath"
	"testing"
)

func TestAppConfigLoad(t *testing.T) {
	projectPath := t.TempDir()
	err := createProjectStructure(projectPath, map[string]string{
		// TODO: Weird things happen when passing in `file:app2/robin.app.json`
		// maybe it's best to fix it later though.
		"robin.json": `{
			"name": "robin",
			"apps": ["./app1/robin.app.json", "app2/robin.app.json"]
		}`,

		"app1/robin.app.json": `{
			"id": "bad-ext",
			"name": "Failing Extension",
			"page": "page.tsx",
			"pageIcon": "🙅‍♂️"
		}`,

		"app2/robin.app.json": `{
			"id": "bad-js-ext",
			"name": "Invalid JS",
			"page": "page.tsx",
			"pageIcon": "💥"
		}`,
	})

	if err != nil {
		t.Fatal(err)
	}

	var projectConfig RobinProjectConfig
	if err := projectConfig.LoadRobinProjectConfig(projectPath); err != nil {
		t.Fatal(err)
	}

	apps, err := projectConfig.GetAllProjectApps()
	if err != nil {
		t.Fatal(err)
	}

	app1Url, err := url.Parse("file://" + filepath.Join(projectPath, "app1/robin.app.json"))
	if err != nil {
		t.Fatal(err)
	}

	app2Url, err := url.Parse("file://" + filepath.Join(projectPath, "app2/robin.app.json"))
	if err != nil {
		t.Fatal(err)
	}

	var expectedApps = []RobinAppConfig{
		{
			ConfigPath: app1Url,
			Id:         "bad-ext",
			Name:       "Failing Extension",
			Page:       "page.tsx",
			PageIcon:   "🙅‍♂️",
		},
		{
			ConfigPath: app2Url,
			Id:         "bad-js-ext",
			Name:       "Invalid JS",
			Page:       "page.tsx",
			PageIcon:   "💥",
		},
	}

	if len(apps) != len(expectedApps) {
		t.Errorf("Expected %d apps, got %d", len(expectedApps), len(apps))
	}

	for index, app := range apps {
		expectedApp := expectedApps[index]

		if app.ConfigPath.String() != expectedApp.ConfigPath.String() {
			t.Errorf("Expected app path %d to be '%s', got '%s'", index, expectedApp.ConfigPath.String(), app.ConfigPath.String())
		}

		expectedApp.ConfigPath = app.ConfigPath

		if app != expectedApp {
			t.Fatalf("expected %#v, got %#v", expectedApp, app)
		}
	}
}
