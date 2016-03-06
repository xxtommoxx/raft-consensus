package raft

import (
	log "github.com/Sirupsen/logrus"
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

type Follower struct {
	*common.SyncService

	timeout      common.LeaderTimeout
	listener     common.EventListener
	keepAlive    chan int
	random       *rand.Rand
	requestCount uint64

	stateStore StateStore
	voteMutex  sync.Mutex
}

func NewFollower(stateStore StateStore, listener common.EventListener, timeout common.LeaderTimeout) *Follower {
	follower := &Follower{
		listener:   listener,
		stateStore: stateStore,
		timeout:    timeout,
		keepAlive:  make(chan int),
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	follower.SyncService = common.NewSyncService(follower.syncStart, follower.startBackGroundTimer, follower.syncStop)

	return follower
}

func (f *Follower) syncStart() error {
	return nil
}

func (f *Follower) startBackGroundTimer() {
	f.keepAlive = make(chan int)
	defer close(f.keepAlive)

	timer := time.NewTimer(f.leaderTimeout())

	var reqCountSinceReset uint64 = atomic.LoadUint64(&f.requestCount)

	for {
		select {
		case timerEvent := <-f.keepAlive:
			if timerEvent == timerStop {
				log.Debug("Stopping leader timer")
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
				if f.Status() == common.Started && reqCountSinceReset == f.requestCount {
					log.Debug("Leader timer expired")
					f.listener.HandleEvent(common.Event{Term: f.stateStore.CurrentTerm(), EventType: common.LeaderKeepAliveTimeout})
				}
			})
		}
	}
}

func (h *Follower) leaderTimeout() time.Duration {
	timeoutRange := h.timeout.MaxMillis - h.timeout.MinMillis + 1
	timeout := time.Duration(h.random.Int63n(timeoutRange)+h.timeout.MinMillis) * time.Millisecond
	log.Debug("Leader timeout:", timeout)
	return timeout
}

func (f *Follower) syncStop() error {
	f.keepAlive <- timerStop
	f.requestCount = 0
	return nil
}

func (f *Follower) resetTimer() {
	f.WithMutex(func() {
		log.Debug("Reseting leader timer")
		f.requestCount++
		f.keepAlive <- timerReset
	})
}

func (f *Follower) KeepAliveRequest(req *rpc.KeepAliveRequest) (*rpc.KeepAliveResponse, error) {
	f.resetTimer()
	return &rpc.KeepAliveResponse{Term: f.stateStore.CurrentTerm()}, nil
}

func (f *Follower) RequestVote(req *rpc.VoteRequest) (bool, error) {
	f.resetTimer()

	log.Debug("Vote request received:", req)

	votedFor := f.stateStore.VotedFor()
	if votedFor == nil || votedFor.Term < req.Term || (votedFor.Term == req.Term && votedFor.NodeId == req.Id) {
		vote := Vote{req.Term, req.Id}
		f.stateStore.SaveVote(&vote)
		log.Debug("Granted vote for:", req)
		return true, nil
	}
	return false, nil
}
