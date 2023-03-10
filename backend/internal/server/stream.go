package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"robinplatform.dev/internal/log"
)

type Stream[Context any, Input any, Output any] struct {
	// Name of the method, used by the client to call it
	Name string

	// SkipInputParsing skips parsing the input, and passes the zero value
	// of the input to the handler.
	SkipInputParsing bool

	// Run implements the actual method. It must always return the same shape,
	// and it must be a struct. The error must be of type *HttpError, and therefore
	// contain a reasonable HTTP status code.
	Run func(input Input, output chan<- Output) error
}

type socketMessage struct {
	// for now, the ID is used exclusively by the client. If
	// there's some kind of message-passing system later on,
	// this ID will probably need some more stringent requirements,
	// but for now, this works fine.
	Id string

	Kind   string
	Method string
	Data   json.RawMessage
}

type socketMessageOut struct {
	// for now, the ID is used exclusively by the client. If
	// there's some kind of message-passing system later on,
	// this ID will probably need some more stringent requirements,
	// but for now, this works fine.
	Id string `json:"id"`

	Kind   string `json:"kind"`
	Method string `json:"method,omitempty"`
	Data   any    `json:"data"`
}

type RpcWebsocket struct {
	handlers map[string]func(*websocket.Conn, string, []byte)
}

var invalidInputMessage []byte = []byte(`{"kind":"error","error": "invalid input"}`)

func (ws *RpcWebsocket) WebsocketHandler() httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(res, req, nil)
		if err != nil {
			logger.Err("Failed to upgrade websocket", log.Ctx{
				"error": err,
			})
			res.WriteHeader(http.StatusInternalServerError)
			res.Write([]byte(fmt.Sprintf(`{"error": %q}`, err.Error())))
			return
		}

		defer conn.Close()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var input socketMessage
			if err := json.Unmarshal(message, &input); err != nil {
				logger.Err("RPC websocket failed to parse", log.Ctx{
					"error":   err,
					"message": message,
				})
				conn.WriteMessage(websocket.TextMessage, invalidInputMessage)
				continue
			}

			switch input.Kind {
			case "call":
				method, ok := ws.handlers[input.Method]
				if !ok {
					logger.Debug("RPC websocket got invalid value for 'method'", log.Ctx{
						"method":  input.Method,
						"message": string(message),
					})
					conn.WriteMessage(websocket.TextMessage, invalidInputMessage)
					continue
				}

				go method(conn, input.Id, input.Data)

				continue

			default:
				logger.Debug("RPC websocket got invalid value for 'kind'", log.Ctx{
					"kind":    input.Kind,
					"message": string(message),
				})
				conn.WriteMessage(websocket.TextMessage, invalidInputMessage)
				continue
			}
		}
	}
}

func (method *Stream[Context, Input, Output]) handleConn(conn *websocket.Conn, id string, inputBytes []byte) {
	var input Input
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		logger.Debug("RPC stream method failed to parse input", log.Ctx{
			"method":     method.Name,
			"id":         id,
			"inputBytes": inputBytes,
		})

		conn.WriteJSON(map[string]any{
			"kind":  "error",
			"error": err.Error(),
		})

		return
	}

	logger.Debug("starting up RPC stream", log.Ctx{
		"method": method.Name,
		"id":     id,
	})

	conn.WriteJSON(map[string]any{
		"kind":   "methodStarted",
		"method": method.Name,
		"id":     id,
	})

	outputChannel := make(chan Output, 8)

	go func() {
		for {
			output, ok := <-outputChannel
			if !ok {
				return
			}

			message := socketMessageOut{
				Method: method.Name,
				Id:     id,
				Kind:   "methodOutput",
				Data:   output,
			}

			if err := conn.WriteJSON(message); err != nil {
				logger.Debug("Failed to write JSON to connection", log.Ctx{
					"method": method.Name,
					"id":     id,
					// "output": output,
					"error": err,
				})

				return
			}
		}
	}()

	if err := method.Run(input, outputChannel); err != nil {
		out := map[string]any{
			"kind":   "error",
			"method": method.Name,
			"id":     id,
			"error":  err.Error(),
		}
		if jsonErr := conn.WriteJSON(out); jsonErr != nil {
			logger.Debug("Failed to write JSON error to connection", log.Ctx{
				"method":  method.Name,
				"id":      id,
				"error":   err,
				"jsonErr": jsonErr,
			})
		}

		return
	}

	out := map[string]any{
		"kind": "methodDone",
		"id":   id,
	}
	if jsonErr := conn.WriteJSON(out); jsonErr != nil {
		logger.Debug("Failed to write JSON done message to connection", log.Ctx{
			"method":  method,
			"id":      id,
			"jsonErr": jsonErr,
		})
	}
}

func (method *Stream[Context, Input, Output]) Register(ctx Context, ws *RpcWebsocket) error {
	_, ok := ws.handlers[method.Name]
	if ok {
		return fmt.Errorf("multiple streams registered to the same name")
	}

	if ws.handlers == nil {
		ws.handlers = map[string]func(*websocket.Conn, string, []byte){}
	}
	ws.handlers[method.Name] = method.handleConn

	return nil
}

// Streaming RPC method support
// Frontend needs to be changed from socket.io to custom thing
// Get config
// Update config

// Extension helpers package
