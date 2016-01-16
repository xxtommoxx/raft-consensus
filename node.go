package raft

import (
	"fmt"
)

var _ = fmt.Printf

/**
	An fsm that contains three states -- leader, follower and candidate.
	Each state has a non-buffered channel associated for sending incoming requests.
	Since there's only one state active at a given point in time, no synchronization is required (in most cases)
	for managing the FSM. I.e. no synchronization is required when storing the max term.

	In most cases, doing the actual work for a given request can happen concurrently as long as it doesn't make
	the fsm transition to another state. For all requests we can quickly know what the next will be without
	doing the work that might take some time.
**/

type NodeFSM struct {
	currentState state
	maxTerm      uint32

	fsm map[state]stateHandler
}

type state int

const (
	leaderState state = iota
	followerState
	candidateState
	invalidState
)

type stateHandler struct {
	requests   chan interface{}
	service    Service
	transition func() state
}

func NewNodeFSM(follower *Follower, candidate *Candidate) *NodeFSM {
	// todo read maxterm from store
	nodeFSM := &NodeFSM{
		currentState: followerState,
	}

	fsm := map[state]stateHandler{
		followerState:  nodeFSM.followerHandler(follower),
		candidateState: nodeFSM.candidateHandler(candidate),
	}

	nodeFSM.fsm = fsm

	return nodeFSM
}

func (this *NodeFSM) candidateHandler(candidate *Candidate) stateHandler {
	return stateHandler{}
}

func (this *NodeFSM) followerHandler(follower *Follower) stateHandler {
	requests := make(chan interface{})

	transition := func() state {
		req := <-requests

		switch req.(type) {
		default:
			// todo
			return invalidState
		}

	}

	return stateHandler{
		requests:   requests,
		service:    follower,
		transition: transition,
	}
}

func (this *NodeFSM) getCurrent() (state, stateHandler) {
	return this.currentState, this.fsm[this.currentState]
}

func (this *NodeFSM) Start() {
	// the main loop that processes requests for the current state
	go func() {
		for {
			currentState, currentStateHandler := this.getCurrent()
			nextState := currentStateHandler.transition()

			if nextState != currentState {
				currentStateHandler.service.Stop()
				this.currentState = nextState
				this.fsm[nextState].service.Start(this.maxTerm)
			}
		}
	}()
}

func (this *NodeFSM) sendRequest(term uint32, fn func(chan interface{})) error {
	if term > this.maxTerm {
		// todo store the term
		this.maxTerm = term
	}

	if term == this.maxTerm {
		requestsChan := this.fsm[this.currentState].requests
		fn(requestsChan)
	} else {
		// todo return error
	}

	return nil
}
