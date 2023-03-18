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
	Category string `json:"category"`
	Name     string `json:"name"`
}

var SubscribeTopic = Stream[SubscribeTopicInput, string]{
	Name: "SubscribeTopic",
	Run: func(req StreamRequest[SubscribeTopicInput], output chan<- string) error {

		id := pubsub.TopicId{
			Category: req.Input.Category,
			Name:     req.Input.Name,
		}

		subscription := make(chan string)
		if err := pubsub.Topics.Subscribe(id, subscription); err != nil {
			return err
		}

		// TODO: this is sorta unnecessary, ideally it should be possible to just wait for the topic to close
		for s := range subscription {
			output <- s
		}

		return nil
	},
}
