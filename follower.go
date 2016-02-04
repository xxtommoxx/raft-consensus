package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"math/rand"
	"sync"
	"sync/atomic"
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

func (n noopFollowerListener) OnKeepAliveTimeout(term uint32) {}

type Follower struct {
	timeout      LeaderTimeout
	listener     FollowerListener
	timeoutRange int64
	keepAlive    chan int
	random       rand.Rand
	isStopping   bool
	requestCount uint64

	stateStore StateStore
	voteMutex  sync.Mutex
	timerMutex sync.Mutex
}

func NewFollower(stateStore StateStore, timeout LeaderTimeout) *Follower {
	return &Follower{
		listener:   noopFollowerListener{},
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

		var reqCountSinceReset uint64 = atomic.LoadUint64(&h.requestCount)

		for {
			select {
			case timerEvent := <-h.keepAlive:
				if timerEvent == timerStop {
					fmt.Println("Terminating leader election timer")
					timer.Stop()
					return
				}

				if timerEvent == timerReset {
					reqCountSinceReset = atomic.LoadUint64(&h.requestCount)

					duration := h.leaderTimeout()
					if !timer.Reset(duration) {
						timer = time.NewTimer(duration)
					}
				}
			case <-timer.C:
				h.withTimerLock(func() {
					if !h.isStopping && reqCountSinceReset == h.requestCount {
						h.listener.OnKeepAliveTimeout(h.stateStore.CurrentTerm())
					}
				})
			}
		}
	}()

	return nil
}

func (h *Follower) Stop() error {
	h.withTimerLock(func() {
		h.isStopping = true
		h.keepAlive <- timerStop
		h.requestCount = 0
		h.isStopping = false
	})
	return nil
}

func (h *Follower) resetTimer() {
	h.withTimerLock(func() {
		h.requestCount++
		h.keepAlive <- timerReset
	})
}

func (h *Follower) withTimerLock(fn func()) {
	h.timerMutex.Lock()
	h.timerMutex.Unlock()
	fn()
}

func (this *Follower) RequestVote(req *rpc.VoteRequest) (bool, error) {
	this.resetTimer()

	this.voteMutex.Lock()
	defer this.voteMutex.Unlock()

	votedFor := this.stateStore.VotedFor()
	if votedFor == nil || votedFor.Term < req.Term {
		vote := Vote{req.Term, req.CandidateId}
		this.stateStore.SaveVote(&vote)
		return true, nil
	}
	return false, nil
}
