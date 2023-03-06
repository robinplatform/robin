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
	"runtime"

	"robinplatform.dev/internal/httpcache"
)

type serializableDaemonEntrypoint []string

func (entrypoint *serializableDaemonEntrypoint) UnmarshalJSON(data []byte) error {
	// first try to unmarshal as a []string
	var entrypointArray []string
	if err := json.Unmarshal(data, &entrypointArray); err == nil {
		*entrypoint = entrypointArray
		return nil
	}

	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// then try to unmarshal as a map[string]string
	var entrypointMap map[string][]string
	if err := json.Unmarshal(data, &entrypointMap); err == nil {
		// verify that the map has an entry for the current platform
		if entrypointMap[platform] == nil {
			return fmt.Errorf("daemon map does not contain an entry for the current platform '%s'", platform)
		}

		*entrypoint = entrypointMap[platform]
		return nil
	}

	return fmt.Errorf("failed to unmarshal daemon entrypoint (expected either a string array or a map)")
}

type serializableRobinAppConfig struct {
	Id       string                       `json:"id"`
	Name     string                       `json:"name"`
	PageIcon string                       `json:"pageIcon"`
	Page     string                       `json:"page"`
	Files    []string                     `json:"files"`
	Daemon   serializableDaemonEntrypoint `json:"daemon"`
}

type RobinAppConfig struct {
	// ConfigPath is a URL pointing to the location of the config file
	ConfigPath *url.URL `json:"-"`

	// Id of the app
	Id string
	// Name of the app
	Name string
	// PageIcon refers to the path to the icon to use for this app
	PageIcon string
	// Page refers to the path to the page to load for this app
	Page string
	// Files refers to the paths to the files that are necessary for this app
	// These files will be copied to the app's working directory
	Files []string
	// Daemon represents the command that should be run to start the app's daemon
	Daemon []string
}

func (appConfig RobinAppConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(&serializableRobinAppConfig{
		Id:       appConfig.Id,
		Name:     appConfig.Name,
		PageIcon: appConfig.PageIcon,
		Page:     appConfig.Page,
		Files:    appConfig.Files,
		Daemon:   appConfig.Daemon,
	})
}

func (appConfig *RobinAppConfig) UnmarshalJSON(data []byte) error {
	var serializableConfig serializableRobinAppConfig
	if err := json.Unmarshal(data, &serializableConfig); err != nil {
		return err
	}

	appConfig.Id = serializableConfig.Id
	appConfig.Name = serializableConfig.Name
	appConfig.PageIcon = serializableConfig.PageIcon
	appConfig.Page = serializableConfig.Page
	appConfig.Files = serializableConfig.Files
	appConfig.Daemon = serializableConfig.Daemon

	return nil
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

func (appConfig *RobinAppConfig) readRobinAppConfig(configPath string) error {
	projectPath, err := GetProjectPath()
	if err != nil {
		return fmt.Errorf("failed to get project path: %s", err)
	}

	configPath = filepath.ToSlash(configPath)
	appConfig.ConfigPath, err = url.Parse(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config path '%s': %s", configPath, err)
	}

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

func GetAllProjectApps() ([]RobinAppConfig, error) {
	projectPath, err := GetProjectPath()
	if err != nil {
		return nil, err
	}

	projectConfig := RobinProjectConfig{}
	if err := projectConfig.LoadRobinProjectConfig(projectPath); err != nil {
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
