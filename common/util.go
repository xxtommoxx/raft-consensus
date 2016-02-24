package common

import "reflect"

func ToChan(in interface{}) chan interface{} {
	cin := reflect.ValueOf(in)
	out := make(chan interface{}, cin.Cap())

	go func() {
		defer close(out)
		for {
			x, ok := cin.Recv()
			if !ok {
				return
			}
			out <- x.Interface()
		}
	}()
	return out
}
