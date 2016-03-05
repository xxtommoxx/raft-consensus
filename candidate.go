package raft

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
)

type Candidate struct {
	*common.SyncService

	quorumOp QuorumStrategyOp
	listener CandidateListener
	client   rpc.Client

	stateStore StateStore
}

type CandidateListener interface {
	QuorumObtained(term uint32)
	QuorumUnobtained(term uint32)
}

type noopCandidateListener struct{}

func (n *noopCandidateListener) QuorumObtained(term uint32)   {}
func (n *noopCandidateListener) QuorumUnobtained(term uint32) {}

func NewCandidate(stateStore StateStore, client rpc.Client, quorumOp QuorumStrategyOp) *Candidate {
	c := &Candidate{
		stateStore: stateStore,
		listener:   &noopCandidateListener{},
		client:     client,
		quorumOp:   quorumOp,
	}

	c.SyncService = common.NewSyncService(c.syncStart, c.startVote, c.syncStop)

	return c
}

func (h *Candidate) SetListener(listener CandidateListener) {
	h.listener = listener
}

func (h *Candidate) startVote() {
	currentTerm := h.stateStore.CurrentTerm()

	log.Println("Starting vote process for term:", currentTerm)

	qOp := h.quorumOp.Accepted(currentTerm) // vote for itself

	if qOp.IsObtained() {
		h.listener.QuorumObtained(currentTerm)
		return
	} else {
		for res := range h.client.SendRequestVote(currentTerm) {
			if res.VoteGranted {
				if qOp = h.quorumOp.Accepted(currentTerm); qOp.IsObtained() {
					h.listener.QuorumObtained(currentTerm)
					return
				}
			}
		}

		h.listener.QuorumUnobtained(currentTerm)
	}

}

func (h *Candidate) syncStart() error {
	return nil
}

func (h *Candidate) syncStop() error {
	return nil
}
