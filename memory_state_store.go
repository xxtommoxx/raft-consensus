package raft

type InMemoryStateStore struct {
	currentTerm    uint32
	nodeIdVotedFor Id
}

func NewInMemoryStateStore() {
	&InMemoryStateStore{}
}

func (s *InMemoryStateStore) CurrentTerm() uint32 {
	return s.currentTerm
}

func (s *InMemoryStateStore) SaveCurrentTerm(term uint32) {
	s.currentTerm = term
}

func (s *InMemoryStateStore) VotedFor() Id {
	return s.nodeIdVotedFor
}

func (s *InMemoryStateStore) SaveVotedFor(nodeId Id) {
	s.nodeIdVotedFor = nodeId
}
