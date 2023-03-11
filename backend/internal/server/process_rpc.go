package server

import (
	"fmt"
	"net/http"

	"robinplatform.dev/internal/process"
)

type StartProcessForAppInput struct {
	AppId      string `json:"appId"`
	ProcessKey string `json:"processKey"`
	Command    string `json:"command"`
}

var StartProcessForApp = AppsRpcMethod[StartProcessForAppInput, map[string]any]{
	Name: "StartProcessForApp",
	Run: func(req RpcRequest[StartProcessForAppInput]) (map[string]any, *HttpError) {
		processConfig := process.ProcessConfig{
			Command: req.Data.Command,
			Id: process.ProcessId{
				Kind:   process.KindAppSpawned,
				Source: req.Data.AppId,
				Key:    req.Data.ProcessKey,
			},
		}

		proc, err := process.Manager.Spawn(processConfig)
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
			Key:    req.Data.ProcessKey,
			Kind:   process.KindAppSpawned,
			Source: req.Data.AppId,
		}

		isAlive := process.Manager.IsAlive(id)

		return map[string]any{
			"processKey": id,
			"isAlive":    isAlive,
		}, nil
	},
}
