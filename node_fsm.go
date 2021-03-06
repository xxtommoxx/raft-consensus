package raft

import (
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"reflect"
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
	req       rpc.Request
	respCh    reflect.Value
	respErrCh chan error
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

	stopCh chan struct{}

	stateStore common.StateStore
	dispatcher *common.EventListenerDispatcher

	*common.SyncService

	log *logrus.Entry
}

func NewNodeFSM(stateStore common.StateStore, dispatcher *common.EventListenerDispatcher,
	follower *Follower, candidate *Candidate, leader *Leader, id string) *NodeFSM {

	nodeFSM := &NodeFSM{
		currentState: followerState,

		stateStore: stateStore,
		dispatcher: dispatcher,

		rpcCh:   make(chan rpcContext),
		eventCh: make(chan common.Event),

		log: logrus.WithFields(logrus.Fields{
			"id": id,
		}),
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
				n.log.Println("Quorum for being leader obtained")
				return leaderState

			case common.QuorumUnobtained:
				n.log.Println("Quorum for being leader unobtained")
				return followerState

			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch rpcCtx.req.(type) {
			case *rpc.VoteRequest:
				n.processAsync(rpcCtx, func() (interface{}, error) {
					return false, nil
				})
				return candidateState

			default:
				return invalidState
			}
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
				n.log.Info("Leader timer expired beginning vote...")
				newTerm := n.stateStore.CurrentTerm() + 1
				n.stateStore.SaveCurrentTerm(newTerm)
				return candidateState

			default:
				return invalidState
			}
		},
		func(rpcCtx rpcContext) state {
			switch req := rpcCtx.req.(type) {
			case *rpc.VoteRequest:
				n.processAsync(rpcCtx, func() (interface{}, error) { return follower.RequestVote(req) })
				return followerState

			case *rpc.KeepAliveRequest:
				n.processAsync(rpcCtx, func() (interface{}, error) { return nil, follower.KeepAliveRequest(req) })
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
			func() state {
				switch e.EventType {
				case common.ResponseReceived:
					return s

				default:
					return fn(e)
				}
			},

			noop)
	}
}

func (n *NodeFSM) commonRpcHandler(s state, fn rpcFn) rpcFn {
	return func(rpcCtx rpcContext) state {
		return n.commonTermStateHandler(s,
			rpcCtx.req.Term(),
			func() {
				currentTerm := n.stateStore.CurrentTerm()
				errorMsg := fmt.Sprintf("Old Term: %v Current Term: %v", rpcCtx.req.Term(), currentTerm)
				rpcCtx.respErrCh <- errors.New(errorMsg)
			},
			func() { // replay by adding it back to the rpc channel after it has been transitioned to follower
				go func() { n.rpcCh <- rpcCtx }()
			},
			func() state { return fn(rpcCtx) },

			func() {
				errorMsg := fmt.Sprintf("Can't handle %v while in %v state", rpcCtx.req, n.currentState)
				rpcCtx.respErrCh <- errors.New(errorMsg)
			})
	}
}

func (n *NodeFSM) commonTermStateHandler(s state, term uint32,
	lt func(), gt func(), eq func() state,
	invalidFn func()) state {

	currentTerm := n.stateStore.CurrentTerm()
	nextState := s

	switch {
	case term < currentTerm:
		lt()

	case term > currentTerm:
		n.stateStore.SaveCurrentTerm(term)

		gt()
		nextState = followerState

	default:
		eqNextState := eq()
		if eqNextState == invalidState {
			invalidFn()
		} else {
			nextState = eqNextState
		}
	}

	return nextState
}

func (n *NodeFSM) syncStart() error {
	n.currentState = followerState

	n.log.Info("Starting NodeFSM in:", n.currentState) // TODO retrive the state of fsm from state store

	_, currentStateHandler := n.getCurrent()

	currentStateHandler.service.Start()

	n.stopCh = make(chan struct{})
	n.dispatcher.Subscribe(n.eventCh)

	n.log.Info("Start NodeFSM complete")

	return nil
}

func (n *NodeFSM) asyncStart() {
	for {
		select {
		case <-n.stopCh:
			n.log.Debug("Shutting down NodeFSM asyncStart")
			return

		case e := <-n.eventCh:
			n.log.Debugf("Received event: %+v", e)
			n.process(func(h stateHandler) state {
				return h.handleEvent(e)
			})

		case rpcCtx := <-n.rpcCh:
			n.log.Debugf("Received rpc request: %v", rpcCtx.req)
			n.process(func(h stateHandler) state {
				k := h.handleRpc(rpcCtx)
				return k
			})
		}
	}
}

func (n *NodeFSM) process(fn func(stateHandler) state) {
	currentState, currentStateHandler := n.getCurrent()

	n.log.Debugf("In current state: %v", currentState)

	nextState := fn(currentStateHandler)

	n.log.Debugf("Next state: %v", nextState)

	if nextState != currentState {
		currentStateHandler.service.Stop()
		n.log.Info("Transitioned to ", nextState)
		n.currentState = nextState
		n.fsm[nextState].service.Start()
	}
}

func (n *NodeFSM) syncStop() error {
	n.log.Info("Shutting down NodeFSM")

	_, stateHandler := n.getCurrent()
	err := stateHandler.service.Stop()
	n.log.Debug("Shut down stateHandler complete")

	close(n.stopCh)
	n.dispatcher.Unsubscribe(n.eventCh)

	n.log.Info("Shutdown NodeFSM complete")

	return err
}

func (n *NodeFSM) getCurrent() (state, stateHandler) {
	return n.currentState, n.fsm[n.currentState]
}

func (n *NodeFSM) processAsync(ctx rpcContext, fn func() (interface{}, error)) {
	go func() {
		defer close(ctx.respErrCh)
		defer ctx.respCh.Close()

		result, err := fn()

		if err != nil {
			n.log.Debugf("Sending error response: %v", err)
			ctx.respErrCh <- err
		} else {
			if result != nil {
				n.log.Debugf("Sending response: %v", result)
				ctx.respCh.Send(reflect.ValueOf(result))
			}
		}
	}()
}

func (n *NodeFSM) RequestVote(vote *rpc.VoteRequest) (<-chan bool, <-chan error) {
	respCh := make(chan bool)
	errorCh := n.sendRpcRequest(vote, respCh)
	return respCh, errorCh
}

func (n *NodeFSM) KeepAlive(req *rpc.KeepAliveRequest) (<-chan struct{}, <-chan error) {
	respCh := make(chan struct{})
	errorCh := n.sendRpcRequest(req, respCh)
	return respCh, errorCh
}

func (n *NodeFSM) sendRpcRequest(req rpc.Request, responseChan interface{}) <-chan error {
	errCh := make(chan error)

	go func() {
		n.rpcCh <- rpcContext{
			req:       req,
			respCh:    reflect.ValueOf(responseChan),
			respErrCh: errCh,
		}
	}()

	return errCh
}
