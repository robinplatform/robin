package server

import (
	"context"
	"fmt"
	"net/http"

	"robinplatform.dev/internal/compile"
)

var GetApps = RpcMethod[struct{}, []compile.RobinAppConfig]{
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

var RunAppMethod = RpcMethod[RunAppMethodInput, any]{
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

		if err := app.StartServer(); err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to start app server: %s", err),
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
