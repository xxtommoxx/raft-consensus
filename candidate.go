package raft

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
)

type Candidate struct {
	*common.SyncService

	quorumOp QuorumStrategyOp
	listener common.EventListener
	client   rpc.Client

	stateStore StateStore
}

func NewCandidate(stateStore StateStore, client rpc.Client, listener common.EventListener, quorumOp QuorumStrategyOp) *Candidate {
	c := &Candidate{
		stateStore: stateStore,
		listener:   listener,
		client:     client,
		quorumOp:   quorumOp,
	}

	c.SyncService = common.NewSyncService(c.syncStart, c.startVote, c.syncStop)

	return c
}

func (h *Candidate) startVote() {
	currentTerm := h.stateStore.CurrentTerm()

	log.Println("Starting vote process for term:", currentTerm)

	qOp := h.quorumOp.Accepted(currentTerm) // vote for itself

	if qOp.IsObtained() {
		h.listener.HandleEvent(common.Event{Term: currentTerm, EventType: common.QuorumObtained})
		return
	} else {
		for res := range h.client.SendRequestVote(currentTerm) {
			if res.VoteGranted {
				if qOp = h.quorumOp.Accepted(currentTerm); qOp.IsObtained() {
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
