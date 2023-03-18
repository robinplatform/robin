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

	// Run implements the actual method.
	Run func(req StreamRequest[Input, Output]) error
}

type StreamRequest[Input any, Output any] struct {
	Method string
	Id     string

	// Server is the instance serving the request
	Server *Server

	// Initial input to the stream
	Input Input

	// The channel this stream request outputs to
	output chan<- socketMessageOut
}

func (s *StreamRequest[Input, Output]) Send(o Output) {
	s.output <- socketMessageOut{
		Method: s.Method,
		Id:     s.Id,
		Kind:   "methodOutput",
		Data:   o,
	}
}

type socketMessageIn struct {
	// for now, the ID is used exclusively by the client. If
	// there's some kind of message-passing system later on,
	// this ID will probably need some more stringent requirements,
	// but for now, this works fine.
	Id string `json:"id"`

	Kind   string          `json:"kind"`
	Method string          `json:"method"`
	Data   json.RawMessage `json:"data"`
}

type socketMessageOut struct {
	// for now, the ID is used exclusively by the client. If
	// there's some kind of message-passing system later on,
	// this ID will probably need some more stringent requirements,
	// but for now, this works fine.
	Id string

	// TODO: This should probably be an enum type
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

type handler func(req StreamRequest[[]byte, any])
type RpcWebsocket struct {
	handlers map[string]handler
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

		// This goroutine does the job of writing to the socket, because the socket
		// cannot be written to concurrently.
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

			var input socketMessageIn
			if err := json.Unmarshal(message, &input); err != nil {
				logger.Err("RPC websocket failed to parse", log.Ctx{
					"error":   err,
					"message": message,
				})
				outputChannel <- socketMessageOut{
					Kind: "error",
					Err:  fmt.Errorf("failed to parse JSON"),
				}
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
					outputChannel <- socketMessageOut{
						Method: input.Method,
						Kind:   "error",
						Id:     input.Id,
						Err:    fmt.Errorf("invalid value for 'method'"),
					}
					continue
				}

				req := StreamRequest[[]byte, any]{
					Method: input.Method,
					Id:     input.Id,
					Server: server,
					Input:  input.Data,
					output: outputChannel,
				}
				go method(req)

				continue

			default:
				logger.Debug("RPC websocket got invalid value for 'kind'", log.Ctx{
					"kind":    input.Kind,
					"message": string(message),
				})
				outputChannel <- socketMessageOut{
					Id:     input.Id,
					Kind:   "error",
					Method: input.Method,
					Err:    fmt.Errorf("invalid value for 'kind'"),
				}
				continue
			}
		}
	}
}

func (method *Stream[Input, Output]) handleConn(rawReq StreamRequest[[]byte, any]) {
	var input Input
	if err := json.Unmarshal(rawReq.Input, &input); err != nil {
		logger.Debug("RPC stream method failed to parse input", log.Ctx{
			"method":     rawReq.Method,
			"id":         rawReq.Id,
			"inputBytes": rawReq.Input,
		})

		rawReq.output <- socketMessageOut{
			Id:     rawReq.Id,
			Method: rawReq.Method,
			Kind:   "error",
			Err:    err,
		}
		return
	}

	logger.Debug("starting up RPC stream", log.Ctx{
		"method": rawReq.Method,
		"id":     rawReq.Id,
	})

	rawReq.output <- socketMessageOut{
		Id:     rawReq.Id,
		Method: rawReq.Method,
		Kind:   "methodStarted",
	}

	req := StreamRequest[Input, Output]{
		Id:     rawReq.Id,
		Method: rawReq.Method,
		Server: rawReq.Server,
		Input:  input,
		output: rawReq.output,
	}

	if err := method.Run(req); err != nil {
		rawReq.output <- socketMessageOut{
			Id:     rawReq.Id,
			Method: rawReq.Method,
			Kind:   "error",
			Err:    err,
		}

		return
	}

	rawReq.output <- socketMessageOut{
		Id:     rawReq.Id,
		Method: rawReq.Method,
		Kind:   "methodDone",
	}
}

func (method *Stream[Input, Output]) Register(ws *RpcWebsocket) error {
	_, ok := ws.handlers[method.Name]
	if ok {
		return fmt.Errorf("multiple streams registered to the same name")
	}

	if ws.handlers == nil {
		ws.handlers = make(map[string]handler)
	}
	ws.handlers[method.Name] = method.handleConn

	return nil
}
