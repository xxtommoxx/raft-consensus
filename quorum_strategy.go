package raft

type QuorumStrategy interface {
	obtained(currentCount uint32) bool
}
