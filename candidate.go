package raft

import (
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"log"
)

type VoteResponse struct{}

type Candidate struct {
	*common.SyncService

	quorum   QuorumStrategy
	listener CandidateListener
	client   rpc.Client

	stateStore StateStore
}

type CandidateListener interface {
	QuorumObtained(term uint32)
}

type noopCandidateListener struct{}

func (n *noopCandidateListener) QuorumObtained(term uint32) {}

func NewCandidate(stateStore StateStore, client rpc.Client, quorum QuorumStrategy) *Candidate {
	c := &Candidate{
		stateStore: stateStore,
		listener:   &noopCandidateListener{},
		client:     client,
		quorum:     quorum,
	}

	c.SyncService = common.NewSyncService(c.syncStart, c.startVote, c.syncStop)

	return c
}

func (h *Candidate) SetListener(listener CandidateListener) {
	h.listener = listener
}

func (h *Candidate) startVote() {
	currentTerm := h.stateStore.CurrentTerm()

	log.Printf("Starting candidate vote process for term: %v", currentTerm)

	responseChan := h.client.SendRequestVote(currentTerm)

	for {
		select {
		case <-responseChan:
			if h.quorum.VoteObtained(currentTerm) {
				h.listener.QuorumObtained(currentTerm)
				return
			}
		}
	}
}

func (h *Candidate) syncStart() error {
	return nil
}

func (h *Candidate) syncStop() error {
	return nil
}
