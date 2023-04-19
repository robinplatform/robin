package server

import (
	"net/http"

	"robinplatform.dev/internal/identity"
	"robinplatform.dev/internal/process"
	"robinplatform.dev/internal/pubsub"
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
		id := process.ProcessId{
			Category: identity.Category("app", req.Data.AppId),
			Key:      req.Data.ProcessKey,
		}

		processConfig := process.ProcessConfig{
			Command: req.Data.Command,
			Args:    req.Data.Args,
			Id:      id,
		}

		proc, err := process.Manager.Spawn(processConfig)
		if err != nil {
			return nil, Errorf(http.StatusInternalServerError, "Failed to spawn new process %s: %s", req.Data.AppId, err)
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
		id := process.ProcessId{
			Category: identity.Category("app", req.Data.AppId),
			Key:      req.Data.ProcessKey,
		}

		if err := process.Manager.Remove(id); err != nil {
			return nil, Errorf(http.StatusInternalServerError, "Failed to kill process %s: %s", req.Data.AppId, err)
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
		id := process.ProcessId{
			Category: identity.Category("app", req.Data.AppId),
			Key:      req.Data.ProcessKey,
		}

		isHealthy := process.Manager.CheckHealth(id)

		return map[string]any{
			"processKey": id,
			"isHealthy":  isHealthy,
		}, nil
	},
}

func PipeTopic[T any](topicId pubsub.TopicId, req *StreamRequest[T, any]) error {
	sub, err := pubsub.SubscribeAny(&pubsub.Topics, topicId)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case s, ok := <-sub.Out:
			if !ok {
				// Channel is closed
				return nil
			}

			var a any = &s
			req.Send(a)

		case <-req.Context.Done():
			return nil
		}
	}
}

type GetProcessLogsInput struct {
	ProcessId process.ProcessId `json:"processId"`
}

var GetProcessLogs = InternalRpcMethod[GetProcessLogsInput, process.LogFileResult]{
	Name: "GetProcessLogs",
	Run: func(req RpcRequest[GetProcessLogsInput]) (process.LogFileResult, *HttpError) {
		result, err := process.Manager.GetLogFile(req.Data.ProcessId)
		if err != nil {
			return process.LogFileResult{}, Errorf(500, "%s", err.Error())
		}

		return result, nil
	},
}
