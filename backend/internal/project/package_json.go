package project

import (
	"encoding/json"
	"fmt"
	"os"
)

type PackageJson struct {
	Name            string
	Version         string
	Scripts         map[string]string
	Dependencies    map[string]string
	DevDependencies map[string]string
	Robin           string
}

// ParsePackageJson parses the given package.json file from the buffer
func ParsePackageJson(buf []byte, packageJson *PackageJson) error {
	if err := json.Unmarshal(buf, packageJson); err != nil {
		return fmt.Errorf("failed to unmarshal package.json file: %w", err)
	}
	return nil
}

// LoadPackageJson loads the package.json file from the file that exists at `filename`
func LoadPackageJson(filename string, packageJson *PackageJson) error {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read package.json file: %w", err)
	}

	return ParsePackageJson(buf, packageJson)
}
