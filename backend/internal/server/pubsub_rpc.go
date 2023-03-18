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
