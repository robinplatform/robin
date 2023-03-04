package server

import (
	"net/http"
	"runtime"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/project"
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

var GetConfig = InternalRpcMethod[struct{}, project.RobinConfig]{
	Name:             "GetConfig",
	SkipInputParsing: true,
	Run: func(_ RpcRequest[struct{}]) (project.RobinConfig, *HttpError) {
		robinConfig, err := project.LoadProjectConfig()
		if err != nil {
			return robinConfig, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}
		return robinConfig, nil
	},
}

var UpdateConfig = InternalRpcMethod[project.RobinConfig, struct{}]{
	Name: "UpdateConfig",
	Run: func(c RpcRequest[project.RobinConfig]) (struct{}, *HttpError) {
		var empty struct{}
		if err := project.UpdateProjectConfig(c.Data); err != nil {
			return empty, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}

		return empty, nil
	},
}
