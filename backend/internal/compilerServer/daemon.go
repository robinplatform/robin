package compilerServer

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
	"strconv"
	"sync/atomic"
	"time"

	"robinplatform.dev/internal/compile/toolkit"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/process"
	"robinplatform.dev/internal/process/health"
	"robinplatform.dev/internal/project"
)

func (app *CompiledApp) IsAlive() bool {
	process, found := process.Manager.FindById(app.ProcessId)
	if !found {
		return false
	}

	return process.CheckHealth()
}

func (app *CompiledApp) keepAlive() {
	defer func() { atomic.StoreInt64(app.keepAliveRunning, 0) }()

	numErrs := 0
	for {
		if app.IsAlive() {
			numErrs = 0
		} else {
			numErrs++
			if numErrs >= 3 {
				logger.Warn("App server shutdown", log.Ctx{
					"appId": app.Id,
				})
				return
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func (app *CompiledApp) GetAppDir() (string, error) {
	appConfig, err := app.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get app config: %w", err)
	}

	if appConfig.ConfigPath.Scheme == "file" {
		return filepath.Dir(appConfig.ConfigPath.Path), nil
	}

	projectAlias, err := project.GetProjectAlias()
	if err != nil {
		return "", fmt.Errorf("failed to get project alias: %w", err)
	}
	return filepath.Join(config.GetRobinPath(), "projects", projectAlias, "apps", app.Id), nil
}

func (app *CompiledApp) setupJsDaemon(processConfig *process.ProcessConfig) error {
	// Ensure that the client is always built first
	if err := app.buildClient(); err != nil {
		return err
	}

	appDir, err := app.GetAppDir()
	if err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	appConfig, err := app.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	if err := app.buildServerBundle(); err != nil {
		return fmt.Errorf("failed to build app server: %w", err)
	}

	// Figure out asset paths
	daemonRunnerFilePath := filepath.Join(appDir, "robin-daemon-runner.js")
	serverBundleFilePath := filepath.Join(appDir, "daemon.bundle.js")
	if appConfig.ConfigPath.Scheme == "file" {
		buf := make([]byte, 4)
		rand.Read(buf)

		tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("robin-app-%s-%x", app.Id, buf))
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return fmt.Errorf("failed to start app server: %w", err)
		}

		daemonRunnerFilePath = filepath.Join(tmpDir, "robin-daemon-runner.js")
		serverBundleFilePath = filepath.Join(tmpDir, "daemon.bundle.js")
	}

	// Write the entrypoint to the temporary file
	if err := os.WriteFile(serverBundleFilePath, []byte(app.ServerJs), 0755); err != nil {
		return fmt.Errorf("failed to start app server: could not create entrypoint: %w", err)
	}

	// Extract the daemon runner onto disk
	daemonRunnerSourceFile, err := toolkit.ToolkitFS.Open("internal/app-daemon.js")
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find daemon runner: %w", err)
	}
	daemonRunnerSource, err := io.ReadAll(daemonRunnerSourceFile)
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find daemon runner: %w", err)
	}

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

	processConfig.Command = "node"
	processConfig.Args = []string{daemonRunnerFilePath}

	processConfig.Env["ROBIN_DAEMON_TARGET"] = serverBundleFilePath

	return nil
}

func (app *CompiledApp) setupCustomDaemon(appConfig project.RobinAppConfig, processConfig *process.ProcessConfig) error {
	processConfig.Command = appConfig.Daemon[0]
	processConfig.Args = appConfig.Daemon[1:]

	return nil
}

func (app *CompiledApp) copyAppFiles(appConfig project.RobinAppConfig, appDir string) error {
	if appConfig.ConfigPath.Scheme == "file" {
		return nil
	}

	for _, appFilePath := range appConfig.Files {
		_, buf, err := appConfig.ReadFile(httpClient, appFilePath)
		if err != nil {
			return fmt.Errorf("failed to setup app files: failed to read %s: %w", appFilePath, err)
		}

		fd, err := os.Create(filepath.Join(appDir, appFilePath))
		if err != nil {
			return fmt.Errorf("failed to setup app files: failed to create %s: %w", appFilePath, err)
		}

		if _, err := fd.Write(buf); err != nil {
			return fmt.Errorf("failed to setup app files: failed to write %s: %w", appFilePath, err)
		}
	}

	return nil
}

