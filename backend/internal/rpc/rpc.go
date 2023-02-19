package rpc

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"robinplatform.dev/internal/log"
)

type HttpError struct {
	StatusCode int
	Message    string
}

// TODO: Add context modeled after rpc.Stream
type Method[Input any, Output any] struct {
	// Name of the method, used by the client to call it
	Name string

	// SkipInputParsing skips parsing the input, and passes the zero value
	// of the input to the handler.
	SkipInputParsing bool

	// Run implements the actual method. It must always return the same shape,
	// and it must be a struct. The error must be of type *HttpError, and therefore
	// contain a reasonable HTTP status code.
	Run func(input Input) (Output, *HttpError)
}

var logger log.Logger = log.New("rpc")

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

func (method *Method[Input, Output]) Register(router *gin.RouterGroup) {
	router.POST("/"+method.Name, func(c *gin.Context) {
		var input Input

		if !method.SkipInputParsing {
			if err := c.ShouldBindJSON(&input); err != nil {
				sendJson(c, http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		result, httpError := method.Run(input)
		if httpError != nil {
			sendJson(c, httpError.StatusCode, gin.H{"error": httpError.Message})
		} else {
			sendJson(c, 200, result)
		}
	})
}
