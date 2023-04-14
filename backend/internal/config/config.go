package config

import (
	"fmt"
	"os"
	"path/filepath"
)

var robinPath string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("failed to get user home directory: %w", err))
	}

	robinPath = filepath.Join(home, ".robin")

	if err := os.MkdirAll(robinPath, 0777); err != nil {
		panic(fmt.Errorf("failed to create robin directory: %w", err))
	}
}

func GetRobinPath() string {
	return robinPath
}

func GetHttpCachePath() string {
	return filepath.Join(robinPath, "data", "http-cache.json")
}
