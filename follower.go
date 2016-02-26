package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/common"
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

func (noopFollowerListener) OnKeepAliveTimeout(term uint32) {}

type Follower struct {
	*common.SyncService

	timeout      common.LeaderTimeout
	listener     FollowerListener
	timeoutRange int64
	keepAlive    chan int
	random       rand.Rand
	requestCount uint64

	stateStore StateStore
	voteMutex  sync.Mutex
}

func NewFollower(stateStore StateStore, timeout common.LeaderTimeout) *Follower {
	follower := &Follower{
		listener:   noopFollowerListener{},
		stateStore: stateStore,
		timeout:    timeout,
	}

	follower.SyncService = common.NewSyncService(follower.syncStart, follower.startBackGroundTimer, follower.syncStop)

	return follower
}

func (h *Follower) SetListener(listener FollowerListener) {
	h.listener = listener
}

func (f *Follower) syncStart() error {
	fmt.Println("Starting follower")
	return nil
}

func (f *Follower) startBackGroundTimer() {
	fmt.Println("Starting follower timer")

	f.keepAlive = make(chan int)
	defer close(f.keepAlive)

	timer := time.NewTimer(f.leaderTimeout())

	var reqCountSinceReset uint64 = atomic.LoadUint64(&f.requestCount)

	for {
		select {
		case timerEvent := <-f.keepAlive:
			if timerEvent == timerStop {
				fmt.Println("Terminating leader election timer")
				timer.Stop()
				return
			}

			if timerEvent == timerReset {
				reqCountSinceReset = atomic.LoadUint64(&f.requestCount)

				duration := f.leaderTimeout()
				if !timer.Reset(duration) {
					timer = time.NewTimer(duration)
				}
			}
		case <-timer.C:
			f.WithMutex(func() {
				if f.Status == common.Started && reqCountSinceReset == f.requestCount {
					f.listener.OnKeepAliveTimeout(f.stateStore.CurrentTerm())
				}
			})
		}
	}
}

func (h *Follower) leaderTimeout() time.Duration {
	return time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
}

func (f *Follower) syncStop() error {
	f.keepAlive <- timerStop
	f.requestCount = 0
	return nil
}

func (f *Follower) resetTimer() {
	f.WithMutex(func() {
		f.requestCount++
		f.keepAlive <- timerReset
	})
}

func (f *Follower) KeepAliveRequest(req *rpc.KeepAliveRequest) {
	f.resetTimer()
}

func (f *Follower) RequestVote(req *rpc.VoteRequest) (bool, error) {
	f.resetTimer()

	votedFor := f.stateStore.VotedFor()
	if votedFor == nil || votedFor.Term < req.Term || (votedFor.Term == req.Term && votedFor.NodeId == req.Id) {
		vote := Vote{req.Term, req.Id}
		f.stateStore.SaveVote(&vote)
		return true, nil
	}
	return false, nil
}
