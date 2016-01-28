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
	rpcCh      chan rpcContext
	internalCh chan internalRequest // internal events where no response is needed
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

type internalRequest struct {
	term    uint32
	request interface{}
}

type rpcContext struct {
	term         uint32
	rpc          interface{}
	errorChan    chan error
	responseChan chan interface{}
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
	transition := this.commonTransitionFor(
		candidateState,
		func(internalReq internalRequest) state {
			switch internalReq.request.(type) {
			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch req := rpcCtx.rpc.(type) {
			default:
				return invalidState
			}
		})

	return stateHandler{
		service:    candidate,
		transition: transition,
	}
}

func (this *NodeFSM) followerHandler(follower *Follower) stateHandler {
	transition := this.commonTransitionFor(
		followerState,
		func(internalReq internalRequest) state {
			switch internalReq.request.(type) {
			case *KeepAliveTimeout:
				this.maxTerm = this.maxTerm + 1
				return candidateState
			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch req := rpcCtx.rpc.(type) {
			case *rpc.VoteRequest:
				this.processAsync(rpcCtx, func() (interface{}, error) { return follower.requestVote(req) })
				return followerState
			default:
				return invalidState
			}
		})

	return stateHandler{
		service:    follower,
		transition: transition,
	}
}

// Helper function that reads from the internal channel and rpc channel.
// Handles the term number encountered so that the passed in functions need not
// be concerned about checking the term number to determine whether or not it can handle it.
// If the function returns invalidState then an error message is sent over the rpc's
// response chan
func (this *NodeFSM) commonTransitionFor(_state state, internalReqFn func(internalRequest) state,
	rpcContextFn func(rpcContext) state) func() state {

	termTransition := func(term uint32, lt func(), gt func(), eq func() state, invalidFn func()) state {
		handleEq := func() state {
			nextState := eq()

			if nextState == invalidState {
				invalidFn()
				return _state
			} else {
				return nextState
			}
		}

		if term > this.maxTerm {
			// TODO store
			this.maxTerm = term

			if this.currentState != followerState {
				gt()
				return followerState
			} else {
				return handleEq()
			}
		} else if term < this.maxTerm {
			lt()
			return _state
		} else {
			return handleEq()
		}
	}

	return func() state {
		select {
		case internalRequest := <-this.internalCh:
			return termTransition(internalRequest.term,
				func() {},
				func() {},
				func() state { return internalReqFn(internalRequest) },
				func() {})

		case rpcCtx := <-this.rpcCh:
			return termTransition(rpcCtx.term,
				func() { rpcCtx.errorChan <- errors.New("Old Term") },
				func() { rpcCtx.errorChan <- errors.New("New Term") },
				func() state { return rpcContextFn(rpcCtx) },
				func() { rpcCtx.errorChan <- errors.New("Can't handle while in ___ state") })
		}
	}
}

func (this *NodeFSM) Start() {
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

func (this *NodeFSM) getCurrent() (state, stateHandler) {
	return this.currentState, this.fsm[this.currentState]
}

func (this *NodeFSM) Stop() {
	// todo close the chans
}

func (this *NodeFSM) processAsync(ctx rpcContext, fn func() (interface{}, error)) {
	go func() {
		result, err := fn()

		if err != nil {
			ctx.errorChan <- err
		} else {
			ctx.responseChan <- result
		}
	}()
}

func (this *NodeFSM) sendInternalRequest(term uint32, request interface{}) {
	this.internalCh <- internalRequest{
		term:    term,
		request: request,
	}
}

func (this *NodeFSM) sendToStateHandler(term uint32, rpc interface{}, responseChan interface{}) {
	this.rpcCh <- rpcContext{
		term:         term,
		rpc:          rpc,
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
	this.sendInternalRequest(timeout.term, timeout)
}
