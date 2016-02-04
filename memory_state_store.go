package raft

import (
	"sync/atomic"
)

// Mainly for testing
type InMemoryStateStore struct {
	currentTerm uint32
	vote        atomic.Value
}

func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{}
}

func (s *InMemoryStateStore) CurrentTerm() uint32 {
	return atomic.LoadUint32(&s.currentTerm)
}

func (s *InMemoryStateStore) SaveCurrentTerm(term uint32) {
	atomic.StoreUint32(&s.currentTerm, term)
}

func (s *InMemoryStateStore) VotedFor() *Vote {
	return s.vote.Load().(*Vote)
}

func (s *InMemoryStateStore) SaveVotedFor(vote *Vote) {
	s.vote.Store(vote)
}
