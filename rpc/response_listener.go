package rpc

import "sync"

type ResponseListener interface {
	ResponseReceived(term uint32)
}

type ResponseListenerDispatcher struct {
	mutex       sync.Mutex
	dispatchers []chan<- uint32
}

func NewResponseListenerDispatcher() *ResponseListenerDispatcher {
	return &ResponseListenerDispatcher{dispatchers: []chan<- uint32{}}
}

func (r *ResponseListenerDispatcher) Subscribe(ch chan<- uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.dispatchers = append(r.dispatchers, ch)
}

func (r *ResponseListenerDispatcher) ResponseReceived(term uint32) {
	for _, d := range r.dispatchers { // TODO use atomic read
		go func() {
			d <- term
		}()
	}
}
