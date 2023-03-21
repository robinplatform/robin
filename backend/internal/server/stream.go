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
	Run func(req *StreamRequest[Input, Output]) error
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

func (req *streamRequest) SendRaw(kind string, data any) {
	req.output <- socketMessageOut{
		Method: req.Method,
		Id:     req.Id,
		Kind:   kind,
		Data:   data,
	}
}

// The idea behind using `ParseInput` instead of something with generics is twofold:
//  1. It reduces the amount of generic code necessary to implement certain things in
//     this file. - Specifically, some of the hooks for handling code are now very very short,
//     and don't need to be duplicated for each generic instantiation
//  2. It allows more flexibility - This is a bit weak, but it does technically allow the user
//     to make some custom parsing error handling, or to parse the input in a custom way.
//
// Using ParseInput also comes with downsides, mostly in usability. It's unclear if the tradeoff is
// worth it yet, but we will see.
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

// This code uses `Send` instead of a channel to try to reduce the number
// of channels/goroutines that need to run at any one time. Otherwise there'd
// need to at least be one goroutine per-stream, and often pubsub uses channels
// with *very slight* caveats, which require an additional goroutine/thread intercepting
// the subscription channel and piping into the stream's channel.
//
// With a send function, we at least eliminate some complexity in the implementation, and also allow
// the user to decide themselves what parallelism paradigms they would like to use.
func (s *StreamRequest[_, Output]) Send(o Output) {
	(*streamRequest)(s).SendRaw("methodOutput", o)
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

type handler func(req *streamRequest) error
type RpcWebsocket struct {
	handlers map[string]handler
}

func (ws *RpcWebsocket) WebsocketHandler(server *Server) httprouter.Handle {
	return func(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
		inFlightRequests := make(map[string]*streamRequest)

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

			req, found := inFlightRequests[input.Id]
			if req == nil {
				mCtx, mCancel := context.WithCancel(ctx)
				req = &streamRequest{
					Method:   input.Method,
					Id:       input.Id,
					Server:   server,
					RawInput: input.Data,
					output:   outputChannel,
					Context:  mCtx,
					cancel:   mCancel,
				}

			}

			switch input.Kind {
			case "call":
				method, ok := ws.handlers[req.Method]
				if !ok {
					logger.Debug("RPC websocket got invalid value for 'method'", log.Ctx{
						"method":  input.Method,
						"message": string(message),
					})

					req.SendRaw("error", "invalid value for 'method'")
					continue MessageLoop
				}

				if req.Id == "" {
					req.SendRaw("error", "'id' field was empty")
					continue MessageLoop
				}

				if found {
					req.SendRaw("error", "'id' field used previous ID value")
					continue MessageLoop
				}

				go runMethod(method, req)
				inFlightRequests[req.Id] = req

			case "cancel":
				if !found {
					req.SendRaw("error", "'id' not found")
					continue MessageLoop
				}

				req.cancel()
				delete(inFlightRequests, req.Id)

			default:
				logger.Debug("RPC websocket got invalid value for 'kind'", log.Ctx{
					"kind":    input.Kind,
					"message": string(message),
				})

				req.SendRaw("error", "invalid value for 'kind'")
			}
		}
	}
}

func runMethod(method handler, rawReq *streamRequest) {
	logger.Debug("starting up RPC stream", log.Ctx{
		"method": rawReq.Method,
		"id":     rawReq.Id,
	})

	rawReq.SendRaw("methodStarted", nil)

	if err := method(rawReq); err != nil {
		rawReq.SendRaw("error", err.Error())

		return
	}

	rawReq.SendRaw("methodDone", nil)
}

func (method *Stream[Input, Output]) handler(rawReq *streamRequest) error {
	req := (*StreamRequest[Input, Output])(rawReq)
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
