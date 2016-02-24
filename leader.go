package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"time"
)

type Leader struct {
	*common.SyncService

	keepAliveMs time.Duration
	stopCh      chan int

	client     rpc.Client
	stateStore StateStore
}

const (
	stopLeaderTimer = iota
)

func NewLeader(keepAliveMs uint32, client rpc.Client, stateStore StateStore) *Leader {
	l := &Leader{
		keepAliveMs: time.Duration(keepAliveMs) * time.Millisecond,
		client:      client,
		stopCh:      make(chan int),
		stateStore:  stateStore,
	}

	l.SyncService = common.NewSyncService(l.syncStart, l.startKeepAliveTimer, l.syncStop)

	return l
}

func (l *Leader) startKeepAliveTimer() {
	fmt.Println("Starting leader keep alive timer")

	timer := time.NewTimer(l.keepAliveMs)

	for {
		select {
		case <-l.stopCh: // currently there is only a stop timer event
			timer.Stop()
		case <-timer.C:
			l.WithMutex(func() {
				if l.Status != common.Stopped {
					l.client.SendKeepAlive(l.stateStore.CurrentTerm())
					timer = time.NewTimer(l.keepAliveMs)
				}
			})
		}
	}

}

func (l *Leader) syncStart() error {
	fmt.Println("Starting leader")

	return nil
}

func (l *Leader) syncStop() error {
	fmt.Println("Stopping leader")

	l.stopCh <- stopLeaderTimer
	return nil
}
