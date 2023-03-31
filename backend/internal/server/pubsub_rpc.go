package server

import (
	"sync"

	"robinplatform.dev/internal/identity"
	"robinplatform.dev/internal/log"
	"robinplatform.dev/internal/project"
	"robinplatform.dev/internal/pubsub"
)

var topicMapMutex = sync.Mutex{}
var topicMap = make(map[string]*pubsub.Topic[any])

func getTopic(topicId pubsub.TopicId) *pubsub.Topic[any] {
	topicMapMutex.Lock()
	defer topicMapMutex.Unlock()

	return topicMap[topicId.String()]
}

func setTopic(topic *pubsub.Topic[any]) {
	topicMapMutex.Lock()
	defer topicMapMutex.Unlock()

	topicMap[topic.Id.String()] = topic
}

type CreateTopicInput struct {
	AppId    string   `json:"appId"`
	Category []string `json:"category"`
	Key      string   `json:"key"`
}

var CreateTopic = AppsRpcMethod[CreateTopicInput, struct{}]{
	Name: "CreateTopic",
	Run: func(req RpcRequest[CreateTopicInput]) (struct{}, *HttpError) {
		projectName, err := project.GetProjectName()
		if err != nil {
			// This should have been resolved long before.
			panic(err)
		}

		if req.Data.AppId == "" {
			return struct{}{}, Errorf(500, "App ID was an empty string")
		}

		categoryParts := []string{"app-topics", projectName, req.Data.AppId}
		categoryParts = append(categoryParts, req.Data.Category...)
		topicId := pubsub.TopicId{
			Category: identity.Category(categoryParts...),
			Key:      req.Data.Key,
		}

		topic, err := pubsub.CreateTopic[any](&pubsub.Topics, topicId)
		if err != nil {
			return struct{}{}, Errorf(500, "%s", err.Error())
		}

		setTopic(topic)

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
		logger.Debug("Publish to topic", log.Ctx{
			"appId":    req.Data.AppId,
			"category": req.Data.Category,
			"key":      req.Data.Key,
		})

		projectName, err := project.GetProjectName()
		if err != nil {
			// This should have been resolved long before.
			panic(err)
		}

		if req.Data.AppId == "" {
			return struct{}{}, Errorf(500, "App ID was an empty string")
		}

		categoryParts := []string{"app-topics", projectName, req.Data.AppId}
		categoryParts = append(categoryParts, req.Data.Category...)
		topicId := pubsub.TopicId{
			Category: identity.Category(categoryParts...),
			Key:      req.Data.Key,
		}

		topic := getTopic(topicId)
		if topic == nil {
			return struct{}{}, Errorf(500, "Topic not found: %s", topicId.String())
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
