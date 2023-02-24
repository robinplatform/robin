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
	Run: func(_ struct{}) ([]compile.RobinAppConfig, *rpc.HttpError) {
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
