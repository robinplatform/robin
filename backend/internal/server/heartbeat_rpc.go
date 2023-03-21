package server

import (
	"time"
)

type Heartbeat struct {
	Ok bool `json:"ok"`
}

var GetHeartbeat = Stream[struct{}, Heartbeat]{
	Name: "GetHeartbeat",
	Run: func(req *StreamRequest[struct{}, Heartbeat]) error {
		for {
			req.Send(Heartbeat{Ok: true})
			time.Sleep(1 * time.Second)
		}
	},
}
