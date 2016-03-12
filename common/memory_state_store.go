package common

import (
	"sync/atomic"
)

// Mainly for testing -- thread safe in terms of memory visibility
type InMemoryStateStore struct {
	currentTerm uint32
	vote        atomic.Value
}

func NewInMemoryStateStore() *InMemoryStateStore {
	store := &InMemoryStateStore{}
	store.vote.Store(&Vote{})
	return store
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

func (s *InMemoryStateStore) SaveVote(vote *Vote) {
	s.vote.Store(vote)
}
