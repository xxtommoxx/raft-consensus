package raft

import (
	"sync"
)

// Mainly for testing
type InMemoryStateStore struct {
	currentTerm uint32
	vote        *Vote
	mutex       *sync.Mutex
}

func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		mutex: &sync.Mutex{},
	}
}

func (s *InMemoryStateStore) CurrentTerm() uint32 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.currentTerm
}

func (s *InMemoryStateStore) SaveCurrentTerm(term uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.currentTerm = term
}

func (s *InMemoryStateStore) VotedFor() *Vote {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.vote
}

func (s *InMemoryStateStore) SaveVotedFor(vote *Vote) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.vote = vote
}
