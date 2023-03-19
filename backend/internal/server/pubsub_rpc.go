package server

import (
	"robinplatform.dev/internal/pubsub"
)

type GetTopicsInput struct {
}

var GetTopics = AppsRpcMethod[GetTopicsInput, []pubsub.TopicInfo]{
	Name: "GetTopics",
	Run: func(req RpcRequest[GetTopicsInput]) ([]pubsub.TopicInfo, *HttpError) {
		names := pubsub.Topics.GetTopicInfo()
		return names, nil
	},
}

type SubscribeTopicInput struct {
	Id pubsub.TopicId `json:"id"`
}

var SubscribeTopic = Stream[SubscribeTopicInput, string]{
	Name: "SubscribeTopic",
	Run: func(req StreamRequest[SubscribeTopicInput, string]) error {
		input, err := req.ParseInput()
		if err != nil {
			return err
		}

		subscription := make(chan string)
		if err := pubsub.Topics.Subscribe(input.Id, subscription); err != nil {
			return err
		}
		defer pubsub.Topics.Unsubscribe(input.Id, subscription)

		for {
			select {
			case s, ok := <-subscription:
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
