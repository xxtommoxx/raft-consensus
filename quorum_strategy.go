package raft

import (
	"sync"
)

type QuorumStrategy interface {
	VoteObtained(term uint32) bool
}

type MajorityStrategy struct {
	numPeers uint32

	votesObtained uint32
	term          uint32
	mutex         sync.Mutex
}

func (c *MajorityStrategy) NewMajorityStrategy(numPeers uint32) *MajorityStrategy {
	return &MajorityStrategy{
		numPeers: numPeers,
	}
}

func (c *MajorityStrategy) VoteObtained(term uint32) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.term < term {
		return false
	} else {
		if c.term == term {
			c.votesObtained++
		} else {
			c.term = term
			c.votesObtained = 1
		}

		return c.votesObtained == (c.numPeers/2)+1
	}
}
