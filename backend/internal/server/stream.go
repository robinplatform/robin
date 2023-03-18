package server

import (
	"context"
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

	// Run implements the actual method.
	Run func(req StreamRequest[Input, Output]) error
}

// TODO: Add context stuffs so that requests can be cancelled
type StreamRequest[Input any, Output any] streamRequest
type streamRequest struct {
	Method  string
	Id      string
	Context context.Context

	// Server is the instance serving the request
	Server *Server

	// Initial input to the stream
	RawInput []byte

	// The channel this stream request outputs to
	output chan<- socketMessageOut

	// Cancel function for the context
	cancel func()
}

func (s *StreamRequest[Input, _]) ParseInput() (Input, error) {
	var input Input
	err := json.Unmarshal(s.RawInput, &input)
	if err != nil {
		logger.Debug("RPC stream method failed to parse input", log.Ctx{
			"method":     s.Method,
			"id":         s.Id,
			"inputBytes": s.RawInput,
		})
	}

	return input, err
}

func (s *StreamRequest[_, Output]) Send(o Output) {
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
	Id string `json:"id,omitempty"`

	// TODO: This should probably be an enum type
	Kind   string `json:"kind"`
	Method string `json:"method,omitempty"`
	Err    string `json:"error,omitempty"`
	Data   any    `json:"data,omitempty"`
}

type handler func(req streamRequest) error
type RpcWebsocket struct {
	handlers map[string]handler
}

func (ws *RpcWebsocket) WebsocketHandler(server *Server) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		inFlightRequests := make(map[string]streamRequest)

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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		outputChannel := make(chan socketMessageOut)

		// This goroutine does the job of writing to the socket, because the socket
		// cannot be written to concurrently.
		go func() {
		WriteLoop:
			for {
				select {
				case message := <-outputChannel:

					if err := conn.WriteJSON(message); err != nil {
						logger.Debug("Failed to write JSON to connection", log.Ctx{
							"method": message.Method,
							"id":     message.Id,
							"error":  err,
						})
					}
				case <-ctx.Done():
					break WriteLoop
				}
			}
		}()

	MessageLoop:
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				// This also happens when the connection closes
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
					Err:  "failed to parse JSON",
				}
				continue MessageLoop
			}

			errMessage := socketMessageOut{
				Method: input.Method,
				Kind:   "error",
				Id:     input.Id,
			}

			switch input.Kind {
			case "call":
				method, ok := ws.handlers[input.Method]
				if !ok {
					logger.Debug("RPC websocket got invalid value for 'method'", log.Ctx{
						"method":  input.Method,
						"message": string(message),
					})

					errMessage.Err = "invalid value for 'method'"
					outputChannel <- errMessage
					continue MessageLoop
				}

				if input.Id == "" {
					errMessage.Err = "'id' field was empty"
					outputChannel <- errMessage
				}

				if _, foundPrevious := inFlightRequests[input.Id]; foundPrevious {
					errMessage.Err = "'id' field used previous ID value"
					outputChannel <- errMessage
				}

				mCtx, mCancel := context.WithCancel(ctx)
				req := streamRequest{
					Method:   input.Method,
					Id:       input.Id,
					Server:   server,
					RawInput: input.Data,
					output:   outputChannel,
					Context:  mCtx,
					cancel:   mCancel,
				}
				go runMethod(method, req)
				inFlightRequests[req.Id] = req

			case "cancel":
				req, found := inFlightRequests[input.Id]
				if !found {
					errMessage.Err = "'id' not found"
					outputChannel <- errMessage
					continue MessageLoop
				}

				req.cancel()
				delete(inFlightRequests, input.Id)

			default:
				logger.Debug("RPC websocket got invalid value for 'kind'", log.Ctx{
					"kind":    input.Kind,
					"message": string(message),
				})

				errMessage.Err = "invalid value for 'kind'"
				outputChannel <- errMessage
			}
		}
	}
}

func runMethod(method handler, rawReq streamRequest) {
	logger.Debug("starting up RPC stream", log.Ctx{
		"method": rawReq.Method,
		"id":     rawReq.Id,
	})

	rawReq.output <- socketMessageOut{
		Id:     rawReq.Id,
		Method: rawReq.Method,
		Kind:   "methodStarted",
	}

	if err := method(rawReq); err != nil {
		rawReq.output <- socketMessageOut{
			Id:     rawReq.Id,
			Method: rawReq.Method,
			Kind:   "error",
			Err:    err.Error(),
		}

		return
	}

	rawReq.output <- socketMessageOut{
		Id:     rawReq.Id,
		Method: rawReq.Method,
		Kind:   "methodDone",
	}
}

func (method *Stream[Input, Output]) handler(rawReq streamRequest) error {
	req := StreamRequest[Input, Output](rawReq)
	return method.Run(req)
}

func (method *Stream[Input, Output]) Register(ws *RpcWebsocket) error {
	_, ok := ws.handlers[method.Name]
	if ok {
		return fmt.Errorf("multiple streams registered to the same name")
	}

	if ws.handlers == nil {
		ws.handlers = make(map[string]handler)
	}
	ws.handlers[method.Name] = method.handler

	return nil
}
