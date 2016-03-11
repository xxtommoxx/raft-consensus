package raft

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
)

type Candidate struct {
	*common.SyncService

	quorumStrategy QuorumStrategy
	listener       common.EventListener
	client         rpc.Client

	stateStore common.StateStore
}

func NewCandidate(stateStore common.StateStore, client rpc.Client, listener common.EventListener, quorumStrategy QuorumStrategy) *Candidate {
	c := &Candidate{
		stateStore:     stateStore,
		listener:       listener,
		client:         client,
		quorumStrategy: quorumStrategy,
	}

	c.SyncService = common.NewSyncService(c.syncStart, c.startVote, c.syncStop)

	return c
}

func (h *Candidate) startVote() {
	currentTerm := h.stateStore.CurrentTerm()

	log.Println("Starting vote process for term:", currentTerm)

	qOp := h.quorumStrategy.NewOp(currentTerm)

	if qOp.IsObtained() {
		h.listener.HandleEvent(common.Event{Term: currentTerm, EventType: common.QuorumObtained})
	} else {
		for res := range h.client.SendRequestVote(currentTerm) {
			if res.VoteGranted {
				qOp.VoteReceived(res.Term)

				if qOp.IsObtained() {
					h.listener.HandleEvent(common.Event{Term: currentTerm, EventType: common.QuorumObtained})
					return
				}
			}
		}

		h.listener.HandleEvent(common.Event{Term: currentTerm, EventType: common.QuorumUnobtained})
	}
}

func (h *Candidate) syncStart() error {
	return nil
}

func (h *Candidate) syncStop() error {
	return nil
}

// a candidate never grants votes for anyone
func (c *Candidate) RequestVote(req *rpc.VoteRequest) (*rpc.VoteResponse, error) {
	resp := &rpc.VoteResponse{
		Term:        c.stateStore.CurrentTerm(),
		VoteGranted: false,
	}
	return resp, nil
}
