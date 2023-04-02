package pubsub

/*
TODO: It may be useful or even necessary to include a "state" field for each topic, so for example,
a subscription can get the list of log statements that happened before it existed. ~Something something monad.~
I don't quite want to implement all that hoopla right this second, but it's something to be aware of.
*/

import (
	"errors"
	"fmt"
	"strings"
	"sync"

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

type Message[T any] struct {
	MessageId int32 `json:"messageId"` // The counter value when this message was sent
	Data      T     `json:"data"`      // The data associated with this message
}

type Topic[T any] struct {
	// `id` is only set at creation time and isn't written to afterwards.
	Id TopicId
	// `metaChannel` is only set at creation time and isn't written to afterwards.
	metaChannel chan MetaTopicInfo

	// This mutex controls the reading and writing of the
	// `subscribers` and `closed` fields.
	m sync.Mutex

	counter     int32
	closed      bool
	subscribers []chan Message[T]
}

type anyTopic interface {
	addAnySubscriber() (Subscription[any], error)
	isClosed() bool
	GetInfo() TopicInfo
}

type Subscription[T any] struct {
	Out         <-chan Message[T]
	Unsubscribe func()
}

func (topic *Topic[T]) addSubscriber() (chan Message[T], error) {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return nil, fmt.Errorf("%w: %s", ErrTopicClosed, topic.Id.String())
	}

	sub := make(chan Message[T], 4)
	topic.subscribers = append(topic.subscribers, sub)

	if topic.metaChannel != nil {
		topic.metaChannel <- MetaTopicInfo{
			Kind: "update",
			Data: TopicInfo{
				Id:              topic.Id,
				Closed:          topic.closed,
				SubscriberCount: len(topic.subscribers),
			},
		}
	}

	return sub, nil
}

func (topic *Topic[T]) addAnySubscriber() (Subscription[any], error) {
	channel, err := topic.addSubscriber()
	if err != nil {
		return Subscription[any]{}, err
	}

	anyChannel := make(chan Message[any])
	go func() {
		for {
			val, ok := <-channel
			if !ok {
				close(anyChannel)
				return
			}

			anyChannel <- Message[any]{
				MessageId: val.MessageId,
				Data:      val.Data,
			}
		}
	}()

	unsub := func() {
		topic.removeSubscriber(channel)

		// This close allows the goroutine to die when the subscriber unsubscribes
		close(channel)
	}

	sub := Subscription[any]{
		Out:         anyChannel,
		Unsubscribe: unsub,
	}

	return sub, nil
}

func (topic *Topic[T]) removeSubscriber(sub <-chan Message[T]) {
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

	if topic.metaChannel != nil {
		topic.metaChannel <- MetaTopicInfo{
			Kind: "update",
			Data: TopicInfo{
				Id:              topic.Id,
				Closed:          topic.closed,
				SubscriberCount: len(topic.subscribers),
			},
		}
	}
}

func (topic *Topic[_]) GetInfo() TopicInfo {
	topic.m.Lock()
	defer topic.m.Unlock()

	return TopicInfo{
		Id:              topic.Id,
		Counter:         topic.counter,
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
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return
	}

	for _, sub := range topic.subscribers {
		sub <- Message[T]{
			MessageId: topic.counter,
			Data:      message,
		}
	}

	topic.counter += 1
}

func (topic *Topic[_]) Close() {
	topic.m.Lock()
	defer topic.m.Unlock()

	if topic.closed {
		return
	}

	topic.closed = true

	if topic.metaChannel != nil {
		topic.metaChannel <- MetaTopicInfo{
			Kind: "close",
			Data: topic.Id,
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

	metaChannel chan MetaTopicInfo

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

	topic := &Topic[T]{Id: id, metaChannel: r.metaChannel}
	r.topics[key] = topic

	if r.metaChannel != nil {
		r.metaChannel <- MetaTopicInfo{
			Kind: "update",
			Data: TopicInfo{
				Id:              topic.Id,
				Closed:          topic.closed,
				SubscriberCount: len(topic.subscribers),
			},
		}
	}

	return topic, nil
}

func (r *Registry) CreateMetaTopics() error {
	r.m.Lock()
	defer r.m.Unlock()

	// This is VERY messy. The meta channel is buffered so that when
	// you subscribe to the meta channel, there's not an automatic deadlock
	// between the subscriber trying to add to the metaChannel and the
	// goroutine below trying to get the meta topic mutex. However,
	// this does not necessarily guarantee that the goroutine won't deadlock later,
	// if a lock on the meta topic happens when a crapton of messages are being sent in
	// the channel.
	metaChannel := make(chan MetaTopicInfo, 8)
	r.metaChannel = metaChannel

	// Lazily create meta topic
	meta, err := createTopic[MetaTopicInfo](r, MetaTopic)
	if err != nil {
		return err
	}

	go func() {
		for item := range metaChannel {
			meta.Publish(item)
		}
	}()

	return nil
}

func getTopic(r *Registry, id TopicId) (anyTopic, error) {
	key := id.String()

	var topicUntyped anyTopic

	r.m.Lock()
	if r.topics != nil {
		topicUntyped = r.topics[key]
	}
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

	sub, err := topicUntyped.addAnySubscriber()
	if err != nil {
		return Subscription[any]{}, err
	}

	return sub, nil
}

func Subscribe[T any](r *Registry, id TopicId) (Subscription[T], error) {
	topicUntyped, err := getTopic(r, id)
	if err != nil {
		return Subscription[T]{}, err
	}

	topic, ok := topicUntyped.(*Topic[T])
	if !ok {
		return Subscription[T]{}, fmt.Errorf("%w: %s topic was the wrong type", ErrTopicDoesntExist, id.String())
	}

	channel, err := topic.addSubscriber()
	if err != nil {
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
	Counter         int32   `json:"counter"`
	SubscriberCount int     `json:"subscriberCount"`
}

type RegistryTopicInfo struct {
	Counter int32                `json:"counter"`
	Info    map[string]TopicInfo `json:"info"`
}

func (r *Registry) GetTopicInfo() RegistryTopicInfo {
	r.m.Lock()
	defer r.m.Unlock()

	out := RegistryTopicInfo{
		Info: make(map[string]TopicInfo, len(r.topics)),
	}

	if t, ok := r.topics[MetaTopic.String()]; ok {
		out.Counter = t.GetInfo().Counter
	}

	for key, topic := range r.topics {
		out.Info[key] = topic.GetInfo()
	}

	return out
}
