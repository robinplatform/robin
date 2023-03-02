package server

import (
	"context"
	"fmt"
	"net/http"

	"robinplatform.dev/internal/compile"
)

type GetAppByIdInput struct {
	AppId string `json:"appId"`
}

var GetAppById = InternalRpcMethod[GetAppByIdInput, compile.RobinAppConfig]{
	Name: "GetAppById",
	Run: func(req RpcRequest[GetAppByIdInput]) (compile.RobinAppConfig, *HttpError) {
		app, err := compile.LoadRobinAppById(req.Data.AppId)
		if err != nil {
			return compile.RobinAppConfig{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to load app by id %s: %s", req.Data.AppId, err),
			}
		}
		return app, nil
	},
}

var GetApps = InternalRpcMethod[struct{}, []compile.RobinAppConfig]{
	Name:             "GetApps",
	SkipInputParsing: true,
	Run: func(_ RpcRequest[struct{}]) ([]compile.RobinAppConfig, *HttpError) {
		apps, err := compile.GetAllProjectApps()
		if err != nil {
			return nil, &HttpError{
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

var RunAppMethod = InternalRpcMethod[RunAppMethodInput, any]{
	Name: "RunAppMethod",
	Run: func(req RpcRequest[RunAppMethodInput]) (any, *HttpError) {
		_, err := compile.LoadRobinAppById(req.Data.AppId)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to load app by id %s: %s", req.Data.AppId, err),
			}
		}

		app, err := req.Server.compiler.GetApp(req.Data.AppId)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				// the error messages from GetApp() are already user-friendly
				Message: err.Error(),
			}
		}

		if !app.IsAlive() {
			if err := app.StartServer(); err != nil {
				return nil, &HttpError{
					StatusCode: http.StatusInternalServerError,
					Message:    fmt.Sprintf("Failed to start app server: %s", err),
				}
			}
		}

		result, err := app.Request(context.TODO(), "POST", "/api/RunAppMethod", map[string]any{
			"serverFile": req.Data.ServerFile,
			"methodName": req.Data.MethodName,
			"data":       req.Data.Data,
		})
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}

		return result, nil
	},
}

type RestartAppInput struct {
	AppId string `json:"appId"`
}

var RestartApp = InternalRpcMethod[RestartAppInput, struct{}]{
	Name: "RestartApp",
	Run: func(req RpcRequest[RestartAppInput]) (struct{}, *HttpError) {
		_, err := compile.LoadRobinAppById(req.Data.AppId)
		if err != nil {
			return struct{}{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to load app by id %s: %s", req.Data.AppId, err),
			}
		}

		app, err := req.Server.compiler.GetApp(req.Data.AppId)
		if err != nil {
			return struct{}{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				// the error messages from GetApp() are already user-friendly
				Message: err.Error(),
			}
		}

		if err := app.StopServer(); err != nil {
			return struct{}{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to stop app server: %s", err),
			}
		}

		if err := app.StartServer(); err != nil {
			return struct{}{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to restart app server: %s", err),
			}
		}

		return struct{}{}, nil
	},
}
