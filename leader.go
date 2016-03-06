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
	stopCh      chan struct{}

	client     rpc.Client
	stateStore StateStore
}

func NewLeader(keepAliveMs uint32, client rpc.Client, stateStore StateStore) *Leader {
	l := &Leader{
		keepAliveMs: time.Duration(keepAliveMs) * time.Millisecond,
		client:      client,
		stateStore:  stateStore,
	}

	l.SyncService = common.NewSyncService(l.syncStart, l.startKeepAliveTimer, l.syncStop)

	return l
}

func (l *Leader) startKeepAliveTimer() {
	log.Debug("Sending keep alive to peers every ", l.keepAliveMs)
	timer := time.NewTimer(l.keepAliveMs)

	for {
		select {
		case <-l.stopCh: // currently there is only a stop timer event
			log.Debug("Stopping keep alive timer")
			timer.Stop()
			return
		case <-timer.C:
			if l.Status() == common.Started {
				l.client.SendKeepAlive(l.stateStore.CurrentTerm())
				timer = time.NewTimer(l.keepAliveMs)
			}
		}
	}

}

func (l *Leader) syncStart() error {
	l.stopCh = make(chan struct{})
	return nil
}

func (l *Leader) syncStop() error {
	close(l.stopCh)
	return nil
}
