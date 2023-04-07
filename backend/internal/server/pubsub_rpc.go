package server

import (
	"fmt"
	"strings"
	"sync"

	"robinplatform.dev/internal/identity"
	"robinplatform.dev/internal/log"
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
		if req.Data.AppId == "" {
			return struct{}{}, Errorf(500, "App ID was an empty string")
		}

		categoryParts := []string{"app-topics", req.Data.AppId}
		categoryParts = append(categoryParts, req.Data.Category...)
		topicId := pubsub.TopicId{
			Category: identity.Category(categoryParts...),
			Key:      req.Data.Key,
		}

		topic := getTopic(topicId)
		if topic != nil {
			// We've already created the topic; since app topics can't be closed right now,
			// it is easier to simply do this additional hack.
			return struct{}{}, nil
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

		if req.Data.AppId == "" {
			return struct{}{}, Errorf(500, "App ID was an empty string")
		}

		categoryParts := []string{"app-topics", req.Data.AppId}
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

		// Hard-coded hack for `/app-topics`, to ensure that the app that created
		// a given topic is started before the subscription is attempted.
		// This is... a little bit silly, to say the least.
		if strings.HasPrefix(input.Id.Category, "/app-topics/") {
			parts := strings.Split(input.Id.Category[1:], "/")
			if len(parts) < 2 {
				return err
			}

			appId := parts[1]
			app, _, err := req.Server.compiler.GetApp(appId)
			if err != nil {
				// the error messages from GetApp() are already user-friendly
				return err
			}

			if !app.IsAlive() {
				if err := app.StartServer(); err != nil {
					return fmt.Errorf("failed to start app server: %w", err)
				}
			}
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
