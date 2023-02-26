package process

import (
	"path"
	"testing"
	"time"
)

func TestSpawnProcess(t *testing.T) {
	dir := t.TempDir()
	dbFile := path.Join(dir, "testing.db")

	manager, err := NewProcessManager(dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := ProcessId{
		Namespace:    NamespaceInternal,
		NamespaceKey: "default",
		Key:          "long",
	}

	err = manager.SpawnPath(ProcessConfig{
		Id:      id,
		Command: "sleep",
		Args:    []string{"100"},
	})
	if err != nil {
		t.Fatalf("error spawning process: %s", err.Error())
	}

	err = manager.Kill(id)
	if err != nil {
		t.Fatalf("failed to kill process %+v: %s", id, err.Error())
	}

	if manager.IsAlive(id) {
		t.Fatalf("manager thinks the process is still alive")
	}
}

func TestSpawnDead(t *testing.T) {
	dir := t.TempDir()
	dbFile := path.Join(dir, "testing.db")

	manager, err := NewProcessManager(dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := ProcessId{
		Namespace:    NamespaceInternal,
		NamespaceKey: "default",
		Key:          "short",
	}

	err = manager.SpawnPath(ProcessConfig{
		Id:      id,
		Command: "sleep",
		Args:    []string{"0"},
	})
	if err != nil {
		t.Fatalf("error spawning process: %s", err.Error())
	}

	time.Sleep(time.Millisecond * 100)

	if manager.IsAlive(id) {
		t.Fatalf("manager thinks the process is still alive")
	}
}

// TODO: test to ensure that writes to the stderr and stdout don't mess with each other
// TODO: test to ensure that children that the process spawns get killed as well
