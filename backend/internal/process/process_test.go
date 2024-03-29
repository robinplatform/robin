package process

import (
	"path/filepath"
	"testing"

	"robinplatform.dev/internal/process/health"
	"robinplatform.dev/internal/pubsub"
)

func TestSpawnProcess(t *testing.T) {
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "testing.db")

	topics := &pubsub.Registry{}
	manager, err := NewProcessManager(topics, dir, dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := ProcessId{Category: "robin", Key: "long"}

	proc, err := manager.SpawnFromPathVar(ProcessConfig{
		Id:      id,
		Command: "sleep",
		Args:    []string{"100"},
	})
	if err != nil {
		t.Fatalf("error spawning process: %s", err.Error())
	}

	if !manager.IsAlive(id) {
		t.Fatalf("manager doesn't think process is alive, even though it just spawned it")
	}

	if !health.PidIsAlive(proc.Pid) {
		t.Fatalf("manager/OS doesn't think process is alive, even though it just spawned it")
	}

	err = manager.Kill(id)
	if err != nil {
		t.Fatalf("failed to kill process %+v: %s", id, err.Error())
	}

	<-proc.Context.Done()

	if manager.IsAlive(id) {
		t.Fatalf("manager thinks the process is still alive")
	}
	if health.PidIsAlive(proc.Pid) {
		t.Fatalf("manager/OS thinks the process is still alive")
	}
}

func TestSpawnDead(t *testing.T) {
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "testing.db")

	topics := &pubsub.Registry{}
	manager, err := NewProcessManager(topics, dir, dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := ProcessId{Category: "robin", Key: "short"}

	proc, err := manager.SpawnFromPathVar(ProcessConfig{
		Id:      id,
		Command: "sleep",
		Args:    []string{"1"},
	})
	if err != nil {
		t.Fatalf("error spawning process: %s", err.Error())
	}

	if !health.PidIsAlive(proc.Pid) {
		t.Fatalf("manager/OS thinks the process is dead before it dies")
	}

	// Wait for the process to die
	<-proc.Context.Done()

	if manager.IsAlive(id) {
		t.Fatalf("manager thinks the process is still alive")
	}
	if health.PidIsAlive(proc.Pid) {
		t.Fatalf("manager/OS thinks the process is still alive")
	}
}

func TestSpawnedBeforeManagerStarted(t *testing.T) {
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "testing.db")

	topicsA := &pubsub.Registry{}
	managerA, err := NewProcessManager(topicsA, dir, dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	id := ProcessId{Category: "robin", Key: "previous"}
	procA, err := managerA.SpawnFromPathVar(ProcessConfig{
		Id:      id,
		Command: "sleep",
		Args:    []string{"1"},
	})
	_ = procA

	if err != nil {
		t.Fatalf("error spawning process: %s", err.Error())
	}

	// This is a weird way to test this, but I think it sorta makes sense if you
	// don't think about it too hard.
	// The idea is, we create two managers, and the first spawns the process,
	// and then we don't touch it anymore. Then, the second is created, as if Robin
	// restarted and the manager is going in fresh with processes that haven't died yet.
	topicsB := &pubsub.Registry{}
	managerB, err := NewProcessManager(topicsB, dir, dbFile)
	if err != nil {
		t.Fatalf("error loading DB: %s", err.Error())
	}

	procB, found := managerB.FindById(id)
	if !found {
		t.Fatalf("error finding processs")
	}

	if !managerB.IsAlive(id) {
		t.Fatalf("manager doesn't think process is alive, even though it just spawned it")
	}

	if !health.PidIsAlive(procB.Pid) {
		t.Fatalf("manager/OS doesn't think process is alive, even though it just spawned it")
	}

	<-procB.Context.Done()

	if managerB.IsAlive(id) {
		t.Fatalf("manager thinks process is alive after it died")
	}

	if health.PidIsAlive(procB.Pid) {
		t.Fatalf("manager/OS thinks process is alive after it died")
	}
}

// TODO: test to ensure that writes to the stderr and stdout don't mess with each other
// TODO: test to ensure that children that the process spawns get killed as well
