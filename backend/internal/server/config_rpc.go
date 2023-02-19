package server

import (
	"net/http"
	"runtime"

	"robin.dev/internal/config"
	"robin.dev/internal/rpc"
)

type GetVersionResponse struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

var GetVersion = rpc.Method[struct{}, GetVersionResponse]{
	Name:             "GetVersion",
	SkipInputParsing: true,
	Run: func(_ struct{}) (GetVersionResponse, *rpc.HttpError) {
		return GetVersionResponse{
			Version: config.GetRobinVersion(),
			OS:      runtime.GOOS,
			Arch:    runtime.GOARCH,
		}, nil
	},
}

var GetConfig = rpc.Method[struct{}, config.RobinConfig]{
	Name:             "GetConfig",
	SkipInputParsing: true,
	Run: func(_ struct{}) (config.RobinConfig, *rpc.HttpError) {
		robinConfig, err := config.LoadProjectConfig()
		if err != nil {
			return robinConfig, &rpc.HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}
		return robinConfig, nil
	},
}

var UpdateConfig = rpc.Method[config.RobinConfig, struct{}]{
	Name: "UpdateConfig",
	Run: func(c config.RobinConfig) (struct{}, *rpc.HttpError) {
		var empty struct{}
		if err := config.UpdateProjectConfig(c); err != nil {
			return empty, &rpc.HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}

		return empty, nil
	},
}
