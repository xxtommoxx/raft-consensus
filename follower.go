package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"math/rand"
	"sync"
	"time"
)

const (
	timerStop = iota
	timerReset
)

type FollowerListener interface {
	OnKeepAliveTimeout(term uint32)
}

type noopFollowerListener struct{}

func (n *noopFollowerListener) OnKeepAliveTimeout(term uint32) {}

type Follower struct {
	timeout      LeaderTimeout
	listener     FollowerListener
	timeoutRange int64
	keepAlive    chan int
	random       *rand.Rand

	stateStore StateStore
	mutex      *sync.Mutex
}

func NewFollower(stateStore StateStore, timeout LeaderTimeout) *Follower {
	return &Follower{
		listener:   &noopFollowerListener{},
		stateStore: stateStore,
		timeout:    timeout,
	}
}

func (h *Follower) SetListener(listener FollowerListener) {
	h.listener = listener
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
				if timerEvent == timerStop {
					fmt.Println("Terminating leader election timer")
					timer.Stop()
					return
				}

				if timerEvent == timerReset {
					duration := h.leaderTimeout()
					if !timer.Reset(duration) {
						timer = time.NewTimer(duration)
					}
				}
			/*
				a race condition is possible whereby a request was just
				received before the timer just expired but it was scheduled out before it had a chance to
				send Reset. This window is small if the timeout is >> than the time between new requests.
				It is assumed that each request is handled in a go routine.
			*/
			case <-timer.C:
				h.listener.OnKeepAliveTimeout(h.stateStore.CurrentTerm())
			}
		}
	}()

	return nil
}

func (h *Follower) Stop() error {
	if h.keepAlive != nil {
		h.keepAlive <- timerStop
	}

	return nil
}

func (h *Follower) resetTimer() {
	h.keepAlive <- timerReset
}

func (this *Follower) RequestVote(req *rpc.VoteRequest) (bool, error) {
	this.resetTimer()

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
