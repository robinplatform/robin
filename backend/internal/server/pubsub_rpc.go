package server

import "robinplatform.dev/internal/pubsub"

type GetTopicsInput struct {
}

var GetTopics = AppsRpcMethod[GetTopicsInput, []string]{
	Name: "GetTopics",
	Run: func(req RpcRequest[GetTopicsInput]) ([]string, *HttpError) {
		names := pubsub.Topics.GetTopics()
		return names, nil
	},
}

type SubscribeTopicInput struct {
}

var SubscribeTopic = Stream[SubscribeTopicInput, string]{
	Name: "SubscribeTopic",
	Run: func(req StreamRequest[SubscribeTopicInput], output chan<- string) error {
		return nil
	},
}
