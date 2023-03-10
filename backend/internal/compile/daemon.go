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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/process"
	"robinplatform.dev/internal/project"

	es "github.com/evanw/esbuild/pkg/api"
)

var (
	// TODO: add something like a write handle to processManager so we don't
	// need to use our own mutex
	daemonProcessMux = &sync.Mutex{}
)

func getExtractServerPlugins(appConfig project.RobinAppConfig, app *CompiledApp) []es.Plugin {
	return []es.Plugin{
		{
			Name: "extract-server-ts",
			Setup: func(build es.PluginBuild) {
				build.OnLoad(es.OnLoadOptions{
					Filter: "\\.server\\.[jt]s$",
				}, func(args es.OnLoadArgs) (es.OnLoadResult, error) {
					var source []byte
					var err error

					if strings.HasPrefix(args.Path, "http://") || strings.HasPrefix(args.Path, "https://") {
						_, source, err = appConfig.ReadFile(&httpClient, args.Path)
					} else {
						source, err = os.ReadFile(args.Path)
					}
					if err != nil {
						return es.OnLoadResult{}, fmt.Errorf("failed to read server file %s: %w", args.Path, err)
					}

					exports, err := getFileExports(&es.StdinOptions{
						Contents:   string(source),
						Sourcefile: args.Path,
						Loader:     es.LoaderTS,
					})
					if err != nil {
						return es.OnLoadResult{}, fmt.Errorf("failed to get exports for %s: %w", args.Path, err)
					}

					serverPolyfill := "import { createRpcMethod } from '@robinplatform/toolkit/internal/rpc';\n\n"
					for _, export := range exports {
						serverPolyfill += fmt.Sprintf(
							"export const %s = createRpcMethod(%q, %q, %q);\n",
							export,
							appConfig.Id,
							args.Path,
							export,
						)
					}

					app.serverExports[args.Path] = exports

					return es.OnLoadResult{
						Contents: &serverPolyfill,
						Loader:   es.LoaderJS,
					}, nil
				})
			},
		},
	}
}

func (app *CompiledApp) getProcessId() process.ProcessId {
	projectAlias, err := project.GetProjectAlias()
	if err != nil {
		panic(fmt.Errorf("failed to get project alias: %w", err))
	}

	return process.ProcessId{
		Kind:   process.KindAppDaemon,
		Source: projectAlias,
		Key:    app.Id,
	}
}

func (app *CompiledApp) IsAlive() bool {
	process, err := process.Manager.FindById(app.getProcessId())
	if err != nil {
		return false
	}

	if !process.IsAlive() {
		return false
	}

	// Send a ping to the process
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", process.Port))
	if resp != nil {
		resp.Body.Close()
	}
	return err == nil && resp.StatusCode == http.StatusOK
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

		// TODO: This should be less frequent. I've set it to be higher right now
		// to make development of robin apps easier/faster, but it should be lower.
		if config.GetReleaseChannel() == "dev" {
			time.Sleep(time.Second / 4)
		} else {
			time.Sleep(10 * time.Second)
		}
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
	appDir, err := app.GetAppDir()
	if err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	appConfig, err := app.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to start app server: %w", err)
	}

	// Build the server bundle, if not already built
	if app.ServerJs == "" {
		if err := app.buildServerBundle(); err != nil {
			return fmt.Errorf("failed to build app server: %w", err)
		}
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
	daemonRunnerSourceFile, err := toolkitFS.Open("internal/app-daemon.js")
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
		_, buf, err := appConfig.ReadFile(&httpClient, appFilePath)
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
	daemonProcessMux.Lock()
	defer daemonProcessMux.Unlock()

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
		Id:      app.getProcessId(),
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

	// Start the app server process
	serverProcess, err := process.Manager.SpawnPath(processConfig)
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
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", serverProcess.Port))
		if err == nil && resp.StatusCode == http.StatusOK {
			if atomic.CompareAndSwapInt64(app.keepAliveRunning, 0, 1) {
				go app.keepAlive()
			}

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
		time.Sleep(500 * time.Millisecond)
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

	if err := process.Manager.Kill(app.getProcessId()); err != nil && !errors.Is(err, process.ErrProcessNotFound) {
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
	if app.httpClient == nil {
		app.httpClient = &http.Client{}
	}

	serverProcess, err := process.Manager.FindById(app.getProcessId())
	if err != nil {
		return AppResponse{StatusCode: 500, Err: fmt.Sprintf("failed to make app request: %s", err)}
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

	resp, err := app.httpClient.Do(req)
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
