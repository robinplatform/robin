package process

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/model"
)

var (
	logger = log.New("process")
)

type ProcessNamespace string

const (
	NamespaceExtensionDaemon ProcessNamespace = "extension-daemon"
	NamespaceExtensionLambda ProcessNamespace = "extension-lambda"
	NamespaceInternal        ProcessNamespace = "internal"
)

// TODO: Rename these three, they don't make sense. Also maybe some
// doc comments to help explain what they do.
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
	Args    []string
}

type Process struct {
	Id        ProcessId
	Pid       int
	StartedAt time.Time
	WorkDir   string
	Env       map[string]string
	Command   string
	Args      []string
}

func (process *Process) waitForExit(pid int) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		logger.Debug("Failed to find process to wait on", log.Ctx{
			"process": process,
			"err":     err,
		})
		return
	}

	if exitCode, err := proc.Wait(); err != nil {
		logger.Debug("Process exited with error", log.Ctx{
			"process":  process,
			"exitCode": exitCode,
			"err":      err,
		})
	} else {
		logger.Debug("Process exited", log.Ctx{
			"process": process,
		})
	}
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
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		cfg.WorkDir = dir
	}

	return nil
}

type ProcessManager struct {
	db model.Store[Process]
}

func NewProcessManager(dbPath string) (*ProcessManager, error) {
	manager := &ProcessManager{}

	var err error
	manager.db, err = model.NewStore[Process](dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create process database: %w", err)
	}

	return manager, nil
}

func (m *ProcessManager) IsAlive(id ProcessId) bool {
	r := m.db.ReadHandle()
	defer r.Close()

	procEntry, found := r.Find(findById(id))
	if !found {
		return false
	}

	// TODO: check the actual error, it might've been a permission error
	// or something else.
	process, err := os.FindProcess(procEntry.Pid)
	if err != nil {
		return false
	}

	// On windows, if we located a process, it's alive.
	// On other platforms, we only have a handle, and need to send a signal
	// to see if it's alive.
	if runtime.GOOS != "windows" {
		err = process.Signal(syscall.Signal(0))
		logger.Debug("got error on signal", log.Ctx{
			"err": err,
		})
	}

	return err == nil

}

// Kill will kill the process with the given id (not PID), and remove it from
// the internal database.
// TODO: Make this work on windows
func (m *ProcessManager) Kill(id ProcessId) error {
	w := m.db.WriteHandle()
	defer w.Close()

	procEntry, found := w.Find(findById(id))
	if !found {
		return fmt.Errorf("id %+v wasn't found in process database", id)
	}

	if err := syscall.Kill(-procEntry.Pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	if err := w.Delete(findById(id)); err != nil {
		return err
	}

	return nil
}

// TODO: 'SpawnPath' is a bad name for this, esp since it does the opposite of spawning
// from a path

func (m *ProcessManager) SpawnPath(config ProcessConfig) error {
	var err error
	config.Command, err = exec.LookPath(config.Command)
	if err != nil {
		return fmt.Errorf("failed to find command %s in $PATH: %w", config.Command, err)
	}

	return m.Spawn(config)
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

	logger.Info("Spawning Process", log.Ctx{
		"config": procConfig,
	})

	empty, err := os.Open(os.DevNull)
	if err != nil {
		return fmt.Errorf("failed to open null device: %w", err)
	}
	defer empty.Close()

	robinPath := config.GetRobinPath()
	procFolderPath := path.Join(robinPath, "processes")

	if err := os.MkdirAll(procFolderPath, 0755); err != nil {
		return fmt.Errorf("failed to create process folder: %w", err)
	}

	procPath := path.Join(procFolderPath, string(procConfig.Id.Namespace)+"-"+procConfig.Id.NamespaceKey+"-"+procConfig.Id.Key+".log")

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

	argStrings := append([]string{procConfig.Command}, procConfig.Args...)
	proc, err := os.StartProcess(procConfig.Command, argStrings, &attr)
	if err != nil {
		return err
	}

	entry := Process{
		Id:        procConfig.Id,
		WorkDir:   procConfig.WorkDir,
		StartedAt: time.Now(),
		Command:   procConfig.Command,
		Args:      procConfig.Args,
		Pid:       proc.Pid,
		Env:       procConfig.Env,
	}

	// Release the process so that it doesn't die on exit
	if err = proc.Release(); err != nil {
		return err
	}

	// Reap zombies
	go entry.waitForExit(entry.Pid)

	logger.Debug("Process created", log.Ctx{
		"process": entry,
	})

	if err := w.Insert(entry); err != nil {
		return err
	}

	return nil
}
