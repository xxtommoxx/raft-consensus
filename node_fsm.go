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

func (s state) String() string {
	switch s {
	case leaderState:
		return "leader"
	case followerState:
		return "follower"
	case candidateState:
		return "candidate"
	case invalidState:
		return "invalid"
	}

	return "unknown"
}

type rpcContext struct {
	term         uint32
	rpc          interface{}
	errorChan    chan error
	responseChan chan interface{}
}

type stateHandler struct {
	service     common.Service
	handleEvent eventFn
	handleRpc   rpcFn
}

type eventFn func(common.Event) state
type rpcFn func(rpcContext) state

/**
	An fsm that contains three states -- leader, follower and candidate.
	Each state has a non-buffered channel associated for sending incoming requests.
**/
type NodeFSM struct {
	currentState state

	fsm map[state]stateHandler

	rpcCh   chan rpcContext
	eventCh chan common.Event // internal events where no response is needed

	termResponseCh chan uint32
	newTermCh      chan uint32

	stopCh chan struct{}

	stateStore StateStore

	*common.SyncService
}

func NewNodeFSM(stateStore StateStore, dispatcher *common.EventListenerDispatcher,
	follower *Follower, candidate *Candidate, leader *Leader) *NodeFSM {

	nodeFSM := &NodeFSM{
		currentState: followerState,

		stateStore: stateStore,

		rpcCh:     make(chan rpcContext),
		eventCh:   make(chan common.Event),
		newTermCh: make(chan uint32, 1),
	}

	nodeFSM.SyncService = common.NewSyncService(nodeFSM.syncStart, nodeFSM.asyncStart, nodeFSM.syncStop)

	nodeFSM.fsm = map[state]stateHandler{
		followerState:  nodeFSM.followerHandler(follower),
		candidateState: nodeFSM.candidateHandler(candidate),
		leaderState:    nodeFSM.leaderHandler(leader),
	}

	dispatcher.Subscribe(nodeFSM.eventCh) // TODO move this to start method and have an Unsubscribe function when stopping

	return nodeFSM
}

func (this *NodeFSM) candidateHandler(candidate *Candidate) stateHandler {
	return this.newStateHandler(
		candidateState,
		candidate,
		func(e common.Event) state {
			switch e.EventType {
			case common.QuorumObtained:
				return leaderState
			case common.QuorumUnobtained:
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
}

func (this *NodeFSM) leaderHandler(leader *Leader) stateHandler {
	return this.newStateHandler(
		leaderState,
		leader,
		func(e common.Event) state {
			return invalidState
		},
		func(rpcCtx rpcContext) state {
			return invalidState
		})
}

func (this *NodeFSM) followerHandler(follower *Follower) stateHandler {
	return this.newStateHandler(
		followerState,
		follower,
		func(e common.Event) state {
			switch e.EventType {
			case common.LeaderKeepAliveTimeout:
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
}

// Helper functions that reads from the event channel and rpc channel.
// Handles the term number encountered so that the passed in functions need not
// be concerned about checking the term number to determine whether or not it can handle it.
func (n *NodeFSM) newStateHandler(s state, service common.Service,
	eFn eventFn, rFn rpcFn) stateHandler {
	return stateHandler{
		service:     service,
		handleEvent: n.commonEventHandler(s, eFn),
		handleRpc:   n.commonRpcHandler(s, rFn),
	}
}

func (n *NodeFSM) commonEventHandler(s state, fn eventFn) eventFn {
	noop := func() {}
	return func(e common.Event) state {
		return n.commonTermStateHandler(s,
			e.Term,
			noop,
			noop,
			func() state { return fn(e) },
			noop)
	}
}

func (this *NodeFSM) commonRpcHandler(s state, fn rpcFn) rpcFn {
	return func(rpcCtx rpcContext) state {
		return this.commonTermStateHandler(s,
			rpcCtx.term,
			func() {
				currentTerm := this.stateStore.CurrentTerm()
				errorMsg := fmt.Sprint("Old Term: %v Current Term: %v", rpcCtx.term, currentTerm)
				rpcCtx.errorChan <- errors.New(errorMsg)
			},
			func() { this.rpcCh <- rpcCtx }, // replay by adding it back to the rpc channel after it has been transitioned to follower
			func() state { return fn(rpcCtx) },
			func() {
				errorMsg := fmt.Sprintf("Can't handle while in %v state", this.currentState)
				rpcCtx.errorChan <- errors.New(errorMsg)
			})
	}
}

func (this *NodeFSM) commonTermStateHandler(s state, term uint32,
	lt func(), gt func(), eq func() state,
	invalidFn func()) state {

	currentTerm := this.stateStore.CurrentTerm()
	nextState := s

	switch {
	case term > currentTerm:
		this.newTermCh <- term

		if this.currentState != followerState { // revert to follower if higher term seen for leader or candidate
			gt()
			nextState = followerState
		}

	case term < currentTerm:
		lt()

	default:
		eqNextState := eq()
		if nextState == invalidState {
			invalidFn()
		} else {
			nextState = eqNextState
		}
	}

	return nextState
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
		case e := <-this.eventCh:
			log.Debug("Received event:", e)
			this.process(func(h stateHandler) state {
				return h.handleEvent(e)
			})

		case rpcCtx := <-this.rpcCh:
			log.Debug("Received rpc:", rpcCtx.rpc)
			this.process(func(h stateHandler) state {
				return h.handleRpc(rpcCtx)
			})
		}
	}
}

func (this *NodeFSM) process(fn func(stateHandler) state) {
	currentState, currentStateHandler := this.getCurrent()
	log.Debug("In current state: ", currentState)

	nextState := fn(currentStateHandler)

	if nextState != currentState {
		currentStateHandler.service.Stop()
		log.Debug("Transitioned to ", nextState)

		this.storeNewTerm()

		this.currentState = nextState
		this.fsm[nextState].service.Start()
	}
}

func (this *NodeFSM) syncStop() error {
	log.Debug("Shutting down node fsm")

	_, stateHandler := this.getCurrent()
	err := stateHandler.service.Stop()
	log.Debug("Shut down state handler complete")

	close(this.stopCh)

	return err
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
