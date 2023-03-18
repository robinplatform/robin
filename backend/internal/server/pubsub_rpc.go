package server

import "robinplatform.dev/internal/pubsub"

type GetTopicsInput struct {
}

var GetTopics = AppsRpcMethod[GetTopicsInput, []pubsub.TopicId]{
	Name: "GetTopics",
	Run: func(req RpcRequest[GetTopicsInput]) ([]pubsub.TopicId, *HttpError) {
		names := pubsub.Topics.GetTopics()
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

		// TODO: this is sorta unnecessary, ideally it should be possible to just wait for the topic to close
		for s := range subscription {
			req.Send(s)
		}

		return nil
	},
}
