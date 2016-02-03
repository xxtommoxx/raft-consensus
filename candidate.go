package raft

type VoteResponse struct{}

const (
	stopVote = iota
)

type Candidate struct {
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
				}
			}
		}
	}()

	return nil
}

func (h *Candidate) Stop() error {
	h.voteCh <- stopVote
	return nil
}
