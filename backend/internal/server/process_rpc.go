package server

import (
	"net/http"

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
		id := process.ProjectAppId(req.Data.AppId, req.Data.ProcessKey)

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
		id := process.ProjectAppId(req.Data.AppId, req.Data.ProcessKey)

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
		id := process.ProjectAppId(req.Data.AppId, req.Data.ProcessKey)

		isAlive := process.Manager.IsAlive(id)

		return map[string]any{
			"processKey": id,
			"isAlive":    isAlive,
		}, nil
	},
}

var ReadAppProcessLogs = Stream[CheckProcessHealthInput, string]{
	Name: "ReadAppProcessLogs",
	Run: func(req *StreamRequest[CheckProcessHealthInput, string]) error {
		input, err := req.ParseInput()
		if err != nil {
			return err
		}

		id := pubsub.AppProcessLogs(input.AppId, input.ProcessKey)

		subscription := make(chan string)
		if err := pubsub.Topics.Subscribe(id, subscription); err != nil {
			return err
		}
		defer pubsub.Topics.Unsubscribe(id, subscription)

		for {
			select {
			case s, ok := <-subscription:
				if !ok {
					// Channel is closed
					return nil
				}

				req.Send(s)

			case <-req.Context.Done():
				return nil
			}
		}
	},
}
