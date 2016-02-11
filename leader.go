package raft

import (
	"fmt"
	"time"
)

type Leader struct {
	*SyncService

	keepAliveMs time.Duration
	stopCh      chan int

	client Client
}

const (
	stopLeaderTimer = iota
)

func NewLeader(keepAliveMs uint32, client Client) *Leader {
	l := &Leader{
		keepAliveMs: time.Duration(keepAliveMs) * time.Millisecond,
		client:      client,
		stopCh:      make(chan int),
	}

	syncService := NewSyncService(l.syncStart, l.startKeepAliveTimer, l.syncStop)
	l.SyncService = syncService

	return l
}

func (l *Leader) startKeepAliveTimer() {
	fmt.Println("Starting leader keep alive timer")

	timer := time.NewTimer(l.keepAliveMs)

	keepAliveCancelChan := make(chan struct{})

	for {
		select {
		case <-l.stopCh: // currently there is only a stop timer event
			timer.Stop()
			if keepAliveCancelChan != nil {
				close(keepAliveCancelChan)
			}
		case <-timer.C:
			l.withMutex(func() {
				if l.serviceState != Stopped {
					l.client.sendKeepAlive(keepAliveCancelChan)
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
