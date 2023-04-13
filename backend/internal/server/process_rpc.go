package server

import (
	"robinplatform.dev/internal/process"
	"robinplatform.dev/internal/pubsub"
)

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
