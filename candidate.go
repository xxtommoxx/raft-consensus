package raft

type Client interface {
	sendRequestVote() <-chan VoteResponse
}

type VoteResponse struct{}

type BaseEvent struct {
	term uint32
}

type Candidate struct {
	quorum   QuorumStrategy
	listener CandidateListener
	client   Client

	stateStore StateStore
}

type CandidateListener interface {
	QuorumObtained(term uint32)
}

type noopCandidateListener struct{}

func (n *noopCandidateListener) QuorumObtained(term uint32) {}

func NewCandidate(stateStore StateStore, client Client, quorum QuorumStrategy) *Candidate {
	return &Candidate{
		stateStore: stateStore,
		listener:   &noopCandidateListener{},
		client:     client,
		quorum:     quorum,
	}
}

func (h *Candidate) SetListener(listener CandidateListener) {
	h.listener = listener
}

func (h *Candidate) Start() error {
	var currentVoteCount uint32 = 0
	currentTerm := h.stateStore.CurrentTerm()

	go func() {
		responseChan := h.client.sendRequestVote()
		for {
			select {
			case <-responseChan:
				currentVoteCount++
				if h.quorum.obtained(currentVoteCount) {
					// todo close responseChan
					h.listener.QuorumObtained(currentTerm)
				}
			}
		}
	}()

	return nil
}

func (h *Candidate) Stop() error {
	return nil
}
