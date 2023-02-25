package process

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/model"
)

type ProcessNamespace string

const (
	NamespaceExtensionSpawn  ProcessNamespace = "extension-spawn"
	NamespaceExtensionDaemon ProcessNamespace = "extension-daemon"
	NamespaceExtensionLambda ProcessNamespace = "extension-lambda"
	NamespaceInternal        ProcessNamespace = "internal"
)

type ProcessId struct {
	Namespace    ProcessNamespace
	NamespaceKey string
	Key          string
}

type ProcessConfig struct {
	Id      ProcessId
	WorkDir string
	Env     map[string]string
	Command string
	Argv    []string
}

type Process struct {
	Id        ProcessId
	Pid       int
	StartedAt time.Time
	WorkDir   string
	Env       map[string]string
	Command   string
	Argv      []string
}

func findById(id ProcessId) func(row Process) bool {
	return func(row Process) bool {
		return row.Id == id
	}
}

func (cfg *ProcessConfig) fillEmptyValues() error {
	if cfg.Id.Key == "" {
		return fmt.Errorf("cannot create process without a Key")
	}

	if cfg.Id.NamespaceKey == "" {
		return fmt.Errorf("cannot create process without a namespace key")
	}

	if cfg.Id.Namespace == "" {
		cfg.Id.Namespace = NamespaceInternal
	}

	if cfg.Env == nil {
		env := os.Environ()
		cfg.Env = make(map[string]string, len(env))
		for _, envVar := range env {
			parts := strings.SplitN(envVar, "=", 2)
			cfg.Env[parts[0]] = parts[1]
		}
	}

	if cfg.WorkDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		cfg.WorkDir = dir
	}

	return nil
}

type ProcessManager struct {
	db     model.Store[Process]
	logger log.Logger
}

func (m *ProcessManager) LoadDb(path string) error {
	if m.db.FilePath != "" {
		return fmt.Errorf("already loaded DB at %s", m.db.FilePath)
	}

	m.db.FilePath = path
	m.logger = log.New("process-manager")

	return m.db.Open()
}

func (m *ProcessManager) IsAlive(id ProcessId) bool {
	r := m.db.ReadHandle()
	defer r.Close()

	procEntry, found := r.Find(findById(id))
	if !found {
		return false
	}

	process, err := os.FindProcess(procEntry.Pid)
	if err != nil {
		// On Windows, if the process doesn't exist, this will
		// error. On Unix, it won't do anything.
		return false
	}

	// TODO: this next part still isn't supported on windows,
	// didn't do it there yet
	err = process.Signal(syscall.Signal(0))
	m.logger.Debug("got error on signal", log.Ctx{
		"err": err,
	})

	return err == nil

}

func (m *ProcessManager) Kill(id ProcessId) error {
	w := m.db.WriteHandle()
	defer w.Close()

	procEntry, found := w.Find(findById(id))
	if !found {
		return fmt.Errorf("id %+v wasn't found in process database", id)
	}

	if err := syscall.Kill(-procEntry.Pid, syscall.SIGKILL); err != nil {
		return err
	}

	if err := w.Delete(findById(id)); err != nil {
		return err
	}

	return nil
}

func (m *ProcessManager) SpawnPath(config ProcessConfig) error {
	var err error
	config.Command, err = exec.LookPath(config.Command)
	if err != nil {
		return err
	}

	return m.Spawn(config)
}

func waitForExit(pid int, logger log.Logger) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		logger.Debug("Failed to find process to wait on", log.Ctx{
			"err": err,
		})
		return
	}

	exitCode, err := proc.Wait()

	logger.Debug("Waited for process to exit", log.Ctx{
		"exitCode": exitCode,
		"err":      err,
	})
}

func (m *ProcessManager) Spawn(procConfig ProcessConfig) error {
	if err := procConfig.fillEmptyValues(); err != nil {
		return err
	}

	w := m.db.WriteHandle()
	defer w.Close()

	prev, found := w.Find(findById(procConfig.Id))
	if found {
		return fmt.Errorf(
			"found previous process with id=%+v",
			prev.Id,
		)
	}

	m.logger.Info("Spawning Process", log.Ctx{
		"id": procConfig.Id,
	})

	empty, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer empty.Close()

	robinPath := config.GetRobinPath()
	procFolderPath := path.Join(robinPath, "processes")
	os.MkdirAll(procFolderPath, 0755)

	procPath := path.Join(procFolderPath, string(procConfig.Id.Namespace)+"-"+procConfig.Id.NamespaceKey+"-"+procConfig.Id.Key)

	output, err := os.OpenFile(procPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer output.Close()

	var attr os.ProcAttr
	attr.Env = make([]string, 0, len(procConfig.Env))
	attr.Dir = procConfig.WorkDir
	attr.Files = []*os.File{empty, output, output}
	attr.Sys = &syscall.SysProcAttr{
		Setpgid: true,
	}

	for key, value := range procConfig.Env {
		attr.Env = append(attr.Env, key+"="+value)
	}

	argStrings := append([]string{procConfig.Command}, procConfig.Argv...)
	proc, err := os.StartProcess(procConfig.Command, argStrings, &attr)
	if err != nil {
		return err
	}

	entry := Process{
		Id:        procConfig.Id,
		WorkDir:   procConfig.WorkDir,
		StartedAt: time.Now(),
		Command:   procConfig.Command,
		Argv:      procConfig.Argv,
		Pid:       proc.Pid,
		Env:       procConfig.Env,
	}

	// Release the process so that it doesn't die on exit
	if err = proc.Release(); err != nil {
		return err
	}

	// Reap zombies
	go waitForExit(entry.Pid, m.logger)

	m.logger.Debug("proc created", log.Ctx{
		"proc": entry.Pid,
	})

	if err := w.Insert(entry); err != nil {
		return err
	}

	return nil
}
