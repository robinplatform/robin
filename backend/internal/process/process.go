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

func (id ProcessId) String() string {
	return fmt.Sprintf(
		"%s-%s-%s",
		id.Namespace,
		id.NamespaceKey,
		id.Key,
	)
}

type ProcessConfig[Meta any] struct {
	Id      ProcessId
	WorkDir string
	Env     map[string]string
	Command string
	Args    []string
	Meta    Meta
}

type Process[Meta any] struct {
	Id        ProcessId
	Pid       int
	StartedAt time.Time
	WorkDir   string
	Env       map[string]string
	Command   string
	Args      []string
	Meta      Meta
}

// TODO: Avoid logging entire Env

func (process *Process[_]) waitForExit(pid int) {
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

func (process *Process[_]) IsAlive() bool {
	// TODO: check the actual error, it might've been a permission error
	// or something else.
	osProcess, err := os.FindProcess(process.Pid)
	if err != nil {
		return false
	}

	// On windows, if we located a process, it's alive.
	// On other platforms, we only have a handle, and need to send a signal
	// to see if it's alive.
	if runtime.GOOS == "windows" {
		return true
	}

	return osProcess.Signal(syscall.Signal(0)) == nil
}

func findById[Meta any](id ProcessId) func(row Process[Meta]) bool {
	return func(row Process[Meta]) bool {
		return row.Id == id
	}
}

func (cfg *ProcessConfig[Meta]) fillEmptyValues() error {
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

type ProcessManager[Meta any] struct {
	db model.Store[Process[Meta]]
}

func NewProcessManager[Meta any](dbPath string) (*ProcessManager[Meta], error) {
	manager := &ProcessManager[Meta]{}

	var err error
	manager.db, err = model.NewStore[Process[Meta]](dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create process database: %w", err)
	}

	return manager, nil
}

func (manager *ProcessManager[Meta]) FindById(id ProcessId) (*Process[Meta], error) {
	r := manager.db.ReadHandle()
	defer r.Close()

	procEntry, found := r.Find(findById[Meta](id))
	if !found {
		return nil, processNotFound(id)
	}
	return &procEntry, nil
}

func (m *ProcessManager[Meta]) IsAlive(id ProcessId) bool {
	r := m.db.ReadHandle()
	defer r.Close()

	procEntry, found := r.Find(findById[Meta](id))
	if !found {
		return false
	}

	return procEntry.IsAlive()
}

// TODO: 'SpawnPath' is a bad name for this, esp since it does the opposite of spawning
// from a path

func (m *ProcessManager[Meta]) SpawnPath(config ProcessConfig[Meta]) (*Process[Meta], error) {
	var err error
	config.Command, err = exec.LookPath(config.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to find command %s in $PATH: %w", config.Command, err)
	}

	return m.Spawn(config)
}

func (m *ProcessManager[Meta]) Spawn(procConfig ProcessConfig[Meta]) (*Process[Meta], error) {
	if err := procConfig.fillEmptyValues(); err != nil {
		return nil, err
	}

	w := m.db.WriteHandle()
	defer w.Close()

	prev, found := w.Find(findById[Meta](procConfig.Id))
	if found {
		if prev.IsAlive() {
			return &prev, processExists(procConfig.Id)
		}
		if err := w.Delete(findById[Meta](procConfig.Id)); err != nil {
			return nil, fmt.Errorf("failed to delete previous process: %w", err)
		}
	}

	logger.Info("Spawning Process", log.Ctx{
		"config": procConfig,
	})

	empty, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("failed to open null device: %w", err)
	}
	defer empty.Close()

	robinPath := config.GetRobinPath()
	procFolderPath := path.Join(robinPath, "processes")

	if err := os.MkdirAll(procFolderPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create process folder: %w", err)
	}

	procPath := path.Join(procFolderPath, string(procConfig.Id.Namespace)+"-"+procConfig.Id.NamespaceKey+"-"+procConfig.Id.Key+".log")

	output, err := os.OpenFile(procPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	defer output.Close()

	var attr os.ProcAttr
	attr.Env = make([]string, 0, len(procConfig.Env))
	attr.Dir = procConfig.WorkDir
	attr.Files = []*os.File{empty, output, output}
	attr.Sys = getProcessSysAttrs()

	for key, value := range procConfig.Env {
		attr.Env = append(attr.Env, key+"="+value)
	}

	argStrings := append([]string{procConfig.Command}, procConfig.Args...)
	proc, err := os.StartProcess(procConfig.Command, argStrings, &attr)
	if err != nil {
		return nil, err
	}

	entry := Process[Meta]{
		Id:        procConfig.Id,
		WorkDir:   procConfig.WorkDir,
		StartedAt: time.Now(),
		Command:   procConfig.Command,
		Args:      procConfig.Args,
		Pid:       proc.Pid,
		Env:       procConfig.Env,
		Meta:      procConfig.Meta,
	}

	// Release the process so that it doesn't die on exit
	if err = proc.Release(); err != nil {
		return nil, err
	}

	// Reap zombies
	go entry.waitForExit(entry.Pid)

	logger.Debug("Process created", log.Ctx{
		"process": entry,
	})

	if err := w.Insert(entry); err != nil {
		return nil, err
	}

	return &entry, nil
}
