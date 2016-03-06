package common

import "sync"

type EventType int

const (
	LeaderKeepAliveTimeout EventType = iota
	QuorumObtained
	QuorumUnobtained
	ResponseReceived
)

func (e EventType) String() string {
	switch e {
	case LeaderKeepAliveTimeout:
		return "LeaderKeepAliveTimeout"
	case QuorumObtained:
		return "QuorumObtained"
	case QuorumUnobtained:
		return "QuorumUnobtained"
	case ResponseReceived:
		return "ResponseReceived"
	}
	return "Unknown event type"
}

type Event struct {
	Term      uint32
	EventType EventType
}

type EventListener interface {
	HandleEvent(Event)
}

type EventListenerDispatcher struct {
	mutex       sync.RWMutex
	dispatchers map[chan<- Event]struct{}
}

func NewEventListenerDispatcher() *EventListenerDispatcher {
	return &EventListenerDispatcher{dispatchers: make(map[chan<- Event]struct{})}
}

func (r *EventListenerDispatcher) Subscribe(ch chan<- Event) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.dispatchers[ch] = struct{}{}
}

func (r *EventListenerDispatcher) Unsubscribe(ch chan<- Event) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.dispatchers, ch)
}

func (r *EventListenerDispatcher) HandleEvent(e Event) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for d, _ := range r.dispatchers {
		go func() {
			d <- e
		}()
	}
}
