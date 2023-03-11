package server

import "robinplatform.dev/internal/process"

type ListProcessesInput struct {
}

var ListProcesses = InternalRpcMethod[ListProcessesInput, []process.Process]{
	Name: "ListProcesses",
	Run: func(c RpcRequest[ListProcessesInput]) ([]process.Process, *HttpError) {
		data := process.Manager.CopyOutData()
		return data, nil
	},
}
