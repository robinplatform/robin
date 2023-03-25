package pubsub

import (
	"fmt"
	"sync"
	"testing"
)

func TestPubSubTopicCollision(t *testing.T) {
	registry := &Registry{}

	topicId := TopicId{
		Category: "wassa",
		Key:      "wassa",
	}
	var err error

	_, err = CreateTopic[string](registry, topicId)
	if err != nil {
		t.Fatalf("topic couldn't be created: %s", err.Error())
	}

	metaTopicId := TopicId{
		Category: "/topics",
		Key:      "meta",
	}
	_, err = CreateTopic[string](registry, metaTopicId)
	if err == nil {
		t.Fatalf("creating meta topic should have failed, but didn't")
	}
}

func RunSubscriberMessages[T comparable](registry *Registry, topicId TopicId, count int, messages []T) error {
	var wStart sync.WaitGroup
	var wStop sync.WaitGroup

	failChannel := make(chan error)

	topic, err := CreateTopic[T](registry, topicId)
	if err != nil {
		return err
	}
	defer topic.Close()

	subscriber := func(sub Subscription[T]) {
		wStart.Done()
		defer wStop.Done()

		index := 0
		for msg := range sub.Out {
			if index > len(messages) {
				failChannel <- fmt.Errorf("read too many messages")
				return
			}
			if msg != messages[index] {
				failChannel <- fmt.Errorf("read the wrong message")
				return
			}
			index++
		}

		if index != len(messages) {
			failChannel <- fmt.Errorf("read too few messages")
		}
	}

	for i := 0; i < count; i++ {
		wStart.Add(1)
		wStop.Add(1)

		sub, err := Subscribe[T](registry, topicId)
		if err != nil {
			return err
		}

		go subscriber(sub)
	}

	wStart.Wait()

	for _, msg := range messages {
		topic.Publish(msg)
	}

	topic.Close()

	wStop.Wait()

	select {
	case err := <-failChannel:
		return err
	default:
		return nil
	}
}

func TestPubSubSimple(t *testing.T) {
	registry := &Registry{}

	topicId := TopicId{
		Category: "wassa",
		Key:      "wassa",
	}

	RunSubscriberMessages(registry, topicId, 10, []string{
		"Hello",
		"Hello",
		"Goodbye",
	})
}

func TestPubSubParallel(t *testing.T) {
	registry := &Registry{}

	id1 := TopicId{
		Category: "wassa",
		Key:      "wassa",
	}

	id2 := TopicId{
		Category: "bassa",
		Key:      "wassa",
	}

	id3 := TopicId{
		Category: "dassa",
		Key:      "wassa",
	}

	go RunSubscriberMessages(registry, id1, 3, []string{
		"Hello", "Blarg", "Goodbye",
	})

	go RunSubscriberMessages(registry, id2, 50, []bool{
		true, false, false, false, true,
	})

	go RunSubscriberMessages(registry, id3, 7, []int{
		1, 2, 3, 4, 5,
	})
}
