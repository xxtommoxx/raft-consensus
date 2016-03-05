package raft

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
)

type state int

const (
	leaderState state = iota
	followerState
	candidateState
	invalidState
)

type stateHandler struct {
	service    common.Service
	transition func() state
}

type internalRequest struct {
	term  uint32
	event internalEvent
}

const (
	leaderTimeout internalEvent = iota
	quorumObtained
	quorumUnobtained
	responseReceived
)

type internalEvent int

type rpcContext struct {
	term         uint32
	rpc          interface{}
	errorChan    chan error
	responseChan chan interface{}
}

/**
	An fsm that contains three states -- leader, follower and candidate.
	Each state has a non-buffered channel associated for sending incoming requests.
**/
type NodeFSM struct {
	currentState state

	fsm        map[state]stateHandler
	rpcCh      chan rpcContext
	internalCh chan internalRequest // internal events where no response is needed

	termResponseCh chan uint32
	newTermCh      chan uint32

	stopCh chan struct{}

	stateStore StateStore

	*common.SyncService
}

func NewNodeFSM(stateStore StateStore, dispatcher *rpc.ResponseListenerDispatcher,
	candidate *Candidate, follower *Follower, leader *Leader) *NodeFSM {

	nodeFSM := &NodeFSM{
		currentState: followerState,

		stateStore: stateStore,

		rpcCh:          make(chan rpcContext),
		internalCh:     make(chan internalRequest),
		termResponseCh: make(chan uint32),
		newTermCh:      make(chan uint32, 1),
	}

	fsm := map[state]stateHandler{
		followerState:  nodeFSM.followerHandler(follower),
		candidateState: nodeFSM.candidateHandler(candidate),
		leaderState:    nodeFSM.leaderHandler(leader),
	}

	nodeFSM.fsm = fsm
	nodeFSM.SyncService = common.NewSyncService(nodeFSM.syncStart, nodeFSM.asyncStart, nodeFSM.syncStop)

	candidate.SetListener(nodeFSM)
	follower.SetListener(nodeFSM)

	dispatcher.Subscribe(nodeFSM.termResponseCh)

	return nodeFSM
}

func (this *NodeFSM) syncStart() error {
	log.Info("Starting node fsm")
	this.currentState = followerState

	_, currentStateHandler := this.getCurrent()

	currentStateHandler.service.Start()

	this.stopCh = make(chan struct{})

	return nil
}

func (this *NodeFSM) asyncStart() {
	for {
		select {
		case <-this.stopCh:
			log.Debug("Shutting down node fsm async start")
			this.storeNewTerm()
			return
		case term := <-this.termResponseCh:
			this.sendInternalRequest(term, responseReceived)
		default:
			currentState, currentStateHandler := this.getCurrent()
			log.Debug("In current state: ", currentState)

			nextState := currentStateHandler.transition()
			log.Debug("Transitioned to ", nextState)

			if nextState != currentState {
				currentStateHandler.service.Stop()

				this.storeNewTerm()

				this.currentState = nextState
				this.fsm[nextState].service.Start()
			}
		}
	}
}

func (this *NodeFSM) syncStop() error {
	log.Debug("Shutting down node fsm")

	_, stateHandler := this.getCurrent()
	err := stateHandler.service.Stop()

	close(this.stopCh)

	return err
}

