package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"github.com/xxtommoxx/raft-consensus/util/fsm"
	"math/rand"
	"sync"
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

type NodeFSM struct {
}

type FollowerListener interface {
	KeepAliveTimeout(currentTerm uint32)
}

/*********** CANDIDATE HANDLER **************/

// type QuorumStrategy interface {
// 	obtained(currentCount uint32) bool
// }
//
// type CandidateHandler struct {
// 	term         uint32
// 	voteReceived uint32
// 	quorum       QuorumStrategy
// }
//
// func (h *CandidateHandler) start(term uint32) {
// 	h.term = term
// }
//
// func (h *CandidateHandler) Handle(e interface{}) (fsm.State, bool) {
// 	switch t := e.(type) {
// 	default:
// 		return fsm.InvalidTransition()
// 	case VoteResponse:
// 		return fsm.Transition(Candidate)
// 	case rpc.KeepAliveResponse:
// 		h.keepAlive <- Reset
// 		return fsm.Transition(Follower)
// 	}
// }

/*********** FOLLOWER HANDLER **************/

/* Timer state */
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

	lifeCycleMutex *sync.Mutex

	listener FollowerListener
}

func NewInstance() *FollowerHandler {
	//timeoutRange := h.timeout.MaxMillis - h.timeout.MinMillis + 1
	return &FollowerHandler{}
}

func (h *FollowerHandler) leaderTimeout() time.Duration {
	return time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
}

func (h *FollowerHandler) Start() {
	fmt.Println("Starting follower handler")

	go func() {
		h.keepAlive = make(chan int)
		defer close(h.keepAlive)

		timer := time.NewTimer(h.leaderTimeout())

		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				h.listener.KeepAliveTimeout()
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

func (h *FollowerHandler) Handle(e interface{}) (fsm.State, bool) {
	switch e.(type) {
	default:
		return fsm.InvalidTransition()
	case KeepAliveTimeout:
		return fsm.Transition(Candidate)
	case rpc.KeepAliveResponse:
		h.keepAlive <- Reset
		return fsm.Transition(Follower)
	}
}
