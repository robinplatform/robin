package server

import (
	"fmt"
	"net/http"

	"robinplatform.dev/internal/compile"
	"robinplatform.dev/internal/rpc"
)

var GetApps = rpc.Method[struct{}, []compile.RobinAppConfig]{
	Name:             "GetApps",
	SkipInputParsing: true,
	Run: func(_ rpc.RpcRequest[struct{}]) ([]compile.RobinAppConfig, *rpc.HttpError) {
		apps, err := compile.GetAllProjectApps()
		if err != nil {
			return nil, &rpc.HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("failed to get apps: %s", err),
			}
		}
		return apps, nil
	},
}

type RunAppMethodInput struct {
	AppId      string         `json:"appId"`
	ServerFile string         `json:"serverFile"`
	MethodName string         `json:"methodName"`
	Data       map[string]any `json:"data"`
}

// RunAppMethodOutput represents the result of RunAppMethod, and under the error
// condition MUST conform to the same shape of the generic error from the RPC router.
// This allows the client to have a single way to parse errors.
type RunAppMethodOutput struct {
	// Type should be either success or error
	Type string `json:"type"`
	// Error will contain an error message when type is error
	Error string `json:"error,omitempty"`
	// Result is the output received from the method
	Result map[string]any `json:"result"`
}

var RunAppMethod = rpc.Method[RunAppMethodInput, RunAppMethodOutput]{
	Name: "RunAppMethod",
	Run: func(req rpc.RpcRequest[RunAppMethodInput]) (RunAppMethodOutput, *rpc.HttpError) {
		fmt.Printf("input: %#v\n", req)

		_, err := compile.LoadRobinAppById(req.Data.AppId)
		if err != nil {
			return RunAppMethodOutput{}, &rpc.HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to load app by id %s: %s", req.Data.AppId, err),
			}
		}

		return RunAppMethodOutput{}, &rpc.HttpError{
			StatusCode: http.StatusInternalServerError,
			Message:    "not implemented yet",
		}
	},
}
