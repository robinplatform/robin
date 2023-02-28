package compile

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"robinplatform.dev/internal/config"
)

type RobinAppConfig struct {
	// ConfigPath is a URL pointing to the location of the config file
	ConfigPath *url.URL `json:"-"`

	// Id of the app
	Id string `json:"id"`
	// Name of the app
	Name string `json:"name"`
	// PageIcon refers to the path to the icon to use for this app
	PageIcon string `json:"pageIcon"`
	// Page refers to the path to the page to load for this app
	Page string `json:"page"`
}

func (appConfig *RobinAppConfig) resolvePath(filePath string) *url.URL {
	parsedUrl, err := url.Parse(filePath)
	if err == nil && parsedUrl.Scheme != "" {
		return parsedUrl
	}

	if filepath.IsAbs(filePath) {
		return appConfig.ConfigPath.ResolveReference(&url.URL{Path: filePath})
	}
	return appConfig.ConfigPath.ResolveReference(&url.URL{Path: filepath.Join(filepath.Dir(appConfig.ConfigPath.Path), filePath)})
}

func (appConfig *RobinAppConfig) ReadFile(filePath string) (*url.URL, []byte, error) {
	var buf []byte
	var err error
	fileUrl := appConfig.resolvePath(filePath)

	if fileUrl.Scheme == "file" {
		buf, err = os.ReadFile(fileUrl.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file '%s': %s", filePath, err)
		}
		return fileUrl, buf, nil
	}

	fileUrl.RawQuery = "bundle"
	req := &http.Request{
		Method: "GET",
		URL:    fileUrl,
	}
	lastReq := req
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			lastReq = req
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file '%s': %s", filePath, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to read file '%s': %s", filePath, resp.Status)
	}

	buf, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file '%s': %s", filePath, err)
	}

	return lastReq.URL, buf, nil
}

func (appConfig *RobinAppConfig) readRobinAppConfig(configPath string) error {
	projectPath, err := config.GetProjectPath()
	if err != nil {
		return fmt.Errorf("failed to get project path: %s", err)
	}

	appConfig.ConfigPath, err = url.Parse(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config path '%s': %s", configPath, err)
	}

	// File paths should be absolute paths
	if appConfig.ConfigPath.Scheme == "" {
		appConfig.ConfigPath.Scheme = "file"
	}
	if appConfig.ConfigPath.Scheme == "file" && !filepath.IsAbs(appConfig.ConfigPath.Path) {
		appConfig.ConfigPath.Path = filepath.Clean(filepath.Join(projectPath, appConfig.ConfigPath.Path))
	}

	if appConfig.ConfigPath.Scheme != "file" && appConfig.ConfigPath.Scheme != "https" {
		return fmt.Errorf("invalid config path scheme '%s' (only file and https are supported)", appConfig.ConfigPath.Scheme)
	}
	if appConfig.ConfigPath.Scheme == "https" && appConfig.ConfigPath.Host != "esm.sh" {
		return fmt.Errorf("cannot load file from host '%s' (only esm.sh is supported)", appConfig.ConfigPath.Host)
	}

	// All paths must end with `robin.app.json`
	if filepath.Base(appConfig.ConfigPath.Path) != "robin.app.json" {
		appConfig.ConfigPath = appConfig.ConfigPath.JoinPath("robin.app.json")
	}

	var buf []byte

	if appConfig.ConfigPath.Scheme == "file" {
		buf, err = os.ReadFile(appConfig.ConfigPath.Path)
		if err != nil {
			return fmt.Errorf("failed to read robin.app.json: %s", err)
		}
	} else if appConfig.ConfigPath.Scheme == "https" {
		resp, err := http.DefaultClient.Do(&http.Request{
			URL: appConfig.ConfigPath,
		})
		if err != nil {
			return fmt.Errorf("failed to read robin.app.json: %s", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to read robin.app.json: http error %s", resp.Status)
		}

		buf, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read robin.app.json: %s", err)
		}
	} else {
		return fmt.Errorf("unsupported config path scheme '%s'", appConfig.ConfigPath.Scheme)
	}

	err = json.Unmarshal(buf, &appConfig)
	if err != nil {
		return fmt.Errorf("failed to parse robin.app.json: %s", err)
	}

	if appConfig.Id == "" {
		return fmt.Errorf("'id' is required")
	}

	// We don't need to verify that page points to a real page yet, we can
	// do that at app compile time
	if appConfig.Page == "" {
		return fmt.Errorf("'page' is required")
	}

	if appConfig.PageIcon == "" {
		return fmt.Errorf("'pageIcon' is required")
	}

	if appConfig.Name == "" {
		return fmt.Errorf("'name' is required")
	}

	return nil
}

func (appConfig *RobinAppConfig) GetSettings() (map[string]any, error) {
	projectConfig, err := config.LoadProjectConfig()
	if err != nil {
		return nil, err
	}
	return projectConfig.AppSettings[appConfig.Id], nil
}

func (appConfig *RobinAppConfig) UpdateSettings(settings map[string]any) error {
	projectConfig, err := config.LoadProjectConfig()
	if err != nil {
		return err
	}

	projectConfig.AppSettings[appConfig.Id] = settings
	return config.UpdateProjectConfig(projectConfig)
}

func GetAllProjectApps() ([]RobinAppConfig, error) {
	projectConfig := config.RobinProjectConfig{}
	if err := projectConfig.LoadRobinProjectConfig(); err != nil {
		return nil, err
	}

	apps := make([]RobinAppConfig, len(projectConfig.Apps))
	for idx, configPath := range projectConfig.Apps {
		if err := apps[idx].readRobinAppConfig(configPath); err != nil {
			return nil, err
		}
	}

	return apps, nil
}

func LoadRobinAppByPath(appPath string) (RobinAppConfig, error) {
	var appConfig RobinAppConfig
	if err := appConfig.readRobinAppConfig(appPath); err != nil {
		return RobinAppConfig{}, err
	}
	return appConfig, nil
}

func LoadRobinAppById(appId string) (RobinAppConfig, error) {
	apps, err := GetAllProjectApps()
	if err != nil {
		return RobinAppConfig{}, err
	}

	for _, app := range apps {
		if app.Id == appId {
			return app, nil
		}
	}

	return RobinAppConfig{}, fmt.Errorf("failed to find app with id '%s'", appId)
}
