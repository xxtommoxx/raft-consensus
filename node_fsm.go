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
		return "leaderState"
	case followerState:
		return "followerState"
	case candidateState:
		return "candidateState"
	case invalidState:
		return "invalidState"
	}

	return "Unknown state"
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
**/
type NodeFSM struct {
	currentState state

	fsm map[state]stateHandler

	rpcCh   chan rpcContext
	eventCh chan common.Event

	termResponseCh chan uint32
	newTermCh      chan uint32

	stopCh chan struct{}

	stateStore StateStore
	dispatcher *common.EventListenerDispatcher

	*common.SyncService
}

func NewNodeFSM(stateStore StateStore, dispatcher *common.EventListenerDispatcher,
	follower *Follower, candidate *Candidate, leader *Leader) *NodeFSM {

	nodeFSM := &NodeFSM{
		currentState: followerState,

		stateStore: stateStore,
		dispatcher: dispatcher,

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

	return nodeFSM
}

func (n *NodeFSM) candidateHandler(candidate *Candidate) stateHandler {
	return n.newStateHandler(
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
			return invalidState
		})
}

func (n *NodeFSM) leaderHandler(leader *Leader) stateHandler {
	return n.newStateHandler(
		leaderState,
		leader,
		func(e common.Event) state {
			return invalidState
		},
		func(rpcCtx rpcContext) state {
			return invalidState
		})
}

func (n *NodeFSM) followerHandler(follower *Follower) stateHandler {
	return n.newStateHandler(
		followerState,
		follower,
		func(e common.Event) state {
			switch e.EventType {
			case common.LeaderKeepAliveTimeout:
				n.newTermCh <- n.stateStore.CurrentTerm() + 1
				return candidateState
			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch req := rpcCtx.rpc.(type) {
			case *rpc.VoteRequest:
				n.processAsync(rpcCtx, func() (interface{}, error) { return follower.RequestVote(req) })
				return followerState
			case *rpc.KeepAliveRequest:
				n.processAsync(rpcCtx, func() (interface{}, error) { return follower.KeepAliveRequest(req) })
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

func (n *NodeFSM) commonRpcHandler(s state, fn rpcFn) rpcFn {
	return func(rpcCtx rpcContext) state {
		return n.commonTermStateHandler(s,
			rpcCtx.term,
			func() {
				currentTerm := n.stateStore.CurrentTerm()
				errorMsg := fmt.Sprint("Old Term: %v Current Term: %v", rpcCtx.term, currentTerm)
				rpcCtx.errorChan <- errors.New(errorMsg)
			},
			func() { n.rpcCh <- rpcCtx }, // replay by adding it back to the rpc channel after it has been transitioned to follower
			func() state { return fn(rpcCtx) },
			func() {
				errorMsg := fmt.Sprintf("Can't handle %v while in %v state", rpcCtx.rpc, n.currentState)
				rpcCtx.errorChan <- errors.New(errorMsg)
			})
	}
}

func (n *NodeFSM) commonTermStateHandler(s state, term uint32,
	lt func(), gt func(), eq func() state,
	invalidFn func()) state {

	currentTerm := n.stateStore.CurrentTerm()
	nextState := s

	switch {
	case term > currentTerm:
		n.newTermCh <- term

		if n.currentState != followerState { // revert to follower if higher term seen for leader or candidate
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

func (n *NodeFSM) syncStart() error {
	log.Info("Starting node fsm")
	n.currentState = followerState

	_, currentStateHandler := n.getCurrent()

	currentStateHandler.service.Start()

	n.stopCh = make(chan struct{})
	n.dispatcher.Subscribe(n.eventCh)

	return nil
}

func (n *NodeFSM) asyncStart() {
	for {
		select {
		case <-n.stopCh:
			log.Debug("Shutting down node fsm async start")
			n.storeNewTerm()
			return
		case e := <-n.eventCh:
			log.Debug("Received event:", e)
			n.process(func(h stateHandler) state {
				return h.handleEvent(e)
			})

		case rpcCtx := <-n.rpcCh:
			log.Debug("Received rpc:", rpcCtx.rpc)
			n.process(func(h stateHandler) state {
				return h.handleRpc(rpcCtx)
			})
		}
	}
}

func (n *NodeFSM) process(fn func(stateHandler) state) {
	currentState, currentStateHandler := n.getCurrent()
	log.Debug("In current state: ", currentState)

	nextState := fn(currentStateHandler)

	if nextState != currentState {
		currentStateHandler.service.Stop()
		log.Debug("Transitioned to ", nextState)

		n.storeNewTerm()

		n.currentState = nextState
		n.fsm[nextState].service.Start()
	}
}

func (n *NodeFSM) syncStop() error {
	log.Debug("Shutting down node fsm")

	_, stateHandler := n.getCurrent()
	err := stateHandler.service.Stop()
	log.Debug("Shut down state handler complete")

	close(n.stopCh)
	n.dispatcher.Unsubscribe(n.eventCh)

	return err
}

func (n *NodeFSM) storeNewTerm() {
	// see if any greater term occurred while processing a request
	select {
	case newTerm := <-n.newTermCh:
		n.stateStore.SaveCurrentTerm(newTerm)
	default:
	}
}

func (n *NodeFSM) getCurrent() (state, stateHandler) {
	return n.currentState, n.fsm[n.currentState]
}

func (n *NodeFSM) processAsync(ctx rpcContext, fn func() (interface{}, error)) {
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

func (n *NodeFSM) RequestVote(vote *rpc.VoteRequest) (<-chan *rpc.VoteResponse, <-chan error) {
	respCh := make(chan *rpc.VoteResponse)
	errorCh := n.sendRpcRequest(vote.Term, vote, respCh)
	return respCh, errorCh
}
func (n *NodeFSM) KeepAlive(req *rpc.KeepAliveRequest) (<-chan *rpc.KeepAliveResponse, <-chan error) {
	respCh := make(chan *rpc.KeepAliveResponse)
	errorCh := n.sendRpcRequest(req.LeaderInfo.Term, req, respCh)
	return respCh, errorCh
}

func (n *NodeFSM) sendRpcRequest(term uint32, rpc interface{}, responseChan interface{}) <-chan error {
	errorCh := make(chan error)

	go func() {
		n.rpcCh <- rpcContext{
			term:         term,
			rpc:          rpc,
			errorChan:    make(chan error),
			responseChan: common.ToForwardedChan(responseChan),
		}
	}()

	return errorCh
}
