package main

import (
	"fmt"

	"robinplatform.dev/internal/config"
)

func main() {
	fmt.Printf("projectPath: %s\n", config.GetProjectPathOrExit())

	robinConfig, err := config.LoadProjectConfig()
	if err != nil {
		panic(err)
	}
	fmt.Printf("config: %#v\n", robinConfig)
}
