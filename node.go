package raft

import (
	"fmt"
	"math/rand"
	"time"
)

type Id string

type Location struct {
	id   Id
	host string
	port uint32
}

type Config struct {
	MinElectionTimeoutMillis   int
	MaxElectionTimeoutMaxMllis int
	peers                      []Location
}

type Node struct {
	config Config
}

func NewNode(config Config, stateStore *StateStore) *Node {
	return &Node{config: config}
}

/*********** NODE FSM **************/

const (
	Leader = iota
	Follower
	Candidate
)

// messages
type KeepAliveTimeout struct{}

type GenericEventHandler interface {
	handle(event interface{})
}

// type TermHandler struct {
// 	eventHandler *EventHandler
// }
//
// func (h *TermHandler) start() {}
//
// func (h *TermHandler) stop() {}
//
// func (h *TermHandler) Handle(currentTerm uint32, eventTerm uint32, e interface{}) (fsm.State, bool) {
// 	if eventTerm > currentTerm {
// 		return fsm.Transition(FOLLOWER)
// 	} else {
// 		h.eventHandler.Handle(currentTerm, eventTerm, e)
// 	}
// }
//
type FollowerListener interface {
	KeepAliveTimeout(term uint32)
}

type Promise struct {
	result interface{}
	err    error

	done chan struct{}
}

func NewPromise(f func() (interface{}, error)) *Promise {

	done := make(chan struct{})

	promise := &Promise{done: done}

	go func() {
		defer close(done)

		result, err := f()
		promise.result = result
		promise.err = err
	}()

	return promise
}

func (p *Promise) Get() (interface{}, error) {
	<-p.done
	return p.result, p.err
}

type State int

const (
	NCandidate State = iota
	NFollower
	NLeader
	NInvalid
)

type NodeFSM struct {
	current State
	maxTerm uint32

	fsm map[State]internal
}

type Lifecycle interface {
	Stop() error
	Start(term uint32) error
}

type internal struct {
	c            chan interface{}
	lifecycle    Lifecycle
	stateHandler func() State
}

func NewNodeFSM(candidate *CandidateHandler) *NodeFSM {

	nodeFSM := &NodeFSM{}

	fsm := map[State]internal{}

	nodeFSM.fsm = fsm

	return nodeFSM
}

func (this *NodeFSM) start() {
	go func() {
		for {
			currentI := this.fsm[this.current]

			nextState := currentI.stateHandler()

			if nextState != this.current {
				currentI.lifecycle.Stop()
				this.fsm[nextState].lifecycle.Start(this.maxTerm)
				this.current = nextState
			}

		}
	}()
}

func (this *NodeFSM) sendRequest(reqTerm uint32, fn func(chan interface{})) error {

	if reqTerm >= this.maxTerm {
		this.maxTerm = reqTerm

		stateChan := this.fsm[this.current].c
		fn(stateChan)
	} else {
		// return an error indicating failed
	}

	return nil
}

func (this *NodeFSM) makeFollowerState() *internal {
	followerChan := make(chan interface{})

	stateHandler := func() State {
		req := <-followerChan

		switch r := req.(type) {
		default:
			fmt.Println(r)
			return NFollower
		}

		return Candidate
	}

	return &internal{c: followerChan, lifecycle: nil, stateHandler: stateHandler}
}

/*


create a promise --> promise will wait for channels response
1) get the current state's channel
2) create a promise to put it on the chan

 one go routine for looper that puts stuff on the chan

state needs a chan for publishing events and a select function that returns next state

for {

	handleChan()
want a function that returns the state after selecting


}


 select followerchan{
	case <-----------`bleh


 }



*/

func (this *NodeFSM) candidate() {

}

func (this *NodeFSM) requestVote() *Promise {
	return nil
}

type RequestVote struct {
	term uint32
}

func (this *NodeFSM) onTimeout(timeout KeepAliveTimeout) {
}

// follower code

const (
	Close = iota
	Reset
)

type LeaderTimeout struct {
	MaxMillis int64
	MinMillis int64
}

type FollowerHandler struct {
	timeout      LeaderTimeout
	timeoutRange int64
	keepAlive    chan int
	random       *rand.Rand
}

func NewInstance() *FollowerHandler {
	return &FollowerHandler{}
}

func (h *FollowerHandler) leaderTimeout() time.Duration {
	return time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
}

func (h *FollowerHandler) Start() {
	fmt.Println("Starting follower")

	go func() {
		h.keepAlive = make(chan int)
		defer close(h.keepAlive)

		timer := time.NewTimer(h.leaderTimeout())

		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				// h.listener.KeepAliveTimeout(h.currentTerm) TODO think about whether we need to pass currentTerm
			case timerEvent := <-h.keepAlive:
				if timerEvent == Close {
					fmt.Println("Terminating leader election timer")
					timer.Stop()
					return
				}

				if timerEvent == Reset {
					duration := h.leaderTimeout()
					if !timer.Reset(duration) {
						timer = time.NewTimer(duration)
					}
				}
			}
		}
	}()
}

func (h *FollowerHandler) Stop() {
	if h.keepAlive != nil {
		h.keepAlive <- Close
	}
}

// candidate code

type QuorumStrategy interface {
	obtained(currentCount uint32) bool
}

type Client interface {
	sendRequestVote() <-chan VoteResponse
}

type VoteResponse struct{}

type BaseEvent struct {
	term uint32
}

type CandidateHandler struct {
	quorum   QuorumStrategy
	listener CandidateListener
	client   Client
}

type CandidateListener interface {
	quorumObtained(term uint32)
}

func (h *CandidateHandler) start(term uint32) error {

	var currentVoteCount uint32 = 0

	go func() {
		responseChan := h.client.sendRequestVote()
		for {
			select {
			case <-responseChan:
				currentVoteCount++
				if h.quorum.obtained(currentVoteCount) {
					// todo close responseChan
					h.listener.quorumObtained(term)
				}
			}
		}
	}()

	return nil
}

func (h *CandidateHandler) stop() error {
	return nil
}
