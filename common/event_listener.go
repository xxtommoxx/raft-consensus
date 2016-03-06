package common

import "sync"

type EventType int

const (
	LeaderKeepAliveTimeout EventType = iota
	QuorumObtained
	QuorumUnobtained
	ResponseReceived
)

type Event struct {
	Term      uint32
	EventType EventType
}

type EventListener interface {
	HandleEvent(Event)
}

type EventListenerDispatcher struct {
	mutex       sync.Mutex
	dispatchers []chan<- Event
}

func NewEventListenerDispatcher() *EventListenerDispatcher {
	return &EventListenerDispatcher{dispatchers: []chan<- Event{}}
}

func (r *EventListenerDispatcher) Subscribe(ch chan<- Event) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.dispatchers = append(r.dispatchers, ch)
}

func (r *EventListenerDispatcher) Event(e Event) {
	for _, d := range r.dispatchers { // TODO use atomic read
		go func() {
			d <- e
		}()
	}
}
