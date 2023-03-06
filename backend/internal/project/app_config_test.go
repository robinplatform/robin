package project

import "testing"

func TestAppConfigLoad(t *testing.T) {
	projectPath := t.TempDir()
	err := createProjectStructure(projectPath, map[string]string{
		"robin.json": `{
			"name": "robin",
			"apps": ["./app1/robin.app.json", "file:app2/robin.app.json"]
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

	var expectedApps = []RobinAppConfig{
		{
			Id:       "bad-ext",
			Name:     "Failing Extension",
			Page:     "page.tsx",
			PageIcon: "🙅‍♂️",
		},
		{
			Id:       "bad-js-ext",
			Name:     "Invalid JS",
			Page:     "page.tsx",
			PageIcon: "💥",
		},
	}

	for index, app := range apps {
		expectedApp := expectedApps[index]

		if app != expectedApp {
			t.Fatalf("expected %#v, got %#v", expectedApp, app)
		}
	}
}
