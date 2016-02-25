package common

import "reflect"

func FowardChan(src <-chan interface{}, toCh interface{}) {
	fowardChanHelper(src, reflect.ValueOf(toCh))
}

func ToForwardedChan(toCh interface{}) chan interface{} {
	toChValue := reflect.ValueOf(toCh)
	src := make(chan interface{}, toChValue.Cap())
	fowardChanHelper(src, toChValue)
	return src
}

func fowardChanHelper(src <-chan interface{}, to reflect.Value) {
	go func() {
		defer to.Close()

		for {
			x, ok := <-src

			if ok {
				to.Send(reflect.ValueOf(x))
			} else {
				return
			}

		}
	}()
}
