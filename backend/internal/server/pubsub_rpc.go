package server

import (
	"robinplatform.dev/internal/pubsub"
)

type CreateTopicInput struct {
	AppId    string   `json:"appId"`
	Category []string `json:"category"`
	Key      string   `json:"key"`
}

var CreateTopic = AppsRpcMethod[CreateTopicInput, struct{}]{
	Name: "CreateTopic",
	Run: func(req RpcRequest[CreateTopicInput]) (struct{}, *HttpError) {
		app, _, err := req.Server.compiler.GetApp(req.Data.AppId)
		if err != nil {
			return struct{}{}, Errorf(500, "%s", err.Error())
		}

		topicId := app.TopicId(req.Data.Category, req.Data.Key)
		if _, err := app.UpsertTopic(topicId); err != nil {
			return struct{}{}, Errorf(500, "%s", err.Error())
		}

		return struct{}{}, nil
	},
}

type PublishTopicInput struct {
	AppId    string   `json:"appId"`
	Category []string `json:"category"`
	Key      string   `json:"key"`
	Data     any      `json:"data"`
}

var PublishTopic = AppsRpcMethod[PublishTopicInput, struct{}]{
	Name: "PublishToTopic",
	Run: func(req RpcRequest[PublishTopicInput]) (struct{}, *HttpError) {
		app, _, err := req.Server.compiler.GetApp(req.Data.AppId)
		if err != nil {
			return struct{}{}, Errorf(500, "%s", err.Error())
		}

		topicId := app.TopicId(req.Data.Category, req.Data.Key)
		topic, err := app.UpsertTopic(topicId)
		if err != nil {
			return struct{}{}, Errorf(500, "topic '%s' not found: %s", topicId.String(), err.Error())
		}

		topic.Publish(req.Data.Data)

		return struct{}{}, nil
	},
}

type GetTopicsInput struct {
}

var GetTopics = AppsRpcMethod[GetTopicsInput, pubsub.RegistryTopicInfo]{
	Name: "GetTopics",
	Run: func(req RpcRequest[GetTopicsInput]) (pubsub.RegistryTopicInfo, *HttpError) {
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

type SubscribeAppTopicInput struct {
	AppId    string   `json:"appId"`
	Category []string `json:"category"`
	Key      string   `json:"key"`
}

var SubscribeAppTopic = Stream[SubscribeAppTopicInput, any]{
	Name: "SubscribeAppTopic",
	Run: func(req *StreamRequest[SubscribeAppTopicInput, any]) error {
		input, err := req.ParseInput()
		if err != nil {
			return err
		}

		app, _, err := req.Server.compiler.GetApp(input.AppId)
		if err != nil {
			// the error messages from GetApp() are already user-friendly
			return err
		}

		if !app.IsAlive() {
			if err := app.StartServer(); err != nil {
				return err
			}
		}

		topicId := app.TopicId(input.Category, input.Key)
		return PipeTopic(topicId, req)
	},
}
