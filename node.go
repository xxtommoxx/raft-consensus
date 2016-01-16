package raft

import (
	"fmt"
)

type state int

const (
	leaderState state = iota
	followerState
	candidateState
	invalidState
)

type NodeFSM struct {
	current state
	maxTerm uint32

	fsm map[state]stateHandler
}

type stateHandler struct {
	requests       chan interface{}
	stateLifecycle Lifecycle
	transition     func() state
}

func NewNodeFSM(follower *Follower, candidate *Candidate) *NodeFSM {
	// todo read maxterm from store
	nodeFSM := &NodeFSM{
		current: followerState,
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

		switch r := req.(type) {
		default:
			fmt.Println(r)
			return invalidState
		}

	}

	return stateHandler{
		requests:       requests,
		stateLifecycle: nil,
		transition:     transition,
	}
}

func (this *NodeFSM) Start() {
	go func() {
		for {
			currentI := this.fsm[this.current]

			nextState := currentI.transition()

			if nextState != this.current {
				currentI.stateLifecycle.Stop()
				this.fsm[nextState].stateLifecycle.Start(this.maxTerm)
				this.current = nextState
			}

		}
	}()
}

func (this *NodeFSM) sendRequest(reqTerm uint32, fn func(chan interface{})) error {
	if reqTerm >= this.maxTerm {
		this.maxTerm = reqTerm

		stateChan := this.fsm[this.current].requests
		fn(stateChan)
	} else {
		// return an error indicating failed
	}

	return nil
}
