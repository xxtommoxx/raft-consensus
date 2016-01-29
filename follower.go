package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"math/rand"
	"sync"
	"time"
)

const (
	Close = iota
	Reset
)

type FollowerListener interface {
	OnKeepAliveTimeout(term uint32)
}

type Follower struct {
	timeout      LeaderTimeout
	Listener     FollowerListener
	timeoutRange int64
	keepAlive    chan int
	random       *rand.Rand

	stateStore StateStore
	mutex      *sync.Mutex
}

func NewInstance(stateStore StateStore) *Follower {
	return &Follower{
		stateStore: stateStore,
	}
}

func (h *Follower) leaderTimeout() time.Duration {
	return time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
}

func (h *Follower) Start() error {
	fmt.Println("Starting follower")

	go func() {
		h.keepAlive = make(chan int)
		defer close(h.keepAlive)

		timer := time.NewTimer(h.leaderTimeout())
		defer timer.Stop()

		for {
			select {
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
			/*
				TODO: a race condition is possible whereby a request was just
				received before the timer just expired but it was scheduled out before it had a chance to
				send Reset
			*/
			case <-timer.C:
				if h.Listener != nil {
					h.Listener.OnKeepAliveTimeout(h.stateStore.CurrentTerm())
				}
			}
		}
	}()

	return nil
}

func (h *Follower) Stop() error {
	if h.keepAlive != nil {
		h.keepAlive <- Close
	}

	return nil
}

func (this *Follower) RequestVote(req *rpc.VoteRequest) (bool, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	votedFor := this.stateStore.VotedFor()
	if votedFor == nil || votedFor.Term < req.Term {
		vote := &Vote{req.Term, req.CandidateId}
		this.stateStore.SaveVote(vote)
		return true, nil
	}
	return false, nil
}
