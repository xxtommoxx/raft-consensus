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

type KeepAliveTimeout struct {
	term           uint32
	waitDurationMs uint64
}

type FollowerListener interface {
	OnKeepAliveTimeout(term uint32)
}

type Follower struct {
	timeout      LeaderTimeout
	Listener     FollowerListener
	timeoutRange int64
	keepAlive    chan int
	random       *rand.Rand

	votedForId   string
	votedForTerm uint32

	mutex *sync.Mutex
}

func NewInstance() *Follower {
	return &Follower{}
}

func (h *Follower) newTimer() *Timer {
	duration := time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
	return time.AfterFunc(duration, func() {
		if h.Listener != nil {
			// h.Listener.OnKeepAliveTimeout(h.
		}
	})
}

func (h *Follower) Start(term uint32) error {
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
				send reset
			*/
			case <-timer.C:
				if h.Listener != nil {
					h.Listener.OnKeepAliveTimeout(&KeepAliveTimeout(term))
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

func (this *Follower) requestVote(req *rpc.VoteRequest) (bool, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.votedForTerm < req.Term {
		// todo store this
		this.votedForTerm = req.Term
		this.votedForId = req.CandidateId
		return true, nil
	}

	return false, nil
}
