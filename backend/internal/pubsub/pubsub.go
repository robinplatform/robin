package pubsub

/*
TODO: I'd like to add generics to this, ideally with some kind of API like this:

Impl:
```
func (reg *Registry) Subscribe[T any](name string, sub chan<- T) error {
  ...
}
```

Usage:
```
pubsub.Subscribe("subscriber", myChannelOfGenericType)
```

Of course, this would not work, because generic types aren't allowed on methods.
However, something similar to this would maybe be nice.

The implementation would likely use something like this to figure out the topic:
```
if narrowedTopic, castSucceeded := topic.(Topic[T]); castSucceeded {
  subscribe to the topic using the narrowedTopic variable...
}
```
*/

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var Topics Registry

var (
	ErrTopicClosed      error = errors.New("tried to operate on a closed topic")
	ErrTopicDoesntExist error = errors.New("tried to operate on a topic that doesn't exist")
	ErrTopicExists      error = errors.New("tried to create a topic that already exists")
	ErrNilSubscriber    error = errors.New("used a nil channel when subscribing")
)

type TopicId struct {
	// Category of the topic. The following categories are reserved:
	// - "robin"
	// - "@robin/*" - everything prefixed with "@robin/" is reserved
	Category string `json:"category"`
	// The name of the topic
	Name string `json:"name"`
}

func (topic *TopicId) String() string {
	return fmt.Sprintf("%s-%s", topic.Category, topic.Name)
}

func (topic *TopicId) HashKey() string {
	// Simple bit of code to prevent collisions
	cat := strings.ReplaceAll(topic.Category, "-", "\\-")
	name := strings.ReplaceAll(topic.Name, "-", "\\-")
	return fmt.Sprintf("%s-%s", cat, name)
}

type Topic struct {
	// `id` is only set at creation time and isn't written to afterwards.
	Id TopicId

	// This mutex controls the reading and writing of the
	// `subscribers` and `closed` fields.
	m sync.Mutex

	counter     int
	closed      bool
	subscribers []chan<- string
}

func (topic *Topic) forEachSubscriber(iterator func(sub chan<- string)) error {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return fmt.Errorf("%w: %s", ErrTopicClosed, topic.Id.String())
	}

	for _, sub := range topic.subscribers {
		iterator(sub)
	}

	return nil
}

func (topic *Topic) addSubscriber(sub chan<- string) error {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return fmt.Errorf("%w: %s", ErrTopicClosed, topic.Id.String())
	}

	topic.subscribers = append(topic.subscribers, sub)

	return nil
}

func (topic *Topic) removeSubscriber(sub chan<- string) {
	topic.m.Lock()
	defer topic.m.Unlock()

	writeIndex := 0
	for readIndex := 0; readIndex < len(topic.subscribers); readIndex++ {
		item := topic.subscribers[readIndex]
		if item == sub {
			continue
		}

		topic.subscribers[writeIndex] = item
		writeIndex += 1
	}

	topic.subscribers = topic.subscribers[:writeIndex]
}

func (topic *Topic) isClosed() bool {
	topic.m.Lock()
	defer topic.m.Unlock()

	return topic.closed
}

func (topic *Topic) Publish(message string) {
	topic.forEachSubscriber(func(sub chan<- string) {
		sub <- message
	})
}

func (topic *Topic) Close() {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return
	}

	topic.closed = true

	for _, channel := range topic.subscribers {
		close(channel)
	}

	topic.subscribers = nil
}

type Registry struct {
	m sync.Mutex

	// TODO: this implementation will scatter stuff all over the heap.
	// It can be fixed with some kind of stable-pointer-arraylist but
	// that's not worth writing right now
	topics map[string]*Topic
}

func (r *Registry) CreateTopic(id TopicId) (*Topic, error) {
	r.m.Lock()
	defer r.m.Unlock()

	if r.topics == nil {
		r.topics = make(map[string]*Topic, 8)
	}

	key := id.HashKey()
	if prev := r.topics[key]; prev != nil && prev.isClosed() {
		return nil, fmt.Errorf("%w: %s", ErrTopicExists, id.String())
	}

	topic := &Topic{Id: id}
	r.topics[key] = topic

	return topic, nil
}

func (r *Registry) Unsubscribe(id TopicId, channel chan<- string) {
	if channel == nil {
		return
	}

	key := id.HashKey()

	r.m.Lock()
	if r.topics == nil {
		r.topics = make(map[string]*Topic, 8)
	}

	topic := r.topics[key]
	r.m.Unlock()

	topic.removeSubscriber(channel)
}

func (r *Registry) Subscribe(id TopicId, channel chan<- string) error {
	if channel == nil {
		return ErrNilSubscriber
	}

	key := id.HashKey()

	r.m.Lock()

	if r.topics == nil {
		r.topics = make(map[string]*Topic, 8)
	}

	topic := r.topics[key]
	r.m.Unlock()

	if topic == nil {
		return fmt.Errorf("%w: %s", ErrTopicDoesntExist, id.String())
	}

	if err := topic.addSubscriber(channel); err != nil {
		return err
	}

	return nil
}

type TopicInfo struct {
	Id              TopicId `json:"id"`
	Closed          bool    `json:"closed"`
	Count           int     `json:"count"`
	SubscriberCount int     `json:"subscriberCount"`
}

func (r *Registry) GetTopicInfo() []TopicInfo {
	r.m.Lock()
	defer r.m.Unlock()

	out := make([]TopicInfo, 0, len(r.topics))

	for _, topic := range r.topics {
		topic.m.Lock()

		out = append(out, TopicInfo{
			Id:              topic.Id,
			Closed:          topic.closed,
			Count:           topic.counter,
			SubscriberCount: len(topic.subscribers),
		})

		topic.m.Unlock()
	}

	return out
}
