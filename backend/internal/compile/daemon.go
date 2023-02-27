package compile

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/process"
)

type processMeta struct {
	Port int `json:"port"`
}

var (
	processManager *process.ProcessManager[processMeta]

	// TODO: add something like a write handle to processManager so we don't
	// need to use our own mutex
	daemonProcessMux = &sync.Mutex{}
)

func init() {
	robinPath := config.GetRobinPath()

	var err error
	processManager, err = process.NewProcessManager[processMeta](filepath.Join(
		robinPath,
		"app-processes.db",
	))
	if err != nil {
		panic(fmt.Errorf("failed to initialize compiler: %w", err))
	}
}

func (app *CompiledApp) getProcessId() process.ProcessId {
	return process.ProcessId{
		Namespace:    process.NamespaceExtensionDaemon,
		NamespaceKey: "app-daemon",
		Key:          app.Id,
	}
}

func (app *CompiledApp) StartServer() error {
	daemonProcessMux.Lock()
	defer daemonProcessMux.Unlock()

	// Figure out a temporary file name to write the entrypoint to
	tmpFileName := ""
	for {
		tmpDir := os.TempDir()
		ext := ""
		if runtime.GOOS == "windows" {
			ext = ".exe"
		}

		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
		}
		tmpFileName = filepath.Join(tmpDir, fmt.Sprintf("robin-app-server-%s-%x%s", app.Id, buf, ext))

		if _, err := os.Stat(tmpFileName); os.IsNotExist(err) {
			break
		} else if !os.IsExist(err) {
			return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
		}
	}

	// Write the entrypoint to the temporary file
	if err := os.WriteFile(tmpFileName, []byte(app.ServerJs), 0755); err != nil {
		return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
	}

	// Find a free port to listen on
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find free port: %w", err)
	}
	portAvailable := listener.Addr().(*net.TCPAddr).Port
	strPortAvailable := strconv.FormatInt(int64(portAvailable), 10)
	listener.Close()

	// Extract the daemon runner onto disk
	daemonRunnerSourceFile, err := toolkitFS.Open("internal/app-daemon.js")
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find daemon runner: %w", err)
	}
	daemonRunnerSource, err := io.ReadAll(daemonRunnerSourceFile)
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find daemon runner: %w", err)
	}

	daemonRunnerFilePath := filepath.Join(os.TempDir(), "robin-daemon-runner.js")
	daemonRunnerFile, err := os.Create(daemonRunnerFilePath)
	if err != nil {
		return fmt.Errorf("failed to start app server: could not create daemon runner: %w", err)
	}
	if _, err := daemonRunnerFile.Write(daemonRunnerSource); err != nil {
		return fmt.Errorf("failed to start app server: could not create daemon runner: %w", err)
	}
	if err := daemonRunnerFile.Close(); err != nil {
		return fmt.Errorf("failed to start app server: could not create daemon runner: %w", err)
	}

	// Start the app server process
	projectPath := config.GetProjectPathOrExit()
	serverProcess, err := processManager.SpawnPath(process.ProcessConfig[processMeta]{
		Id: process.ProcessId{
			Namespace:    process.NamespaceExtensionDaemon,
			NamespaceKey: "app-daemon",
			Key:          app.Id,
		},
		Command: "node",
		Args:    []string{daemonRunnerFilePath},
		WorkDir: projectPath,
		Env: map[string]string{
			"ROBIN_PROCESS_TYPE":  "daemon",
			"ROBIN_DAEMON_TARGET": tmpFileName,
			"PORT":                strPortAvailable,
		},
		Meta: processMeta{
			Port: portAvailable,
		},
	})
	if err != nil && !errors.Is(err, process.ErrProcessAlreadyExists) {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	// Wait for process to become ready
	for i := 0; i < 10; i++ {
		// Make sure the process is still running
		if !serverProcess.IsAlive() {
			return fmt.Errorf("failed to start app server: process died")
		}

		// Send a ping to the process
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", serverProcess.Meta.Port))
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		if resp == nil {
			logger.Debug("Failed to ping app server", log.Ctx{
				"appId": app.Id,
				"pid":   serverProcess.Pid,
				"err":   err,
			})
		} else {
			logger.Debug("Failed to ping app server", log.Ctx{
				"appId":  app.Id,
				"pid":    serverProcess.Pid,
				"err":    err,
				"status": resp.StatusCode,
			})
		}

		// Wait a bit
		time.Sleep(1 * time.Second)
	}

	if err := app.StopServer(); err != nil {
		logger.Warn("Failed to stop unhealthy app server", log.Ctx{
			"appId": app.Id,
			"pid":   serverProcess.Pid,
			"err":   err,
		})
	}

	return fmt.Errorf("failed to start app server: process did not become ready")
}

func (app *CompiledApp) StopServer() error {
	daemonProcessMux.Lock()
	defer daemonProcessMux.Unlock()

	if err := processManager.Kill(app.getProcessId()); err != nil && !errors.Is(err, process.ErrProcessNotFound) {
		return fmt.Errorf("failed to stop app server: %w", err)
	}
	return nil
}

func (app *CompiledApp) Request(ctx context.Context, method string, reqPath string, body map[string]any) (any, error) {
	if app.httpClient == nil {
		app.httpClient = &http.Client{}
	}

	serverProcess, err := processManager.FindById(app.getProcessId())
	if err != nil {
		return nil, fmt.Errorf("failed to make app request: %w", err)
	}

	serializedBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize app request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		fmt.Sprintf("http://localhost:%d%s", serverProcess.Meta.Port, reqPath),
		bytes.NewReader(serializedBody),
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("failed to create app request: %w", err)
	}

	resp, err := app.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make app request: %w", err)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read app response: %w (http status %d)", err, resp.StatusCode)
	}

	var respBody struct {
		Type   string
		Error  string
		Result any
	}
	if err := json.Unmarshal(buf, &respBody); err != nil {
		return nil, fmt.Errorf("failed to deserialize app response: %w (http status %d)", err, resp.StatusCode)
	}

	if respBody.Type == "error" {
		return nil, fmt.Errorf("failed to make app request: %s", respBody.Error)
	}

	return respBody.Result, nil
}
