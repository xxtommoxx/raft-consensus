package raft

type Vote struct {
	Term   uint32
	NodeId string
}

// These functions should panic if there is a failure
// since raft needs this to function correctly.
type StateStore interface {
	CurrentTerm() uint32
	SaveCurrentTerm(term uint32)

	VotedFor() *Vote
	SaveVote(vote *Vote)
}
