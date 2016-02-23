package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
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
	fmt.Println("Starting candidate vote process")

	currentTerm := h.stateStore.CurrentTerm()
	var currentVoteCount uint32 = 0

	cancelChan := make(chan struct{})
	responseChan := h.client.SendRequestVote(cancelChan)

	for {
		select {
		case <-responseChan:
			currentVoteCount++
			if h.quorum.obtained(currentVoteCount) {
				h.listener.QuorumObtained(currentTerm)
				return
			}
		}
	}
}

func (h *Candidate) syncStart() error {
	fmt.Println("Starting candidate")
	return nil
}

func (h *Candidate) syncStop() error {
	return nil
}
