package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

type ProcessConfig struct {
	Id      ProcessId
	WorkDir string
	Env     map[string]string
	Command string
	Args    []string

	// Ideally the port should be optional, and be somewhat integrated into
	// whatever the healthcheck code ends up being, but for now this works decently well.
	Port int
}

func (processConfig *ProcessConfig) getLogFilePath() string {
	robinPath := config.GetRobinPath()
	processLogsFolderPath := filepath.Join(robinPath, "logs", "processes")
	processLogsPath := filepath.Join(processLogsFolderPath, string(processConfig.Id.Namespace)+"-"+processConfig.Id.NamespaceKey+"-"+processConfig.Id.Key+".log")
	return processLogsPath
}

type Process struct {
	Id        ProcessId
	Pid       int
	StartedAt time.Time
	WorkDir   string
	Env       map[string]string
	Command   string
	Args      []string
	Port      int // see above
}

// TODO: Avoid logging entire Env

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

func (process *Process) IsAlive() bool {
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

	parentEnv := os.Environ()
	env := make(map[string]string, len(parentEnv)+len(cfg.Env))

	// copy over parent env first
	for _, envVar := range parentEnv {
		parts := strings.SplitN(envVar, "=", 2)
		env[parts[0]] = parts[1]
	}

	// then override with any custom env vars
	for k, v := range cfg.Env {
		env[k] = v
	}

	// and replace the config's env with the new one
	cfg.Env = env

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

func (manager *ProcessManager) FindById(id ProcessId) (*Process, error) {
	r := manager.db.ReadHandle()
	defer r.Close()

	procEntry, found := r.Find(findById(id))
	if !found {
		return nil, processNotFound(id)
	}
	return &procEntry, nil
}

func (m *ProcessManager) IsAlive(id ProcessId) bool {
	process, err := m.FindById(id)
	if err != nil {
		return false
	}
	return process.IsAlive()
}

// TODO: 'SpawnPath' is a bad name for this, esp since it does the opposite of spawning
// from a path

func (m *ProcessManager) SpawnPath(config ProcessConfig) (*Process, error) {
	var err error
	config.Command, err = exec.LookPath(config.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to find command %s in $PATH: %w", config.Command, err)
	}

	return m.Spawn(config)
}

func (m *ProcessManager) Spawn(procConfig ProcessConfig) (*Process, error) {
	if err := procConfig.fillEmptyValues(); err != nil {
		return nil, err
	}

	w := m.db.WriteHandle()
	defer w.Close()

	prev, found := w.Find(findById(procConfig.Id))
	if found {
		if prev.IsAlive() {
			logger.Debug("Found previous process", log.Ctx{
				"processId": procConfig.Id,
				"pid":       prev.Pid,
			})
			return &prev, processExists(procConfig.Id)
		}

		logger.Debug("Found previous dead process entry, deleting it", log.Ctx{
			"processId": procConfig.Id,
		})
		if err := m.remove(w, prev.Id); err != nil {
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

	processLogsPath := procConfig.getLogFilePath()
	processLogsFolderPath := filepath.Dir(processLogsPath)

	if err := os.MkdirAll(processLogsFolderPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create process folder: %w", err)
	}

	output, err := os.Create(processLogsPath)
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

	entry := Process{
		Id:        procConfig.Id,
		WorkDir:   procConfig.WorkDir,
		StartedAt: time.Now(),
		Command:   procConfig.Command,
		Args:      procConfig.Args,
		Pid:       proc.Pid,
		Env:       procConfig.Env,
	}

	// Reap zombies
	go entry.waitForExit(entry.Pid)

	logger.Debug("Process created", log.Ctx{
		"id":       entry.Id,
		"pid":      entry.Pid,
		"logsPath": processLogsPath,
	})

	if err := w.Insert(entry); err != nil {
		logger.Debug("Failed to insert process into database", log.Ctx{
			"error": err.Error(),
		})

		// If we failed to insert the process into the database, kill it
		// so we don't end up with an unmanaged process
		if err := proc.Kill(); err != nil {
			logger.Err("Failed to kill unmanaged process", log.Ctx{
				"error":   err,
				"process": entry,
			})
		}

		return nil, err
	}

	// Release the process so that it doesn't die on exit
	if err = proc.Release(); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Remove will kill the process if it is alive, and then remove it from the database
func (manager *ProcessManager) remove(db model.WHandle[Process], id ProcessId) error {
	procEntry, found := db.Find(findById(id))
	if !found {
		return nil
	}

	if procEntry.IsAlive() {
		if err := manager.Kill(id); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	if err := db.Delete(findById(id)); err != nil {
		return fmt.Errorf("failed to delete process: %w", err)
	}

	return nil
}

// Remove will kill the process if it is alive, and then remove it from the database
func (manager *ProcessManager) Remove(id ProcessId) error {
	db := manager.db.WriteHandle()
	defer db.Close()
	return manager.remove(db, id)
}
