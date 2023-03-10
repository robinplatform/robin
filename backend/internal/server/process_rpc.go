package server

import (
	"fmt"
	"net/http"
	"path/filepath"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/process"
)

type procMeta struct{}

var processManager *process.ProcessManager[procMeta]

func init() {
	robinPath := config.GetRobinPath()

	var err error
	processManager, err = process.NewProcessManager[procMeta](filepath.Join(
		robinPath,
		"data",
		"app-spawned-processes.db",
	))
	if err != nil {
		panic(fmt.Errorf("failed to initialize app process spawner: %w", err))
	}
}

type StartProcessForAppInput struct {
	AppId      string `json:"appId"`
	ProcessKey string `json:"processKey"`
	Command    string `json:"command"`
}

var StartProcessForApp = AppsRpcMethod[StartProcessForAppInput, map[string]any]{
	Name: "StartProcessForApp",
	Run: func(req RpcRequest[StartProcessForAppInput]) (map[string]any, *HttpError) {
		processConfig := process.ProcessConfig[procMeta]{
			Command: req.Data.Command,
			Id: process.ProcessId{
				Namespace:    process.NamespaceExtensionSpawned,
				NamespaceKey: req.Data.AppId,
				Key:          req.Data.ProcessKey,
			},
		}

		proc, err := processManager.Spawn(processConfig)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to spawn new process %s: %s", req.Data.AppId, err),
			}
		}

		return map[string]any{
			"processKey": proc.Id,
			"pid":        proc.Pid,
		}, nil
	},
}

type CheckProcessHealthInput struct {
	AppId      string `json:"appId"`
	ProcessKey string `json:"processKey"`
}

var CheckProcessHealth = AppsRpcMethod[CheckProcessHealthInput, map[string]any]{
	Name: "CheckProcessHealth",
	Run: func(req RpcRequest[CheckProcessHealthInput]) (map[string]any, *HttpError) {
		id := process.ProcessId{
			Key:          req.Data.ProcessKey,
			Namespace:    process.NamespaceExtensionSpawned,
			NamespaceKey: req.Data.AppId,
		}

		isAlive := processManager.IsAlive(id)

		return map[string]any{
			"processKey": id,
			"isAlive":    isAlive,
		}, nil
	},
}
