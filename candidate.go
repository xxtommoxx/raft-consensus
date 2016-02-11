package raft

import "fmt"

type VoteResponse struct{}

const (
	stopVote = iota
)

type Candidate struct {
	*SyncService

	quorum   QuorumStrategy
	listener CandidateListener
	client   Client

	stateStore StateStore
	voteCh     chan int
}

type CandidateListener interface {
	QuorumObtained(term uint32)
}

type noopCandidateListener struct{}

func (n *noopCandidateListener) QuorumObtained(term uint32) {}

func NewCandidate(stateStore StateStore, client Client, quorum QuorumStrategy) *Candidate {
	c := &Candidate{
		stateStore: stateStore,
		listener:   &noopCandidateListener{},
		client:     client,
		quorum:     quorum,
	}
	syncService := NewSyncService(c.syncStart, c.startVote, c.syncStop)
	c.SyncService = syncService

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
	responseChan := h.client.sendRequestVote(cancelChan)

	for {
		select {
		case <-h.voteCh:
			close(cancelChan)
		case <-responseChan:
			currentVoteCount++
			if h.quorum.obtained(currentVoteCount) {
				close(cancelChan)
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
	h.voteCh <- stopVote
	return nil
}
