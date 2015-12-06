package fsm

import (
	"fmt"
	"reflect"
)

type State uint32
type EventType reflect.Type
type Event interface{}
type EventHandler interface {
	Handle(e Event, s State)
	CanHandle(e EventType) bool
}

type fsm struct {
	stateTransitions map[State]map[EventType]State
	handlers         map[EventType][]EventHandler

	currentState State
}

type Transition struct {
	EventType interface{}
	To        State
}

func NewFSM(initialState State, stateTransitions map[State][]Transition, eventHandlers []EventHandler) *fsm {
	handlers := make(map[EventType][]EventHandler)
	eventTypeSet := make(map[EventType]bool)

	_stateTransitions := toInternalStateTransitions(stateTransitions)

	for _, eventTypeMap := range _stateTransitions {
		for eventType, _ := range eventTypeMap {
			eventTypeSet[eventType] = true
		}
	}

	for eventType, _ := range eventTypeSet {
		for _, h := range eventHandlers {
			if h.CanHandle(eventType) {
				if arr, ok := handlers[eventType]; ok {
					handlers[eventType] = append(arr, h)
				} else {
					handlers[eventType] = []EventHandler{h}
				}
			}
		}
	}

	return &fsm{
		stateTransitions: _stateTransitions,
		handlers:         handlers,
		currentState:     initialState,
	}
}

func toInternalStateTransitions(stateTransitions map[State][]Transition) map[State]map[EventType]State {
	_stateTransitions := make(map[State]map[EventType]State)

	for from, transitions := range stateTransitions {
		for _, t := range transitions {
			eventType := reflect.TypeOf(t.EventType)

			if eventMap, ok := _stateTransitions[from]; ok {
				eventMap[eventType] = t.To
			} else {
				_stateTransitions[from] = map[EventType]State{
					eventType: t.To,
				}
			}
		}
	}

	return _stateTransitions
}

func (fsm *fsm) Transition(e Event) State {
	eventType := reflect.TypeOf(e)

	if handlers, ok := fsm.handlers[eventType]; ok {
		for _, h := range handlers {
			h.Handle(e, fsm.currentState)
		}
	} else {
		panic(fmt.Sprintf("No handler found for event type: %v", eventType))
	}

	if eventMap, ok := fsm.stateTransitions[fsm.currentState]; ok {
		if nextState, ok := eventMap[eventType]; ok {
			fsm.updateState(nextState)
			return nextState
		}
	}
	panic(fmt.Sprintf("No valid transition from state: %v for event type: %v", fsm.currentState, eventType))
}

func (fsm *fsm) updateState(newState State) {
	fsm.currentState = newState
}
