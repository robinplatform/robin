package server

import (
	"fmt"
	"net/http"

	"robinplatform.dev/internal/compile"
)

type GetAppSettingsByIdInput struct {
	AppId string `json:"appId"`
}

var GetAppSettingsById = RpcMethod[GetAppSettingsByIdInput, map[string]any]{
	Name: "GetAppSettingsById",
	Run: func(req RpcRequest[GetAppSettingsByIdInput]) (map[string]any, *HttpError) {
		app, err := compile.LoadRobinAppById(req.Data.AppId)
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to load app by id %s: %s", req.Data.AppId, err),
			}
		}

		settings, err := app.GetSettings()
		if err != nil {
			return nil, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to get settings for app %s: %s", req.Data.AppId, err),
			}
		}

		if settings == nil {
			settings = map[string]any{}
		}
		return settings, nil
	},
}

type UpdateAppSettingsInput struct {
	AppId    string         `json:"appId"`
	Settings map[string]any `json:"settings"`
}

var UpdateAppSettings = RpcMethod[UpdateAppSettingsInput, struct{}]{
	Name: "UpdateAppSettings",
	Run: func(req RpcRequest[UpdateAppSettingsInput]) (struct{}, *HttpError) {
		app, err := compile.LoadRobinAppById(req.Data.AppId)
		if err != nil {
			return struct{}{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to load app by id %s: %s", req.Data.AppId, err),
			}
		}

		err = app.UpdateSettings(req.Data.Settings)
		if err != nil {
			return struct{}{}, &HttpError{
				StatusCode: http.StatusInternalServerError,
				Message:    fmt.Sprintf("Failed to update settings for app %s: %s", req.Data.AppId, err),
			}
		}

		return struct{}{}, nil
	},
}
