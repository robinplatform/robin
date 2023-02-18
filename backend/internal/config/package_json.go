package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type PackageJson struct {
	Name            string
	Version         string
	Scripts         map[string]string
	Dependencies    map[string]string
	DevDependencies map[string]string
}

// LoadPackageJson loads the package.json file from the file that exists at `filename`
func LoadPackageJson(filename string) (*PackageJson, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json file: %w", err)
	}

	var packageJson PackageJson
	if err := json.Unmarshal(buf, &packageJson); err != nil {
		return nil, fmt.Errorf("failed to unmarshal package.json file: %w", err)
	}

	return &packageJson, nil
}
