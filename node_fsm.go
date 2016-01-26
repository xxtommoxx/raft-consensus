package raft

import (
	"errors"
	"fmt"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"reflect"
)

var _ = fmt.Printf
var _ = &rpc.VoteRequest{}

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

	fsm        map[state]stateHandler
	requestCh  chan requestContext
	internalCh chan interface{} // internal events where no response is needed
}

type state int

const (
	leaderState state = iota
	followerState
	candidateState
	invalidState
)

type stateHandler struct {
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
	transition := func() state {

		select {
		case internalEvent := <-this.internalCh:
			fmt.Println(internalEvent)
			return invalidState
		case reqCtx := <-this.requestCh:
			nextState, responseFn := func() (state, func() (interface{}, error)) {
				switch r := reqCtx.request.(type) {

				case *rpc.VoteRequest:
					return followerState, func() (interface{}, error) { return follower.requestVote(r) }
				}

				return invalidState, invalidResponseFn
			}()

			this.processAsync(reqCtx, responseFn)

			return nextState
		}

	}

	return stateHandler{
		service:    follower,
		transition: transition,
	}
}

func invalidResponseFn() (interface{}, error) {
	return nil, nil
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

func (this *NodeFSM) processAsync(ctx requestContext, fn func() (interface{}, error)) error {
	if ctx.term > this.maxTerm { // do this outside of the go routine for sync purposes
		// todo store the term
		this.maxTerm = ctx.term
	}

	if ctx.term == this.maxTerm {
		go func() {
			result, err := fn()

			if err != nil {
				ctx.errorChan <- err
			} else {
				ctx.responseChan <- result
			}
		}()

		return nil
	} else {
		oldTermError := errors.New("Old Term")
		go func() {
			ctx.errorChan <- oldTermError
		}()

		return oldTermError
	}
}

type requestContext struct {
	term         uint32
	request      interface{}
	errorChan    chan error
	responseChan chan interface{}
}

func (this *NodeFSM) sendInternalEvent(event interface{}) {
	this.internalCh <- event
}

func (this *NodeFSM) sendToStateHandler(term uint32, request interface{}, responseChan interface{}) {
	this.requestCh <- requestContext{
		term:         term,
		request:      request,
		errorChan:    make(chan error),
		responseChan: toChan(responseChan),
	}
}

func toChan(in interface{}) chan interface{} {
	out := make(chan interface{})
	cin := reflect.ValueOf(in)

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

func (this *NodeFSM) OnKeepAliveTimeout(timeout *KeepAliveTimeout) {
	this.sendInternalEvent(timeout)
}
