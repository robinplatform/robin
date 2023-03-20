package server

import (
	"fmt"
	"net/http"

	"robinplatform.dev/internal/process"
)

type StartProcessForAppInput struct {
	AppId      string   `json:"appId"`
	ProcessKey string   `json:"processKey"`
	Command    string   `json:"command"`
	Args       []string `json:"args"`
}

var StartProcessForApp = AppsRpcMethod[StartProcessForAppInput, map[string]any]{
	Name: "StartProcess",
	Run: func(req RpcRequest[StartProcessForAppInput]) (map[string]any, *HttpError) {
		id, err := process.NewId(req.Data.AppId, req.Data.ProcessKey)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("invalid ID for spawning new process: %s", err),
			}
		}

		processConfig := process.ProcessConfig{
			Command: req.Data.Command,
			Args:    req.Data.Args,
			Id:      id,
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

type StopProcessForAppInput struct {
	AppId      string `json:"appId"`
	ProcessKey string `json:"processKey"`
}

var StopProcessForApp = AppsRpcMethod[StartProcessForAppInput, map[string]any]{
	Name: "StopProcess",
	Run: func(req RpcRequest[StartProcessForAppInput]) (map[string]any, *HttpError) {
		id, err := process.NewId(req.Data.AppId, req.Data.ProcessKey)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("invalid ID for stopping process: %s", err),
			}
		}

		err = process.Manager.Remove(id)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to kill process %s: %s", req.Data.AppId, err),
			}
		}

		return map[string]any{}, nil
	},
}

type CheckProcessHealthInput struct {
	AppId      string `json:"appId"`
	ProcessKey string `json:"processKey"`
}

var CheckProcessHealth = AppsRpcMethod[CheckProcessHealthInput, map[string]any]{
	Name: "CheckProcessHealth",
	Run: func(req RpcRequest[CheckProcessHealthInput]) (map[string]any, *HttpError) {
		id, err := process.NewId(req.Data.AppId, req.Data.ProcessKey)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("invalid ID for stopping process: %s", err),
			}
		}

		isAlive := process.Manager.IsAlive(id)

		return map[string]any{
			"processKey": id,
			"isAlive":    isAlive,
		}, nil
	},
}
