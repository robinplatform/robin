package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"robinplatform.dev/internal/log"
)

type Stream[Input any, Output any] struct {
	// Name of the method, used by the client to call it
	Name string

	// SkipInputParsing skips parsing the input, and passes the zero value
	// of the input to the handler.
	SkipInputParsing bool

	// Run implements the actual method. It must always return the same shape,
	// and it must be a struct. The error must be of type *HttpError, and therefore
	// contain a reasonable HTTP status code.
	Run func(input StreamRequest[Input, Output]) error
}

type StreamRequest[Input any, Output any] struct {
	// Server is the instance serving the request
	Server *Server

	// Initial input to the stream
	Input Input

	Output chan<- Output
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
	Id string

	Kind   string
	Method string
	Err    error
	Data   any
}

type socketMessageOutJSON struct {
	Id     string `json:"id,omitempty"`
	Kind   string `json:"kind"`
	Method string `json:"method,omitempty"`
	Err    string `json:"error,omitempty"`
	Data   any    `json:"data,omitempty"`
}

type RpcWebsocket struct {
	handlers map[string]func(id string, req StreamRequest[[]byte, socketMessageOut])
}

var invalidInputMessage = socketMessageOut{
	Kind: "error",
	Err:  fmt.Errorf("invalid input"),
}

func (ws *RpcWebsocket) WebsocketHandler(server *Server) httprouter.Handle {
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

		outputChannel := make(chan socketMessageOut)

		go func() {
			for message := range outputChannel {
				o := socketMessageOutJSON{
					Id:     message.Id,
					Kind:   message.Kind,
					Method: message.Method,
					Data:   message.Data,
				}

				if message.Err != nil {
					o.Err = message.Err.Error()
				}

				if err := conn.WriteJSON(o); err != nil {
					logger.Debug("Failed to write JSON to connection", log.Ctx{
						"method": o.Method,
						"id":     o.Id,
						"error":  err,
					})
				}
			}
		}()

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
				outputChannel <- invalidInputMessage
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
					outputChannel <- invalidInputMessage
					continue
				}

				req := StreamRequest[[]byte, socketMessageOut]{
					Server: server,
					Input:  input.Data,
					Output: outputChannel,
				}
				go method(input.Id, req)

				continue

			default:
				logger.Debug("RPC websocket got invalid value for 'kind'", log.Ctx{
					"kind":    input.Kind,
					"message": string(message),
				})
				outputChannel <- invalidInputMessage
				continue
			}
		}
	}
}

func (method *Stream[Input, Output]) handleConn(id string, rawReq StreamRequest[[]byte, socketMessageOut]) {
	var input Input
	if err := json.Unmarshal(rawReq.Input, &input); err != nil {
		logger.Debug("RPC stream method failed to parse input", log.Ctx{
			"method":     method.Name,
			"id":         id,
			"inputBytes": rawReq.Input,
		})

		rawReq.Output <- socketMessageOut{
			Id:     id,
			Method: method.Name,
			Kind:   "error",
			Err:    err,
		}
		return
	}

	logger.Debug("starting up RPC stream", log.Ctx{
		"method": method.Name,
		"id":     id,
	})

	rawReq.Output <- socketMessageOut{
		Id:     id,
		Method: method.Name,
		Kind:   "methodStarted",
	}

	outputChannel := make(chan Output, 8)

	go func() {
		for output := range outputChannel {
			rawReq.Output <- socketMessageOut{
				Method: method.Name,
				Id:     id,
				Kind:   "methodOutput",
				Data:   output,
			}
		}
	}()

	req := StreamRequest[Input, Output]{
		Server: rawReq.Server,
		Input:  input,
		Output: outputChannel,
	}

	if err := method.Run(req); err != nil {
		rawReq.Output <- socketMessageOut{
			Id:     id,
			Method: method.Name,
			Kind:   "methodStarted",
			Err:    err,
		}

		return
	}

	rawReq.Output <- socketMessageOut{
		Id:     id,
		Method: method.Name,
		Kind:   "methodDone",
	}
}

func (method *Stream[Input, Output]) Register(ws *RpcWebsocket) error {
	_, ok := ws.handlers[method.Name]
	if ok {
		return fmt.Errorf("multiple streams registered to the same name")
	}

	if ws.handlers == nil {
		ws.handlers = make(map[string]func(id string, req StreamRequest[[]byte, socketMessageOut]))
	}
	ws.handlers[method.Name] = method.handleConn

	return nil
}
