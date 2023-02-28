package server

import (
	"net/http"
	"runtime"

	"robinplatform.dev/internal/config"
)

type GetVersionResponse struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

var GetVersion = InternalRpcMethod[struct{}, GetVersionResponse]{
	Name:             "GetVersion",
	SkipInputParsing: true,
	Run: func(_ RpcRequest[struct{}]) (GetVersionResponse, *HttpError) {
		return GetVersionResponse{
			Version: config.GetRobinVersion(),
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
		}, nil
	},
}

var GetConfig = InternalRpcMethod[struct{}, config.RobinConfig]{
	Name:             "GetConfig",
	SkipInputParsing: true,
	Run: func(_ RpcRequest[struct{}]) (config.RobinConfig, *HttpError) {
		robinConfig, err := config.LoadProjectConfig()
		if err != nil {
			return robinConfig, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}
		return robinConfig, nil
	},
}

var UpdateConfig = InternalRpcMethod[config.RobinConfig, struct{}]{
	Name: "UpdateConfig",
	Run: func(c RpcRequest[config.RobinConfig]) (struct{}, *HttpError) {
		var empty struct{}
		if err := config.UpdateProjectConfig(c.Data); err != nil {
			return empty, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}

		return empty, nil
	},
}
