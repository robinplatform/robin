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

	// If it doesn't exist, create it
	if _, err := os.Stat(robinPath); os.IsNotExist(err) {
		if err := os.MkdirAll(robinPath, 0777); err != nil {
			panic(fmt.Errorf("failed to create robin directory: %w", err))
		}
	}
}

func GetRobinPath() string {
	return robinPath
}
