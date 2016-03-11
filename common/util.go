package common

import (
	"reflect"
)

type ForwardChan struct {
	ErrorCh   chan error
	SourceCh  chan interface{}
	toChValue reflect.Value
}

func NewForwardChan(src chan interface{}, toCh interface{}) *ForwardChan {
	f := &ForwardChan{
		SourceCh:  src,
		toChValue: reflect.ValueOf(toCh),
	}

	f.forward()

	return f
}

func ForwardWithErrorCh(toCh interface{}) *ForwardChan {
	toChValue := reflect.ValueOf(toCh)

	f := &ForwardChan{
		ErrorCh:   make(chan error),
		SourceCh:  make(chan interface{}, toChValue.Cap()),
		toChValue: toChValue,
	}

	f.forward()

	return f
}

func (f *ForwardChan) Close() {
	close(f.SourceCh)
}

func (f *ForwardChan) closeRemaining() {
	f.toChValue.Close()
	if f.ErrorCh != nil {
		close(f.ErrorCh)
	}
}

func (f *ForwardChan) forward() {
	go func() {
		defer f.closeRemaining()

		for {
			x, ok := <-f.SourceCh

			if ok {
				f.toChValue.Send(reflect.ValueOf(x))
			} else {
				return
			}
		}
	}()
}
