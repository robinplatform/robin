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

It may also be useful or even necessary to include a "state" field for each topic, so for example,
a subscription can get the list of log statements that happened before it existed. ~Something something monad.~
I don't quite want to implement all that hoopla right this second, but it's something to be aware of.
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var Topics Registry

func init() {
	err := Topics.CreateMetaTopics()
	if err != nil {
		panic(err)
	}
}

var (
	ErrTopicClosed      error = errors.New("tried to operate on a closed topic")
	ErrTopicDoesntExist error = errors.New("tried to operate on a topic that doesn't exist")
	ErrTopicExists      error = errors.New("tried to create a topic that already exists")
	ErrNilSubscriber    error = errors.New("used a nil channel when subscribing")
)

var (
	MetaTopic TopicId = TopicId{Category: "@robin/topics", Name: "meta"}
)

type TopicId struct {
	// Category of the topic. The following categories are reserved:
	// - "robin"
	// - "@robin/*" - everything prefixed with "@robin/" is reserved
	//
	// Currently used:
	// - "@robin/logs/{app-category}" logs for an app with a certain category
	// - "@robin/topics" meta category for information about topics
	Category string `json:"category"`
	// The name of the topic
	Name string `json:"name"`
}

func (topic *TopicId) String() string {
	return fmt.Sprintf("%s/%s", topic.Category, topic.Name)
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

func (r *Registry) Close(topic *Topic) {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return
	}

	topic.closed = true

	if meta := r.metaTopic.Load(); meta != nil {
		data, err := json.Marshal(MetaTopicInfo{
			Kind: "close",
			Data: topic.Id,
		})

		// TODO: handle errors
		if err == nil {
			meta.Publish(string(data))
		}
	}

	for _, channel := range topic.subscribers {
		close(channel)
	}

	topic.subscribers = nil
}

type MetaTopicInfo struct {
	Kind string `json:"kind"`
	Data any    `json:"data"`
}

type Registry struct {
	m sync.Mutex

	metaTopic atomic.Pointer[Topic]

	// TODO: this implementation will scatter stuff all over the heap.
	// It can be fixed with some kind of stable-pointer-arraylist but
	// that's not worth writing right now
	topics map[string]*Topic
}

func (r *Registry) CreateTopic(id TopicId) (*Topic, error) {
	if strings.HasPrefix(id.Category, "@robin/topics") {
		return nil, ErrTopicExists
	}

	r.m.Lock()
	defer r.m.Unlock()

	return r.createTopic(id)
}

// Requires caller to take the lock
func (r *Registry) createTopic(id TopicId) (*Topic, error) {
	if r.topics == nil {
		r.topics = make(map[string]*Topic, 8)
	}

	key := id.String()
	if prev := r.topics[key]; prev != nil && !prev.isClosed() {
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

	key := id.String()

	r.m.Lock()
	if r.topics == nil {
		r.topics = make(map[string]*Topic, 8)
	}

	topic := r.topics[key]
	r.m.Unlock()

	topic.removeSubscriber(channel)
}

func (r *Registry) pollMetaInfo() {
	for {
		r.m.Lock()

		for _, topic := range r.topics {
			topic.m.Lock()

			if topic.closed {
				topic.m.Unlock()
				continue
			}

			info := MetaTopicInfo{
				Kind: "update",
				Data: TopicInfo{
					Id:              topic.Id,
					Closed:          topic.closed,
					Count:           topic.counter,
					SubscriberCount: len(topic.subscribers),
				},
			}

			topic.m.Unlock()

			data, err := json.Marshal(info)
			if err != nil {
				continue
			}

			r.metaTopic.Load().Publish(string(data))
		}

		r.m.Unlock()

		time.Sleep(time.Millisecond * 500)
	}
}

func (r *Registry) CreateMetaTopics() error {
	r.m.Lock()
	defer r.m.Unlock()

	// Lazily create meta topic
	meta, err := r.createTopic(MetaTopic)
	if err != nil {
		return err
	}
	r.metaTopic.Store(meta)

	go r.pollMetaInfo()

	return nil
}

func (r *Registry) Subscribe(id TopicId, channel chan<- string) error {
	if channel == nil {
		return ErrNilSubscriber
	}

	key := id.String()

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

func (r *Registry) GetTopicInfo() map[string]TopicInfo {
	r.m.Lock()
	defer r.m.Unlock()

	out := make(map[string]TopicInfo, len(r.topics))

	for key, topic := range r.topics {
		topic.m.Lock()

		out[key] = TopicInfo{
			Id:              topic.Id,
			Closed:          topic.closed,
			Count:           topic.counter,
			SubscriberCount: len(topic.subscribers),
		}

		topic.m.Unlock()
	}

	return out
}
