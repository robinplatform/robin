package pubsub

import (
	"sync"
	"testing"
)

func TestPubSubSimple(t *testing.T) {
	var registry Registry

	topicId := TopicId{
		Category: "wassa",
		Key:      "wassa",
	}
	topic, err := CreateTopic(&registry, topicId)
	if err != nil {
		t.Fatalf("topic couldn't be created: %s", err.Error())
	}

	var wStart sync.WaitGroup
	var wStop sync.WaitGroup
	failChannel := make(chan string, 10)
	defer close(failChannel)

	subscriber := func(channel <-chan string) {
		wStart.Done()

		message, ok := <-channel
		if !ok {
			failChannel <- "failed to read from channel"
			return
		}
		if message != "Hello" {
			failChannel <- "message was wrong"
			return
		}

		select {
		case message, ok = <-channel:
			failChannel <- "read too many messages"
			return
		default:
		}

		wStop.Done()
	}

	for i := 0; i < 10; i++ {
		wStart.Add(1)
		wStop.Add(1)

		sub, err := Subscribe(&registry, topicId)
		if err != nil {
			t.Fatalf("error: %s", err.Error())
		}

		go subscriber(sub.Out)
	}

	wStart.Wait()

	topic.Publish("Hello")

	endChan := make(chan bool, 1)

	go func() {
		wStop.Wait()
		endChan <- true
	}()

	select {
	case message := <-failChannel:
		t.Fatalf(message)
	case <-endChan:
	}
}
