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
	Listener CandidateListener
	client   Client

	stateStore StateStore
}

type CandidateListener interface {
	QuorumObtained(term uint32)
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
					h.Listener.QuorumObtained(currentTerm)
				}
			}
		}
	}()

	return nil
}

func (h *Candidate) Stop() error {
	return nil
}
