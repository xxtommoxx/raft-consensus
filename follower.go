package raft

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	Close = iota
	Reset
)

type FollowerListener interface {
	KeepAliveTimeout(term uint32)
}

type Follower struct {
	timeout      LeaderTimeout
	timeoutRange int64
	keepAlive    chan int
	random       *rand.Rand
}

func NewInstance() *Follower {
	return &Follower{}
}

func (h *Follower) Start() {
	fmt.Println("Starting follower")

	go func() {
		h.keepAlive = make(chan int)
		defer close(h.keepAlive)

		timer := time.NewTimer(h.leaderTimeout())

		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				// h.listener.KeepAliveTimeout(h.currentTerm) TODO think about whether we need to pass currentTerm
			case timerEvent := <-h.keepAlive:
				if timerEvent == Close {
					fmt.Println("Terminating leader election timer")
					timer.Stop()
					return
				}

				if timerEvent == Reset {
					duration := h.leaderTimeout()
					if !timer.Reset(duration) {
						timer = time.NewTimer(duration)
					}
				}
			}
		}
	}()
}

func (h *Follower) Stop() {
	if h.keepAlive != nil {
		h.keepAlive <- Close
	}
}

func (h *Follower) leaderTimeout() time.Duration {
	return time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
}
