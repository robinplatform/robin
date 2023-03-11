package process

import (
	"path/filepath"

	"robinplatform.dev/internal/config"
)

type DummyMeta struct {
}

var Manager *ProcessManager[DummyMeta]

func init() {
	robinPath := config.GetRobinPath()
	manager, err := NewProcessManager[DummyMeta](filepath.Join(
		robinPath,
		"data",
		"spawned-processes.db",
	))

	if err != nil {
		panic(err)
	}

	Manager = manager
}