func (this *NodeFSM) candidateHandler(candidate *Candidate) stateHandler {
	transition := this.commonTransitionFor(
		candidateState,
		func(internalReq internalRequest) state {
			switch internalReq.event {
			case quorumObtained:
				return leaderState
			case quorumUnobtained:
				return followerState
			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch rpcCtx.rpc.(type) {
			default:
				return invalidState
			}
		})

	return stateHandler{
		service:    candidate,
		transition: transition,
	}
}

func (this *NodeFSM) leaderHandler(leader *Leader) stateHandler {
	transition := this.commonTransitionFor(
		leaderState,
		func(internalReq internalRequest) state {
			switch internalReq.event {
			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch rpcCtx.rpc.(type) {
			default:
				return invalidState
			}
		})

	return stateHandler{
		service:    leader,
		transition: transition,
	}
}

func (this *NodeFSM) followerHandler(follower *Follower) stateHandler {
	transition := this.commonTransitionFor(
		followerState,
		func(internalReq internalRequest) state {
			switch internalReq.event {
			case leaderTimeout:
				this.newTermCh <- this.stateStore.CurrentTerm() + 1
				return candidateState
			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch req := rpcCtx.rpc.(type) {
			case *rpc.VoteRequest:
				this.processAsync(rpcCtx, func() (interface{}, error) { return follower.RequestVote(req) })
				return followerState
			case *rpc.KeepAliveRequest:
				this.processAsync(rpcCtx, func() (interface{}, error) { return follower.KeepAliveRequest(req) })
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

		currentTerm := this.stateStore.CurrentTerm()

		if term > currentTerm {
			this.newTermCh <- term

			if this.currentState != followerState { // revert to follower if higher term seen for leader and candidate
				gt()
				return followerState
			}
		} else if term < currentTerm {
			lt()
			return _state
		}

		return handleEq()
	}

	return func() state {
		select {
		case internalRequest := <-this.internalCh:
			log.Debug("Got internal request: ", internalRequest)

			return termTransition(internalRequest.term,
				func() {},
				func() {},
				func() state { return internalReqFn(internalRequest) },
				func() {})

		case rpcCtx := <-this.rpcCh:
			log.Debug("Got rpc request: ", rpcCtx.rpc)

			return termTransition(rpcCtx.term,
				func() {
					rpcCtx.errorChan <- errors.New(fmt.Sprint("Old Term: %v Current Term: %v", rpcCtx.term, this.stateStore.CurrentTerm()))
				},
				func() { this.rpcCh <- rpcCtx }, // replay by adding it back to the rpc channel after it has been transitioned to follower
				func() state { return rpcContextFn(rpcCtx) },
				func() {
					rpcCtx.errorChan <- errors.New(fmt.Sprintf("Can't handle while in %v state", this.currentState))
				})
		}
	}
}

func (this *NodeFSM) storeNewTerm() {
	// see if any greater term occurred while processing a request
	select {
	case newTerm := <-this.newTermCh:
		this.stateStore.SaveCurrentTerm(newTerm)
	default:
	}
}

func (this *NodeFSM) getCurrent() (state, stateHandler) {
	return this.currentState, this.fsm[this.currentState]
}

func (this *NodeFSM) processAsync(ctx rpcContext, fn func() (interface{}, error)) {
	go func() {
		result, err := fn()

		defer close(ctx.errorChan)
		defer close(ctx.responseChan)

		if err != nil {
			ctx.errorChan <- err
		} else {
			ctx.responseChan <- result
		}
	}()
}

func (this *NodeFSM) OnKeepAliveTimeout(term uint32) {
	this.sendInternalRequest(term, leaderTimeout)
}

func (this *NodeFSM) QuorumObtained(term uint32) {
	log.Debug("Quorum obtained for term: ", term)
	this.sendInternalRequest(term, quorumObtained)
}

func (this *NodeFSM) QuorumUnobtained(term uint32) {
	log.Debug("Quorum unobtained for term: ", term)
	this.sendInternalRequest(term, quorumUnobtained)
}

func (this *NodeFSM) RequestVote(vote *rpc.VoteRequest) (<-chan *rpc.VoteResponse, <-chan error) {
	respCh := make(chan *rpc.VoteResponse)
	errorCh := this.sendRpcRequest(vote.Term, vote, respCh)
	return respCh, errorCh
}
func (this *NodeFSM) KeepAlive(req *rpc.KeepAliveRequest) (<-chan *rpc.KeepAliveResponse, <-chan error) {
	respCh := make(chan *rpc.KeepAliveResponse)
	errorCh := this.sendRpcRequest(req.LeaderInfo.Term, req, respCh)
	return respCh, errorCh
}

func (this *NodeFSM) sendInternalRequest(term uint32, ie internalEvent) {
	this.internalCh <- internalRequest{
		term:  term,
		event: ie,
	}
}

func (this *NodeFSM) sendRpcRequest(term uint32, rpc interface{}, responseChan interface{}) <-chan error {
	errorCh := make(chan error)

	go func() {
		this.rpcCh <- rpcContext{
			term:         term,
			rpc:          rpc,
			errorChan:    make(chan error),
			responseChan: common.ToForwardedChan(responseChan),
		}
	}()

	return errorCh
}
