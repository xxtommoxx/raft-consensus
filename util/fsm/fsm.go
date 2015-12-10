package fsm

import (
	"errors"
)

const InvalidState State = -1

type State int32
type EventHandler interface {
	Handle(event interface{}) (State, bool)
}

type fsm struct {
	handlers     map[State]EventHandler
	currentState State
}

func Transition(to State) (State, bool) {
	return to, true
}

func InvalidTransition() (State, bool) {
	return InvalidState, false
}

func NewFSM(initialState State, handlers map[State]EventHandler) *fsm {
	return &fsm{handlers, initialState}
}

func (fsm *fsm) Fire(event interface{}) (State, error) {
	if h, ok := fsm.handlers[fsm.currentState]; ok {
		if to, success := h.Handle(event); success {
			fsm.updateState(to)
			return to, nil
		} else {
			return InvalidState, errors.New("Failed to handle") // TODO better msg
		}
	}

	return InvalidState, errors.New("Unable to find handler.") // TODO better msg
}

func (fsm *fsm) Stay() (State, bool) {
	return fsm.currentState, true
}

func (fsm *fsm) updateState(newState State) {
	fsm.currentState = newState
}
