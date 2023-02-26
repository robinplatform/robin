package server

import (
	"time"
)

type Heartbeat struct {
	Ok bool `json:"ok"`
}

var GetHeartbeat = Stream[struct{}, struct{}, Heartbeat]{
	Name:             "GetHeartbeat",
	SkipInputParsing: true,
	Run: func(_ struct{}, output chan<- Heartbeat) error {
		for {
			output <- Heartbeat{Ok: true}
			time.Sleep(1 * time.Second)
		}
	},
}
