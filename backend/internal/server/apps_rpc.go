package server

import (
	"net/http"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/rpc"
)

var GetApps = rpc.Method[struct{}, []config.RobinAppConfig]{
	Name:             "GetApps",
	SkipInputParsing: true,
	Run: func(_ struct{}) ([]config.RobinAppConfig, *rpc.HttpError) {
		appConfig, err := config.LoadRobinProjectConfig()
		if err != nil {
			return nil, &rpc.HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    err.Error(),
			}
		}
		return appConfig.Apps, nil
	},
}
