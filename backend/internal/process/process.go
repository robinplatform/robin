package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/nxadm/tail"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/model"
	"robinplatform.dev/internal/pubsub"
)

var (
	logger = log.New("process")
)

// An identifier for a process.
type ProcessId struct {
	// The name of the system/app that spawned this process
	// The following names are reserved:
	// - robin - this is for internal apps
	// - @robin/* - anything starting with @robin-platform/* is reserved for systems in Robin
	Source string `json:"source"`
	// The name that this process has been given
	Key string `json:"key"`
}

func (id ProcessId) String() string {
	return fmt.Sprintf(
		"%s-%s",
		id.Source,
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

func (id *ProcessId) getLogFilePath() string {
	robinPath := config.GetRobinPath()
	processLogsFolderPath := filepath.Join(robinPath, "logs", "processes")
	processLogsPath := filepath.Join(processLogsFolderPath, id.Source+"-"+id.Key+".log")
	return processLogsPath
}

type Process struct {
	Id        ProcessId         `json:"id"`
	Pid       int               `json:"pid"`
	StartedAt time.Time         `json:"startedAt"`
	WorkDir   string            `json:"workDir"`
	Env       map[string]string `json:"env"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Port      int               `json:"port"` // see docs in ProcessConfig

	// NOTE: The context and cancel fields are only valid because
	// the store doesn't re-load data from disk when the file is updated.

	Context context.Context `json:"-"` // This Context gets canceled when the process dies.
	cancel  func()          `json:"-"` // Cancel the context
}

func InternalId(name string) ProcessId {
	return ProcessId{
		Source: "robin",
		Key:    name,
	}
}

func NewId(source string, name string) (ProcessId, error) {
	if name == "robin" {
		return ProcessId{}, fmt.Errorf("tried to use internal \"robin\" namespace")
	}

	if strings.HasPrefix(name, "@robin/") {
		return ProcessId{}, fmt.Errorf("tried to use internal \"@robin/*\" namespace")

	}

	return ProcessId{
		Source: source,
		Key:    name,
	}, nil
}

func (process *Process) waitForExit() {
	proc, err := os.FindProcess(process.Pid)
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

func (process *Process) osProcessIsAlive() bool {
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

	if cfg.Id.Source == "" {
		return fmt.Errorf("cannot create process without a source")
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

// This is essentially a global type, but it's set up as an instance for testing purposes.
// Use `process.Manager` to manage processes.
type ProcessManager struct {
	// Data persisted to disk about processes
	db model.Store[Process]

	// Context for long running operations, the parent
	// of all process contexts
	ctx context.Context
	// Cancel function for the context
	cancel func()
}

func NewProcessManager(dbPath string) (*ProcessManager, error) {
	manager := &ProcessManager{}

	var err error
	manager.db, err = model.NewStore[Process](dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create process database: %w", err)
	}

	manager.ctx, manager.cancel = context.WithCancel(context.Background())

	manager.db.ForEachWriting(func(proc *Process) {
		proc.Context, proc.cancel = context.WithCancel(manager.ctx)
	})

	// This needs to be copied out because otherwise you'd have a situation where
	// the process being referenced is modified by another thread in parallel
	for _, proc := range manager.db.ShallowCopyOutData() {
		if !proc.osProcessIsAlive() {
			proc.cancel()
			continue
		}

		go proc.pollForExit()
	}

	return manager, nil
}

// This polls to see if the process is still alive; this is necessary
// because if the process is not our child, we can't use process.Wait()
// anymore. This can happen if robin restarts but the child is still alive.
func (proc *Process) pollForExit() {
	for {
		if !proc.osProcessIsAlive() {
			proc.cancel()
			return
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (r *RHandle) FindById(id ProcessId) (*Process, error) {
	procEntry, found := r.db.Find(findById(id))
	if !found {
		return nil, processNotFound(id)
	}
	return &procEntry, nil
}

func (r *RHandle) IsAlive(id ProcessId) bool {
	process, err := r.FindById(id)
	if err != nil {
		return false
	}
	return process.IsAlive()
}

func (process *Process) pipeTailIntoTopic(topic *pubsub.Topic) {
	defer topic.Close()

	logger.Debug("Starting pipe into topic", log.Ctx{
		"topic": topic.Id.String(),
	})

	config := tail.Config{
		ReOpen: true,
		Follow: true,
	}
	out, err := tail.TailFile(process.Id.getLogFilePath(), config)
	if err != nil {
		logger.Err("failed to tail file", log.Ctx{
			"err": err.Error(),
		})
		return
	}

	defer out.Cleanup()

	for {
		select {
		case <-process.Context.Done():
			return

		case line, ok := <-out.Lines:
			if !ok {
				return
			}

			if line.Err != nil {
				logger.Err("got error in tail line", log.Ctx{
					"err": line.Err.Error(),
				})
			}

			topic.Publish(line.Text)
		}
	}
}

// This reads the path variable to find the right executable.
func (w *WHandle) SpawnFromPathVar(config ProcessConfig) (*Process, error) {
	var err error
	config.Command, err = exec.LookPath(config.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to find command %s in $PATH: %w", config.Command, err)
	}

	return w.Spawn(config)
}

// This spawns a process using the given arguments and executable path.
func (w *WHandle) Spawn(procConfig ProcessConfig) (*Process, error) {
	if err := procConfig.fillEmptyValues(); err != nil {
		return nil, err
	}

	prev, found := w.db.Find(findById(procConfig.Id))
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
		if err := w.Remove(prev.Id); err != nil {
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

	processLogsPath := procConfig.Id.getLogFilePath()
	processLogsFolderPath := filepath.Dir(processLogsPath)

	if err := os.MkdirAll(processLogsFolderPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create process folder: %w", err)
	}

	// Don't close the file, instead pass it on to the tail goroutine later on
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

	topicId := pubsub.TopicId{
		Category: fmt.Sprintf("@robin/logs/%s", procConfig.Id.Source),
		Name:     procConfig.Id.Key,
	}

	topic, err := pubsub.Topics.CreateTopic(topicId)
	if err != nil {
		logger.Err("error creating topic", log.Ctx{
			"err": err.Error(),
		})
		_ = proc.Kill()
		return nil, err
	}

	ctx, cancel := context.WithCancel(w.Read.m.ctx)

	entry := Process{
		Id:        procConfig.Id,
		WorkDir:   procConfig.WorkDir,
		StartedAt: time.Now(),
		Command:   procConfig.Command,
		Args:      procConfig.Args,
		Pid:       proc.Pid,
		Env:       procConfig.Env,
		Port:      procConfig.Port,

		Context: ctx,
		cancel:  cancel,
	}

	// Write output to file
	go entry.pipeTailIntoTopic(topic)

	// Reap zombies
	go entry.waitForExit()

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

		return nil, err
	}

	// Release the process so that it doesn't die on exit
	if err = proc.Release(); err != nil {
		return nil, err
	}

	return &entry, nil
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
