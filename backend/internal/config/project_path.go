package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"robinplatform.dev/internal/static"
)

var (
	projectPath string
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type ProjectPathNotFoundError struct {
	visited []string
}

func (e ProjectPathNotFoundError) Error() string {
	var sb strings.Builder

	sb.WriteString("Could not find a robin.json file\n\n")
	sb.WriteString("Checked:\n")
	for _, dir := range e.visited {
		sb.WriteString(fmt.Sprintf("\t%s\n", dir))
	}
	sb.WriteString("\n")

	return sb.String()
}

func findProjectPath(currentDir string, visited []string) (string, error) {
	if currentDir == "/" {
		return "", ProjectPathNotFoundError{visited: visited}
	}

	if !fileExists(filepath.Join(currentDir, "robin.json")) {
		return findProjectPath(filepath.Dir(currentDir), append(visited, currentDir))
	}
	return currentDir, nil
}

func SetProjectPath(givenProjectPath string) (string, error) {
	if !filepath.IsAbs(givenProjectPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error: failed to get cwd: %s", err)
		}

		givenProjectPath = filepath.Join(cwd, givenProjectPath)
	}

	givenProjectPath = filepath.Clean(givenProjectPath)
	if fileExists(filepath.Join(givenProjectPath, "robin.json")) {
		projectPath = givenProjectPath
		return projectPath, nil
	}

	return "", ProjectPathNotFoundError{visited: []string{givenProjectPath}}
}

var GetProjectPath = static.CreateOnce(func() (string, error) {
	if projectPath == "" {
		// First try to load it from the env. We don't use this as a hint, but rather as an
		// exact path to the project. We just perform a quick check to make sure it is a valid
		// robin project.
		envProjectPath := os.Getenv("ROBIN_PROJECT_PATH")
		if envProjectPath != "" {
			return SetProjectPath(envProjectPath)
		}

		// Otherwise perform a recursive check from the cwd to find the closest robin project.
		cwd, err := os.Getwd()
		if err != nil {
			panic(fmt.Errorf("failed to get cwd: %s", err))
		}

		discoveredProjectPath, err := findProjectPath(cwd, nil)
		if err != nil {
			return "", err
		}
		return SetProjectPath(discoveredProjectPath)
	}
	return projectPath, nil
})

func GetProjectPathOrExit() string {
	projectPath, err := GetProjectPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	return projectPath
}
