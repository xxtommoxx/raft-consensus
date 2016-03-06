package raft

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"math/rand"
	"sync"
	"time"
)

type Follower struct {
	*common.SyncService

	timeout  common.LeaderTimeout
	listener common.EventListener
	random   *rand.Rand

	resetTimerCh chan struct{}
	stopCh       chan struct{}

	stateStore StateStore
	voteMutex  sync.Mutex
}

func NewFollower(stateStore StateStore, listener common.EventListener, timeout common.LeaderTimeout) *Follower {
	follower := &Follower{
		listener:   listener,
		stateStore: stateStore,
		timeout:    timeout,
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	follower.SyncService = common.NewSyncService(follower.syncStart, follower.startBackGroundTimer, follower.syncStop)

	return follower
}

func (f *Follower) syncStart() error {
	f.resetTimerCh = make(chan struct{})
	f.stopCh = make(chan struct{})

	return nil
}

func (f *Follower) startBackGroundTimer() {
	timer := time.NewTimer(f.leaderTimeout())

	for {
		select {
		case <-f.stopCh:
			timer.Stop()
			close(f.resetTimerCh)
			return
		case <-f.resetTimerCh:
			duration := f.leaderTimeout()
			if !timer.Reset(duration) {
				timer = time.NewTimer(duration)
			}
		case <-timer.C:
			if f.Status() == common.Started {
				log.Debug("Leader timer expired")
				f.listener.HandleEvent(common.Event{Term: f.stateStore.CurrentTerm(), EventType: common.LeaderKeepAliveTimeout})
			}
		}
	}
}

func (h *Follower) leaderTimeout() time.Duration {
	timeoutRange := h.timeout.MaxMillis - h.timeout.MinMillis + 1
	timeout := time.Duration(h.random.Int63n(timeoutRange)+h.timeout.MinMillis) * time.Millisecond
	log.Debug("Using leader timeout:", timeout)
	return timeout
}

func (f *Follower) syncStop() error {
	close(f.stopCh)
	return nil
}

func (f *Follower) resetTimer() {
	log.Debug("Reseting leader timer")
	f.resetTimerCh <- struct{}{}
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
