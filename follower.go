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

	stateStore common.StateStore
	voteMutex  sync.Mutex
}

func NewFollower(stateStore common.StateStore, listener common.EventListener, timeout common.LeaderTimeout) *Follower {
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
	return nil
}

func (f *Follower) startBackGroundTimer() {
	timer := time.NewTimer(f.leaderTimeout())

	for {
		select {
		case <-f.StopCh:
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
				f.listener.HandleEvent(
					common.Event{
						Term:      f.stateStore.CurrentTerm(),
						EventType: common.LeaderKeepAliveTimeout,
					})
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
	return nil
}

func (f *Follower) resetTimer() {
	log.Debug("Reseting leader timer")
	f.resetTimerCh <- struct{}{}
}

func (f *Follower) KeepAliveRequest(req *rpc.KeepAliveRequest) error {
	f.resetTimer()
	return nil
}

func (f *Follower) RequestVote(req *rpc.VoteRequest) (bool, error) {
	f.resetTimer()

	voteGranted := false

	f.WithMutex(func() {
		votedFor := f.stateStore.VotedFor()
		reqId := req.Id()
		reqTerm := req.Term()

		if votedFor == nil || votedFor.Term < reqTerm || (votedFor.Term == reqTerm && votedFor.NodeId == reqId) {
			vote := common.Vote{reqTerm, reqId}
			f.stateStore.SaveVote(&vote)
			log.Debugf("Granted vote for id:%v term: %v", reqId, reqTerm)
			voteGranted = true
		}
	})

	return voteGranted, nil
}
