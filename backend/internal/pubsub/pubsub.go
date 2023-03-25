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
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"robinplatform.dev/internal/identity"
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
	MetaTopic TopicId = TopicId{Category: "/topics", Key: "meta"}
)

// Identifier of a topic
type TopicId identity.Id

func (topic TopicId) String() string {
	return (identity.Id)(topic).String()
}

type Topic[T any] struct {
	// `id` is only set at creation time and isn't written to afterwards.
	Id TopicId
	// `registry` is only set at creation time and isn't written to afterwards.
	registry *Registry

	// This mutex controls the reading and writing of the
	// `subscribers` and `closed` fields.
	m sync.Mutex

	closed      bool
	subscribers []chan T
}

type anyTopic interface {
	addAnySubscriber(chan any) (func(), error)
	isClosed() bool
	getInfo() TopicInfo
}

type Subscription[T any] struct {
	Out         <-chan T
	Unsubscribe func()
}

func (topic *Topic[T]) forEachSubscriber(iterator func(sub chan<- T)) error {
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

func (topic *Topic[T]) addSubscriber(sub chan T) error {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return fmt.Errorf("%w: %s", ErrTopicClosed, topic.Id.String())
	}

	topic.subscribers = append(topic.subscribers, sub)

	return nil
}

func (topic *Topic[T]) addAnySubscriber(sub chan any) (func(), error) {
	channel := make(chan T)
	if err := topic.addSubscriber(channel); err != nil {
		return nil, err
	}

	go func() {
		for {
			val, ok := <-channel
			if !ok {
				close(sub)
				return
			}

			sub <- val
		}
	}()

	unsub := func() {
		// This close allows the goroutine to die when the subscriber unsubscribes
		close(channel)
		topic.removeSubscriber(channel)
	}

	return unsub, nil
}

func (topic *Topic[T]) removeSubscriber(sub <-chan T) {
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

func (topic *Topic[_]) getInfo() TopicInfo {
	topic.m.Lock()
	defer topic.m.Unlock()

	return TopicInfo{
		Id:              topic.Id,
		Closed:          topic.closed,
		SubscriberCount: len(topic.subscribers),
	}
}

func (topic *Topic[_]) isClosed() bool {
	topic.m.Lock()
	defer topic.m.Unlock()

	return topic.closed
}

func (topic *Topic[T]) Publish(message T) {
	topic.forEachSubscriber(func(sub chan<- T) {
		sub <- message
	})
}

func (topic *Topic[_]) Close() {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return
	}

	topic.closed = true

	if meta := topic.registry.metaTopic.Load(); meta != nil {
		meta.Publish(MetaTopicInfo{
			Kind: "close",
			Data: topic.Id,
		})
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

	metaTopic atomic.Pointer[Topic[MetaTopicInfo]]

	// TODO: this implementation will scatter stuff all over the heap.
	// It can be fixed with some kind of stable-pointer-arraylist but
	// that's not worth writing right now
	topics map[string]anyTopic
}

func CreateTopic[T any](r *Registry, id TopicId) (*Topic[T], error) {
	if strings.HasPrefix(id.Category, "/topics") {
		return nil, ErrTopicExists
	}

	r.m.Lock()
	defer r.m.Unlock()

	return createTopic[T](r, id)
}

// Requires caller to take the lock
func createTopic[T any](r *Registry, id TopicId) (*Topic[T], error) {
	if r.topics == nil {
		r.topics = make(map[string]anyTopic, 8)
	}

	key := id.String()
	if prev := r.topics[key]; prev != nil && !prev.isClosed() {
		return nil, fmt.Errorf("%w: %s", ErrTopicExists, id.String())
	}

	topic := &Topic[T]{Id: id, registry: r}
	r.topics[key] = topic

	return topic, nil
}

func (r *Registry) pollMetaInfo() {
	for {
		time.Sleep(time.Second / 2)

		r.m.Lock()

		metaTopic := r.metaTopic.Load()
		if metaTopic == nil {
			continue
		}

		for _, topic := range r.topics {
			info := topic.getInfo()
			if info.Closed {
				continue
			}

			metaTopic.Publish(MetaTopicInfo{
				Kind: "update",
				Data: info,
			})
		}

		r.m.Unlock()

	}
}

func (r *Registry) CreateMetaTopics() error {
	r.m.Lock()
	defer r.m.Unlock()

	// Lazily create meta topic
	meta, err := createTopic[MetaTopicInfo](r, MetaTopic)
	if err != nil {
		return err
	}
	r.metaTopic.Store(meta)

	go r.pollMetaInfo()

	return nil
}

func getTopic(r *Registry, id TopicId) (anyTopic, error) {
	key := id.String()

	r.m.Lock()

	if r.topics == nil {
		r.topics = make(map[string]anyTopic, 8)
	}

	topicUntyped := r.topics[key]
	r.m.Unlock()

	if topicUntyped == nil {
		return nil, fmt.Errorf("%w: %s", ErrTopicDoesntExist, id.String())
	}

	return topicUntyped, nil
}

func SubscribeAny(r *Registry, id TopicId) (Subscription[any], error) {
	topicUntyped, err := getTopic(r, id)
	if err != nil {
		return Subscription[any]{}, err
	}

	channel := make(chan any)

	unsub, err := topicUntyped.addAnySubscriber(channel)
	if err != nil {
		return Subscription[any]{}, err
	}

	sub := Subscription[any]{
		Out:         channel,
		Unsubscribe: unsub,
	}

	return sub, nil
}

func Subscribe[T any](r *Registry, id TopicId) (Subscription[T], error) {
	topicUntyped, err := getTopic(r, id)
	if err != nil {
		return Subscription[T]{}, err
	}

	channel := make(chan T)

	topic, ok := topicUntyped.(*Topic[T])
	if !ok {
		return Subscription[T]{}, fmt.Errorf("%w: %s topic was the wrong type", ErrTopicDoesntExist, id.String())
	}

	if err := topic.addSubscriber(channel); err != nil {
		return Subscription[T]{}, err
	}

	sub := Subscription[T]{
		Out: channel,
		Unsubscribe: func() {
			topic.removeSubscriber(channel)
		},
	}

	return sub, nil
}

type TopicInfo struct {
	Id              TopicId `json:"id"`
	Closed          bool    `json:"closed"`
	SubscriberCount int     `json:"subscriberCount"`
}

func (r *Registry) GetTopicInfo() map[string]TopicInfo {
	r.m.Lock()
	defer r.m.Unlock()

	out := make(map[string]TopicInfo, len(r.topics))

	for key, topic := range r.topics {
		out[key] = topic.getInfo()
	}

	return out
}
