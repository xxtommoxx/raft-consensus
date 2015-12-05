package main

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/util/fsm"
	"reflect"
)

const (
	S1 = iota
	S2
	S3
)

type E1 struct {
}
type E2 struct{}

type E1Handler struct{}

func (E1Handler) Handle(e fsm.Event, s fsm.State) {
	fmt.Printf("E1Handler handling event %v in state: %v\n\n", reflect.TypeOf(e), s)

}

func (E1Handler) CanHandle(e fsm.EventType) bool {
	return e == reflect.TypeOf(E1{})
}

type E1E2EventHandler struct{}

func (E1E2EventHandler) Handle(e fsm.Event, s fsm.State) {
	fmt.Printf("E1E2Handler handling event %v in state: %v\n\n", reflect.TypeOf(e), s)

}

func (E1E2EventHandler) CanHandle(e fsm.EventType) bool {
	return e == reflect.TypeOf(E1{}) || e == reflect.TypeOf(E2{})
}

func main() {
	s1Transitions := []fsm.Transition{
		fsm.Transition{E1{}, S2},
		fsm.Transition{E2{}, S3},
	}

	s2Transitions := []fsm.Transition{
		fsm.Transition{E1{}, S1},
		fsm.Transition{E2{}, S3},
	}

	s3Transitions := []fsm.Transition{
		fsm.Transition{E2{}, S3},
	}

	testFsm := fsm.NewFSM(S1, map[fsm.State][]fsm.Transition{
		S1: s1Transitions,
		S2: s2Transitions,
		S3: s3Transitions,
	}, []fsm.EventHandler{
		E1E2EventHandler{},
		E1Handler{}},
	)

	testFsm.Transition(E1{})
	testFsm.Transition(E2{})
	testFsm.Transition(E2{})
}
