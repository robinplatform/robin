package config

import (
	"fmt"
	"os"
	"path"
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

func bailProjectPathSearch(visited []string) {
	fmt.Fprintf(os.Stderr, "Could not find robin.config.ts file\n\n")
	fmt.Fprintf(os.Stderr, "Checked:\n")
	for _, dir := range visited {
		fmt.Fprintf(os.Stderr, "\t%s\n", dir)
	}
	fmt.Fprintf(os.Stderr, "\n")

	os.Exit(1)
}

func findProjectPath(currentDir string, visited []string) string {
	if currentDir == "/" {
		bailProjectPathSearch(visited)
	}

	if fileExists(path.Join(currentDir, "robin.config.ts")) {
		return findProjectPath(path.Dir(currentDir), append(visited, currentDir))
	}
	return currentDir
}

func SetProjectPath(givenProjectPath string) string {
	givenProjectPath = path.Clean(givenProjectPath)
	if fileExists(path.Join(givenProjectPath, "robin.config.ts")) {
		projectPath = givenProjectPath
		return projectPath
	}

	bailProjectPathSearch([]string{givenProjectPath})
	// This will never be reached, but the compiler doesn't know that.
	return ""
}

func GetProjectPath() string {
	if projectPath == "" {
		// First try to load it from the env. We don't use this as a hint, but rather as an
		// exact path to the project. We just perform a quick check to make sure it is a valid
		// robin project.
		envProjectPath := os.Getenv("PROJECT_PATH")
		if envProjectPath != "" {
			return SetProjectPath(envProjectPath)
		}

		// Otherwise perform a recursive check from the cwd to find the closest robin project.
		cwd, err := os.Getwd()
		if err != nil {
			panic(fmt.Errorf("failed to get cwd: %s", err))
		}
		projectPath = findProjectPath(cwd, nil)
	}
	return projectPath
}
