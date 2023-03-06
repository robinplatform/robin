package project

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"robinplatform.dev/internal/httpcache"
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
		return appConfig.ConfigPath.ResolveReference(&url.URL{Path: filepath.ToSlash(filePath)})
	}

	targetPath := filepath.Join(filepath.Dir(appConfig.ConfigPath.Path), filePath)
	return appConfig.ConfigPath.ResolveReference(&url.URL{Path: filepath.ToSlash(targetPath)})
}

func (appConfig *RobinAppConfig) ReadFile(httpClient *httpcache.CacheClient, targetPath string) (*url.URL, []byte, error) {
	fileUrl := appConfig.resolvePath(targetPath)

	if fileUrl.Scheme == "file" {
		buf, err := os.ReadFile(fileUrl.Path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file '%s': %w", targetPath, err)
		}
		return fileUrl, buf, nil
	}

	res, err := httpClient.Get(fileUrl.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file '%s': %w", targetPath, err)
	}

	lastUrl, _ := url.Parse(res.RequestUrl)
	return lastUrl, []byte(res.Body), nil
}

func (appConfig *RobinAppConfig) readRobinAppConfig(projectConfig *RobinProjectConfig, configPath string) error {
	// TODO: this sorta works, but there's some messiness that we probably need to sort out with
	// Windows paths, since the configPath gets checked into version control
	configPath = filepath.ToSlash(configPath)

	var err error
	appConfig.ConfigPath, err = url.Parse(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config path '%s': %s", configPath, err)
	}

	projectPath := filepath.ToSlash(projectConfig.ProjectPath)

	// File paths should be absolute paths
	if appConfig.ConfigPath.Scheme == "" {
		appConfig.ConfigPath.Scheme = "file"
	}
	if appConfig.ConfigPath.Scheme == "file" && !path.IsAbs(appConfig.ConfigPath.Path) {
		appConfig.ConfigPath.Path = path.Clean(path.Join(projectPath, appConfig.ConfigPath.Path))
	}

	if appConfig.ConfigPath.Scheme != "file" && appConfig.ConfigPath.Scheme != "https" {
		return fmt.Errorf("invalid config path scheme '%s' (only file and https are supported)", appConfig.ConfigPath.Scheme)
	}

	// All paths must end with `robin.app.json`
	if path.Base(appConfig.ConfigPath.Path) != "robin.app.json" {
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
	projectConfig, err := LoadProjectConfig()
	if err != nil {
		return nil, err
	}
	return projectConfig.AppSettings[appConfig.Id], nil
}

func (appConfig *RobinAppConfig) UpdateSettings(settings map[string]any) error {
	projectConfig, err := LoadProjectConfig()
	if err != nil {
		return err
	}

	projectConfig.AppSettings[appConfig.Id] = settings
	return UpdateProjectConfig(projectConfig)
}

func (projectConfig *RobinProjectConfig) GetAllProjectApps() ([]RobinAppConfig, error) {
	apps := make([]RobinAppConfig, len(projectConfig.Apps))
	for idx, configPath := range projectConfig.Apps {
		if err := apps[idx].readRobinAppConfig(projectConfig, configPath); err != nil {
			return nil, err
		}
	}

	return apps, nil
}

func GetAllProjectApps() ([]RobinAppConfig, error) {
	var projectConfig RobinProjectConfig
	if err := projectConfig.LoadFromEnv(); err != nil {
		return nil, err
	}

	return projectConfig.GetAllProjectApps()
}

func (projectConfig *RobinProjectConfig) LoadRobinAppByPath(appPath string) (RobinAppConfig, error) {
	var appConfig RobinAppConfig
	if err := appConfig.readRobinAppConfig(projectConfig, appPath); err != nil {
		return RobinAppConfig{}, err
	}
	return appConfig, nil
}

func LoadRobinAppById(appId string) (RobinAppConfig, error) {
	var projectConfig RobinProjectConfig
	if err := projectConfig.LoadFromEnv(); err != nil {
		return RobinAppConfig{}, err
	}

	return projectConfig.LoadRobinAppById(appId)
}

func (projectConfig *RobinProjectConfig) LoadRobinAppById(appId string) (RobinAppConfig, error) {
	apps, err := projectConfig.GetAllProjectApps()
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
