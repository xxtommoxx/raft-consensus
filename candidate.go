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
}

type CandidateListener interface {
	quorumObtained(term uint32)
}

func (h *Candidate) start(term uint32) error {
	var currentVoteCount uint32 = 0

	go func() {
		responseChan := h.client.sendRequestVote()
		for {
			select {
			case <-responseChan:
				currentVoteCount++
				if h.quorum.obtained(currentVoteCount) {
					// todo close responseChan
					h.listener.quorumObtained(term)
				}
			}
		}
	}()

	return nil
}

func (h *Candidate) stop() error {
	return nil
}
