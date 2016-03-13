package raft

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"time"
)

type Leader struct {
	*common.SyncService

	keepAliveMs time.Duration

	client     rpc.Client
	stateStore common.StateStore
}

func NewLeader(keepAliveMs uint32, client rpc.Client, stateStore common.StateStore) *Leader {
	l := &Leader{
		keepAliveMs: time.Duration(keepAliveMs) * time.Millisecond,
		client:      client,
		stateStore:  stateStore,
	}

	l.SyncService = common.NewSyncService(l.syncStart, l.startKeepAliveTimer, l.syncStop)

	return l
}

func (l *Leader) startKeepAliveTimer() {
	log.Info("Sending keep alive to peers every ", l.keepAliveMs)

	currentTerm := l.stateStore.CurrentTerm()
	timer := time.NewTimer(l.keepAliveMs)

	for {
		select {
		case <-l.StopCh: // currently there is only a stop timer event
			log.Debug("Stopping keep alive timer")
			timer.Stop()
			return
		case <-timer.C:
			if l.Status() == common.Started {
				l.client.SendKeepAlive(currentTerm, l.StopCh)
				timer = time.NewTimer(l.keepAliveMs)
			}
		}
	}
}

func (l *Leader) syncStart() error {
	return nil
}

func (l *Leader) syncStop() error {
	return nil
}
