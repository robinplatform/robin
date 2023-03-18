package server

import (
	"time"
)

type Heartbeat struct {
	Ok bool `json:"ok"`
}

var GetHeartbeat = Stream[struct{}, Heartbeat]{
	Name:             "GetHeartbeat",
	SkipInputParsing: true,
	Run: func(req StreamRequest[struct{}, Heartbeat]) error {
		for {
			req.Output <- Heartbeat{Ok: true}
			time.Sleep(1 * time.Second)
		}
	},
}
