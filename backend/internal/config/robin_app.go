package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
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
	if path.IsAbs(filePath) {
		return appConfig.ConfigPath.ResolveReference(&url.URL{Path: filePath})
	}
	return appConfig.ConfigPath.ResolveReference(&url.URL{Path: path.Join(path.Dir(appConfig.ConfigPath.Path), filePath)})
}

func (appConfig *RobinAppConfig) ReadFile(filePath string) (string, error) {
	var buf []byte
	var err error
	fileUrl := appConfig.resolvePath(filePath)

	if fileUrl.Scheme == "file" {
		buf, err = os.ReadFile(fileUrl.Path)
		if err != nil {
			return "", fmt.Errorf("failed to read file '%s': %s", filePath, err)
		}
	} else {
		switch path.Ext(fileUrl.Path) {
		case "", ".js", ".jsx", ".ts", ".tsx":
			fileUrl.RawQuery = "module"
		}

		req := &http.Request{
			Method: "GET",
			URL:    fileUrl,
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to read file '%s': %s", filePath, err)
		}

		buf, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read file '%s': %s", filePath, err)
		}
	}

	return string(buf), nil
}

type robinProjectConfig struct {
	// Name of the app
	Name string `json:"name"`
	// Apps to load for this project
	Apps []string `json:"apps"`
}

type RobinProjectConfig struct {
	// Name of the app
	Name string
	// Apps to load for this project
	Apps []RobinAppConfig
}

func readRobinAppConfig(configPath string, appConfig *RobinAppConfig) error {
	var buf []byte
	var err error

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
	if appConfig.ConfigPath.Scheme == "https" && appConfig.ConfigPath.Host != "unpkg.com" {
		return fmt.Errorf("cannot load file from host '%s' (only unpkg.com is supported)", appConfig.ConfigPath.Host)
	}

	// All paths must end with `robin.app.json`
	if path.Base(appConfig.ConfigPath.Path) != "robin.app.json" {
		appConfig.ConfigPath = appConfig.ConfigPath.JoinPath("robin.app.json")
	}

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

func LoadRobinProjectConfig() (RobinProjectConfig, error) {
	projectPath, err := GetProjectPath()
	if err != nil {
		return RobinProjectConfig{}, err
	}

	storedConfig := robinProjectConfig{}
	parsedConfig := RobinProjectConfig{}
	configPath := path.Join(projectPath, "robin.json")

	buf, err := os.ReadFile(configPath)
	if err != nil {
		return parsedConfig, fmt.Errorf("failed to read robin.app.json: %s", err)
	}

	err = json.Unmarshal(buf, &storedConfig)
	if err != nil {
		return parsedConfig, fmt.Errorf("failed to parse robin.app.json: %s", err)
	}

	parsedConfig.Name = storedConfig.Name
	parsedConfig.Apps = make([]RobinAppConfig, len(storedConfig.Apps))
	for i, appConfigPath := range storedConfig.Apps {
		err := readRobinAppConfig(appConfigPath, &parsedConfig.Apps[i])
		if err != nil {
			return parsedConfig, fmt.Errorf("failed to read robin app config in '%s': %s", appConfigPath, err)
		}
	}

	return parsedConfig, nil
}

func LoadRobinAppById(appId string) (RobinAppConfig, error) {
	projectConfig, err := LoadRobinProjectConfig()
	if err != nil {
		return RobinAppConfig{}, err
	}

	for _, appConfig := range projectConfig.Apps {
		if appConfig.Id == appId {
			return appConfig, nil
		}
	}

	return RobinAppConfig{}, fmt.Errorf("failed to find app with id '%s'", appId)
}
