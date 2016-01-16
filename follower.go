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
	Listener     FollowerListener
	timeoutRange int64
	keepAlive    chan int
	random       *rand.Rand
}

func NewInstance() *Follower {
	return &Follower{}
}

func (h *Follower) Start(term uint32) error {
	fmt.Println("Starting follower")

	go func() {
		h.keepAlive = make(chan int)
		defer close(h.keepAlive)

		timer := time.NewTimer(h.leaderTimeout())

		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				if h.Listener != nil {
					h.Listener.KeepAliveTimeout(term)
				}
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

	return nil
}

func (h *Follower) Stop() error {
	if h.keepAlive != nil {
		h.keepAlive <- Close
	}

	return nil
}

func (h *Follower) leaderTimeout() time.Duration {
	return time.Duration(h.random.Int63n(h.timeoutRange)+h.timeout.MinMillis) * time.Millisecond
}
