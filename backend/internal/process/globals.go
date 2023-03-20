package process

import (
	"path/filepath"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/pubsub"
)

var Manager *ProcessManager

func init() {
	robinPath := config.GetRobinPath()
	manager, err := NewProcessManager(&pubsub.Topics,
		filepath.Join(robinPath, "logs", "processes"),
		filepath.Join(
			robinPath,
			"data",
			"spawned-processes.db",
		),
	)

	if err != nil {
		panic(err)
	}

	Manager = manager
}
