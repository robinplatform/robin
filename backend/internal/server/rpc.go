package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type HttpError struct {
	StatusCode int
	Message    string
}

var ErrSkipResponse = HttpError{StatusCode: 0, Message: "skip response"}

func Errorf(statusCode int, format string, args ...interface{}) *HttpError {
	return &HttpError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf(format, args...),
	}
}

type RpcRequest[Input any] struct {
	// Data is the input sent by the client
	Data Input

	// Request is the raw HTTP request
	Request *http.Request

	// Response is the raw HTTP response
	Response http.ResponseWriter

	// Server is the instance serving the request
	Server *Server
}

type RpcMethod[Input any, Output any] struct {
	// Name of the method, used by the client to call it
	Name string

	// SkipInputParsing skips parsing the input, and passes the zero value
	// of the input to the handler.
	SkipInputParsing bool

	// Run implements the actual method. It must always return the same shape,
	// and it must be a struct. The error must be of type *HttpError, and therefore
	// contain a reasonable HTTP status code.
	Run func(req RpcRequest[Input]) (Output, *HttpError)
}

func sendJson(req *http.Request, res http.ResponseWriter, statusCode int, data interface{}) {
	res.Header().Set("Content-Type", "application/json")

	var buf []byte
	var err error

	if userAgent, ok := req.Header["User-Agent"]; ok && len(userAgent) > 0 && strings.HasPrefix(userAgent[0], "curl/") {
		buf, err = json.MarshalIndent(data, "", "\t")
	} else {
		buf, err = json.Marshal(data)
	}

	if err != nil {
		if statusCode == 200 {
			statusCode = http.StatusInternalServerError
		}
		buf = []byte(fmt.Sprintf(`{"type": "error", "error": %q}`, err.Error()))
	}

	res.WriteHeader(statusCode)
	res.Write(buf)
}

func (method *RpcMethod[Input, Output]) Register(server *Server, router RouterGroup) {
	router.Handle("POST", "/"+method.Name, func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		defer func() {
			if err := recover(); err != nil {
				sendJson(req, res, http.StatusInternalServerError, map[string]any{"type": "error", "error": fmt.Sprintf("%s", err)})
			}
		}()

		var input Input

		if !method.SkipInputParsing {
			buf, err := io.ReadAll(req.Body)
			if err != nil {
				sendJson(req, res, http.StatusInternalServerError, map[string]any{"type": "error", "error": err.Error()})
				return
			}

			if err := json.Unmarshal(buf, &input); err != nil {
				sendJson(req, res, http.StatusBadRequest, map[string]any{"type": "error", "error": err.Error()})
				return
			}
		}

		result, httpError := method.Run(RpcRequest[Input]{
			Data:     input,
			Request:  req,
			Response: res,
			Server:   server,
		})
		if httpError != nil && *httpError == ErrSkipResponse {
			// do nothing
		} else if httpError != nil {
			sendJson(req, res, httpError.StatusCode, map[string]any{"type": "error", "error": httpError.Message})
		} else {
			sendJson(req, res, 200, result)
		}
	})
}
