package process

import (
	"path/filepath"

	"robinplatform.dev/internal/config"
)

type DummyMeta struct {
}

var Manager *ProcessManager

func init() {
	robinPath := config.GetRobinPath()
	manager, err := NewProcessManager(filepath.Join(
		robinPath,
		"data",
		"spawned-processes.db",
	))

	if err != nil {
		panic(err)
	}

	Manager = manager
}
