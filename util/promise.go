package util

type Promise struct {
	result interface{}
	err    error

	done chan struct{}
}

func NewPromise(f func() (interface{}, error)) *Promise {

	done := make(chan struct{})

	promise := &Promise{done: done}

	go func() {
		defer close(done)

		result, err := f()
		promise.result = result
		promise.err = err
	}()

	return promise
}

func (p *Promise) Get() (interface{}, error) {
	<-p.done
	return p.result, p.err
}
