package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type HttpError struct {
	StatusCode int
	Message    string
}

func Errorf(statusCode int, format string, args ...interface{}) *HttpError {
	return &HttpError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf(format, args...),
	}
}

type RpcRequest[Input any] struct {
	// Data is the input sent by the client
	Data Input

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

func sendJson(c *gin.Context, statusCode int, data interface{}) {
	if strings.HasPrefix(c.GetHeader("User-Agent"), "curl/") {
		buf, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(statusCode, "application/json", buf)
	} else {
		c.JSON(statusCode, data)
	}
}

func (method *RpcMethod[Input, Output]) Register(server *Server, router *gin.RouterGroup) {
	router.POST("/"+method.Name, func(c *gin.Context) {
		var input Input

		if !method.SkipInputParsing {
			if err := c.ShouldBindJSON(&input); err != nil {
				sendJson(c, http.StatusBadRequest, gin.H{"type": "error", "error": err.Error()})
				return
			}
		}

		result, httpError := method.Run(RpcRequest[Input]{
			Data:   input,
			Server: server,
		})
		if httpError != nil {
			sendJson(c, httpError.StatusCode, gin.H{"type": "error", "error": httpError.Message})
		} else {
			sendJson(c, 200, result)
		}
	})
}
