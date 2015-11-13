package raft

type StateStore interface {
	CurrentTerm() uint32
	SaveCurrentTerm(term uint32)

	VotedFor() Id
	SaveVotedFor(nodeId Id)
}
