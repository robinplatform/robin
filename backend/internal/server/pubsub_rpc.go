package server

import (
	"robinplatform.dev/internal/pubsub"
)

type GetTopicsInput struct {
}

var GetTopics = AppsRpcMethod[GetTopicsInput, map[string]pubsub.TopicInfo]{
	Name: "GetTopics",
	Run: func(req RpcRequest[GetTopicsInput]) (map[string]pubsub.TopicInfo, *HttpError) {
		names := pubsub.Topics.GetTopicInfo()
		return names, nil
	},
}

type SubscribeTopicInput struct {
	Id pubsub.TopicId `json:"id"`
}

var SubscribeTopic = Stream[SubscribeTopicInput, any]{
	Name: "SubscribeTopic",
	Run: func(req *StreamRequest[SubscribeTopicInput, any]) error {
		input, err := req.ParseInput()
		if err != nil {
			return err
		}

		sub, err := pubsub.SubscribeAny(&pubsub.Topics, input.Id)
		if err != nil {
			return err
		}
		defer sub.Unsubscribe()

		for {
			select {
			case s, ok := <-sub.Out:
				if !ok {
					// Channel is closed
					return nil
				}

				req.Send(s)

			case <-req.Context.Done():
				return nil
			}
		}
	},
}
