package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"robinplatform.dev/internal/identity"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/model"
	"robinplatform.dev/internal/process/health"
	"robinplatform.dev/internal/pubsub"
)

var (
	logger = log.New("process")
)

// An identifier for a process.
type ProcessId identity.Id

func (p ProcessId) String() string {
	return (identity.Id)(p).String()
}

type ProcessConfig struct {
	Id      ProcessId
	WorkDir string
	Env     map[string]string
	Command string
	Args    []string
	Port    int

	HealthCheck health.HealthCheck
}

type Process struct {
	Id        ProcessId         `json:"id"`
	Pid       int               `json:"pid"`
	StartedAt time.Time         `json:"startedAt"`
	WorkDir   string            `json:"workDir"`
	Env       map[string]string `json:"env"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Port      int               `json:"port"`

	HealthCheck health.SerializableHealthCheck `json:"healthCheck"`

	// NOTE: The fields below are only valid because
	// the store doesn't re-load data from disk when the file is updated.
	// They're not serializable, and get filled in at startup.

	logsTopic *pubsub.Topic[string] `json:"-"`
	Context   context.Context       `json:"-"` // This Context gets canceled when the process dies.
	cancel    func()                `json:"-"` // Cancel the context
}

func waitForExit(process pollPidContext) {
	proc, err := os.FindProcess(process.pid)
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

	process.cancel()
}

func (process *Process) IsAlive() bool {
	select {
	case <-process.Context.Done():
		return false
	default:
		return true
	}
}

type LogFileResult struct {
	Text    string `json:"text"`
	Counter int32  `json:"counter"` // TODO: bad name
}

func findById(id ProcessId) func(row Process) bool {
	return func(row Process) bool {
		return row.Id == id
	}
}

func (cfg *ProcessConfig) fillEmptyValues() error {
	if cfg.Id.Key == "" {
		return fmt.Errorf("cannot create process without a key")
	}

	if cfg.Id.Category == "" {
		return fmt.Errorf("cannot create process without a category")
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

	if cfg.HealthCheck == nil {
		cfg.HealthCheck = &health.ProcessHealthCheck{}
	}

	return nil
}

// This is essentially a global type, but it's set up as an instance for testing purposes.
// Use `process.Manager` to manage processes.
type ProcessManager struct {
	processLogsFolderPath string

	// Data persisted to disk about processes
	db model.Store[Process]

	registry *pubsub.Registry

	// Context for long running operations, the parent
	// of all process contexts
	ctx context.Context
	// Cancel function for the context
	cancel func()
}

func NewProcessManager(registry *pubsub.Registry, logsPath string, dbPath string) (*ProcessManager, error) {
	manager := &ProcessManager{}

	var err error
	manager.db, err = model.NewStore[Process](dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create process database: %w", err)
	}

	manager.processLogsFolderPath = logsPath
	manager.registry = registry

	manager.ctx, manager.cancel = context.WithCancel(context.Background())

	procIds := make([]pollPidContext, 0)
	var topicCreationErr error
	err = manager.db.ForEachWriting(func(proc *Process) {
		proc.Context, proc.cancel = context.WithCancel(manager.ctx)

		if !health.PidIsAlive(proc.Pid) {
			proc.cancel()
			return
		}

		procIds = append(procIds, pollPidContext{
			pid:    proc.Pid,
			cancel: proc.cancel,
		})

		topic, err := manager.logTopicForProcId(proc.Id)
		if err != nil {
			topicCreationErr = err
			return
		}

		proc.logsTopic = topic

		go manager.pipeTailIntoTopic(topicTailInfo{
			processId: proc.Id,
			logsTopic: topic,
			Context:   proc.Context,
		})
	})
	if topicCreationErr != nil {
		return nil, topicCreationErr
	}
	if err != nil {
		return nil, err
	}

	// Hand off procIds to the goroutine
	go pollForExit(procIds)

	return manager, nil
}

// This polls to see if the process is still alive; this is necessary
// because if the process is not our child, we can't use process.Wait()
// anymore. This can happen if robin restarts but the child is still alive.
type pollPidContext struct {
	cancel func()
	pid    int
}

func pollForExit(processes []pollPidContext) {
	for {
		if len(processes) == 0 {
			return
		}

		nextProcesses := make([]pollPidContext, 0, len(processes))
		for _, proc := range processes {
			if !health.PidIsAlive(proc.pid) {
				proc.cancel()
			} else {
				nextProcesses = append(nextProcesses, proc)
			}
		}

		processes = nextProcesses

		time.Sleep(200 * time.Millisecond)
	}
}

func (r *RHandle) FindById(id ProcessId) (Process, bool) {
	procEntry, found := r.db.Find(findById(id))
	if !found {
		return Process{}, false
	}
	return procEntry, true
}

func (r *RHandle) IsAlive(id ProcessId) bool {
	process, found := r.FindById(id)
	if !found {
		return false
	}
	return process.IsAlive()
}

func (r *RHandle) CheckHealth(id ProcessId) bool {
	process, found := r.FindById(id)
	if !found {
		return false
	}
	return process.CheckHealth()
}

func (proc *Process) CheckHealth() bool {
	return proc.HealthCheck.Check(health.RunningProcessInfo{Pid: proc.Pid, Port: proc.Port})
}

// This reads the path variable to find the right executable.
func (w *WHandle) SpawnFromPathVar(config ProcessConfig) (Process, error) {
	var err error
	config.Command, err = exec.LookPath(config.Command)
	if err != nil {
		return Process{}, fmt.Errorf("failed to find command %s in $PATH: %w", config.Command, err)
	}

	return w.Spawn(config)
}

// This spawns a process using the given arguments and executable path.
func (w *WHandle) Spawn(procConfig ProcessConfig) (Process, error) {
	if err := procConfig.fillEmptyValues(); err != nil {
		return Process{}, err
	}

	healthCheck, err := health.NewHealthCheck(procConfig.HealthCheck)
	if err != nil {
		return Process{}, err
	}

	prev, found := w.db.Find(findById(procConfig.Id))
	if found {
		if prev.IsAlive() {
			logger.Debug("Found previous process", log.Ctx{
				"processId": procConfig.Id,
				"pid":       prev.Pid,
			})
			return prev, processExists(procConfig.Id)
		}

		logger.Debug("Found previous dead process entry, deleting it", log.Ctx{
			"processId": procConfig.Id,
		})
		if err := w.Remove(prev.Id); err != nil {
			return Process{}, fmt.Errorf("failed to delete previous process: %w", err)
		}
	}

	logger.Info("Spawning Process", log.Ctx{
		"config": procConfig,
	})

	empty, err := os.Open(os.DevNull)
	if err != nil {
		return Process{}, fmt.Errorf("failed to open null device: %w", err)
	}
	defer empty.Close()

	processLogsPath := w.Read.m.getLogFilePath(procConfig.Id)
	processLogsFolderPath := filepath.Dir(processLogsPath)

	if err := os.MkdirAll(processLogsFolderPath, 0755); err != nil {
		return Process{}, fmt.Errorf("failed to create process folder: %w", err)
	}

	// Don't close the file, instead pass it on to the tail goroutine later on
	output, err := os.Create(processLogsPath)
	if err != nil {
		return Process{}, err
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
		return Process{}, err
	}
	defer proc.Release()

	topic, err := w.Read.m.logTopicForProcId(procConfig.Id)
	if err != nil {
		_ = proc.Kill()
		return Process{}, err
	}

	ctx, cancel := context.WithCancel(w.Read.m.ctx)

	entry := Process{
		Id:          procConfig.Id,
		WorkDir:     procConfig.WorkDir,
		StartedAt:   time.Now(),
		Command:     procConfig.Command,
		Args:        procConfig.Args,
		Pid:         proc.Pid,
		Env:         procConfig.Env,
		Port:        procConfig.Port,
		HealthCheck: healthCheck,

		logsTopic: topic,
		Context:   ctx,
		cancel:    cancel,
	}

	// Write output to file
	go w.Read.m.pipeTailIntoTopic(topicTailInfo{
		processId: entry.Id,
		logsTopic: entry.logsTopic,
		Context:   entry.Context,
	})

	// Reap zombies
	go waitForExit(pollPidContext{
		pid:    entry.Pid,
		cancel: entry.cancel,
	})

	logger.Debug("Process created", log.Ctx{
		"id":       entry.Id,
		"pid":      entry.Pid,
		"logsPath": processLogsPath,
	})

	if err := w.db.Insert(entry); err != nil {
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

		return Process{}, err
	}

	return entry, nil
}

// Remove will kill the process if it is alive, and then remove it from the database
func (w *WHandle) Remove(id ProcessId) error {
	procEntry, found := w.db.Find(findById(id))
	if !found {
		return nil
	}

	if procEntry.IsAlive() {
		if err := w.Kill(id); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	if err := w.db.Delete(findById(id)); err != nil {
		return fmt.Errorf("failed to delete process: %w", err)
	}

	return nil
}

// TODO:
//   - Maybe this should take in a function and allow the user
//     to change the data before its outputted
//   - Also, since there's no GC right now, old processes
//     that are dead will still have their entries in the DB
func (r *RHandle) CopyOutData() []Process {
	data := r.db.ShallowCopyOutData()

	for i := 0; i < len(data); i += 1 {
		proc := &data[i]

		env := proc.Env
		proc.Env = make(map[string]string, len(env))
		for k, v := range env {
			proc.Env[k] = v
		}

		args := proc.Args
		proc.Args = make([]string, 0, len(args))
		proc.Args = append(proc.Args, args...)
	}

	return data
}
