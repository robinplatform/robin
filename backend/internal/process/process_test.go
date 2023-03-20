package process

import (
	"path/filepath"
	"testing"
)

func TestSpawnProcess(t *testing.T) {
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "testing.db")

	manager, err := NewProcessManager(dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := InternalId("long")

	_, err = manager.SpawnFromPathVar(ProcessConfig{
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
	dbFile := filepath.Join(dir, "testing.db")

	manager, err := NewProcessManager(dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := InternalId("short")

	proc, err := manager.SpawnFromPathVar(ProcessConfig{
		Id:      id,
		Command: "sleep",
		Args:    []string{"0"},
	})
	if err != nil {
		t.Fatalf("error spawning process: %s", err.Error())
	}

	// Wait for the process to die
	<-proc.Context.Done()

	if manager.IsAlive(id) {
		t.Fatalf("manager thinks the process is still alive")
	}
}

// TODO: test to ensure that writes to the stderr and stdout don't mess with each other
// TODO: test to ensure that children that the process spawns get killed as well