func (app *CompiledApp) StartServer() error {
	w := process.Manager.WriteHandle()
	defer w.Close()

	if proc, found := w.Read.FindById(app.ProcessId); found && proc.IsAlive() {
		return nil
	}

	appDir, err := app.GetAppDir()
	if err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	appConfig, err := app.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	// Setup the app's dependencies
	if err := app.copyAppFiles(appConfig, appDir); err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	projectPath := project.GetProjectPathOrExit()
	processConfig := process.ProcessConfig{
		Id:      app.ProcessId,
		WorkDir: appDir,
		Env: map[string]string{
			"ROBIN_APP_ID":       app.Id,
			"ROBIN_PROCESS_TYPE": "daemon",
			"ROBIN_PROJECT_PATH": projectPath,
		},
	}

	// Setup the daemon runner
	if appConfig.Daemon == nil {
		if err := app.setupJsDaemon(&processConfig); err != nil {
			return fmt.Errorf("failed to start app server: %w", err)
		}
	} else {
		if err := app.setupCustomDaemon(appConfig, &processConfig); err != nil {
			return fmt.Errorf("failed to start app server: %w", err)
		}
	}

	// Find a free port to listen on
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return fmt.Errorf("failed to start app server: could not find free port: %w", err)
	}
	portAvailable := listener.Addr().(*net.TCPAddr).Port
	strPortAvailable := strconv.FormatInt(int64(portAvailable), 10)
	listener.Close()

	// Add port info to the process config
	processConfig.Env["PORT"] = strPortAvailable
	processConfig.Port = portAvailable
	processConfig.HealthCheck = health.HttpHealthCheck{
		Method: http.MethodGet,
		Url:    fmt.Sprintf("http://localhost:%s/api/health", strPortAvailable),
	}

	// Start the app server process
	serverProcess, err := w.SpawnFromPathVar(processConfig)
	if err != nil && !errors.Is(err, process.ErrProcessAlreadyExists) {
		logger.Err("Failed to start app server", log.Ctx{
			"appId": app.Id,
			"err":   err,
		})
		return fmt.Errorf("failed to start app server: %w", err)
	}

	// Wait for process to become ready
	for i := 0; i < 10; i++ {
		// Make sure the process is still running
		if !serverProcess.IsAlive() {
			return fmt.Errorf("failed to start app server: process died")
		}

		// Send a ping to the process
		if serverProcess.CheckHealth() {
			if atomic.CompareAndSwapInt64(app.keepAliveRunning, 0, 1) {
				go app.keepAlive()
			}

			return nil
		}

		logger.Debug("Failed to ping app server", log.Ctx{
			"appId": app.Id,
			"pid":   serverProcess.Pid,
			"err":   err,
		})

		// Wait a bit
		time.Sleep(500 * time.Millisecond)
	}

	logger.Warn("Stopping unhealthy server", log.Ctx{
		"appId": app.Id,
		"pid":   serverProcess.Pid,
	})

	if err := app.stopServer(w); err != nil {
		logger.Warn("Failed to stop unhealthy app server", log.Ctx{
			"appId": app.Id,
			"pid":   serverProcess.Pid,
			"err":   err,
		})
	}

	return fmt.Errorf("failed to start app server: process did not become ready")
}

func (app *CompiledApp) stopServer(w process.WHandle) error {
	if err := w.Kill(app.ProcessId); err != nil && !errors.Is(err, process.ErrProcessNotFound) {
		return err
	}
	return nil
}

func (app *CompiledApp) StopServer() error {
	w := process.Manager.WriteHandle()
	defer w.Close()

	if err := app.stopServer(w); err != nil {
		return fmt.Errorf("failed to stop app server: %w", err)
	}
	return nil
}

type AppResponse struct {
	StatusCode int
	Err        string
	Body       []byte
}

func (app *CompiledApp) Request(ctx context.Context, method string, reqPath string, body any) AppResponse {
	serverProcess, found := process.Manager.FindById(app.ProcessId)
	if !found {
		return AppResponse{StatusCode: 500, Err: "failed to make app request: app process not found"}
	}

	serializedBody, err := json.Marshal(body)
	if err != nil {
		return AppResponse{StatusCode: 500, Err: fmt.Sprintf("failed to serialize app request body: %s", err)}
	}

	logger.Debug("Making app request", log.Ctx{
		"appId":  app.Id,
		"pid":    serverProcess.Pid,
		"method": method,
		"path":   reqPath,
	})
	req, err := http.NewRequestWithContext(
		ctx,
		method,
		fmt.Sprintf("http://localhost:%d%s", serverProcess.Port, reqPath),
		bytes.NewReader(serializedBody),
	)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return AppResponse{StatusCode: 500, Err: fmt.Sprintf("failed to create app request: %s", err)}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AppResponse{StatusCode: 500, Err: fmt.Sprintf("failed to make app request: %s", err)}
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return AppResponse{StatusCode: 500, Err: fmt.Sprintf("failed to read app response: %s (http status %d)", err, resp.StatusCode)}
	}

	if resp.StatusCode != http.StatusOK {
		return AppResponse{StatusCode: resp.StatusCode, Err: string(buf)}
	}

	return AppResponse{StatusCode: http.StatusOK, Body: buf}
}
