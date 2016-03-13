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

	client        rpc.Client
	clientSession rpc.ClientSession

	stateStore common.StateStore
}

func NewCandidate(stateStore common.StateStore, client rpc.Client,
	listener common.EventListener, quorumStrategy QuorumStrategy) *Candidate {

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

	log.Printf("Starting vote process for term: %v", currentTerm)

	eventFn := common.NewEvent(currentTerm)

	qOp := h.quorumStrategy.NewOp(currentTerm)

	if qOp.IsObtained() {
		h.listener.HandleEvent(eventFn(common.QuorumObtained))
	} else {
		for res := range h.clientSession.SendRequestVote(currentTerm, h.StopCh) {
			if op := qOp.VoteReceived(res.Term()); op.IsObtained() {
				h.listener.HandleEvent(eventFn(common.QuorumObtained))
				return
			}
		}

		h.listener.HandleEvent(eventFn(common.QuorumUnobtained))
	}
}

func (h *Candidate) syncStart() error {
	h.clientSession = h.client.NewSession()
	return nil
}

func (h *Candidate) syncStop() error {
	h.clientSession.Terminate()
	h.clientSession = nil
	return nil
}
