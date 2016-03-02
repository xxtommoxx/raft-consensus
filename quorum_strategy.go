package raft

import (
	"math"
	"sync"
)

type QuorumStrategy interface {
	VoteObtained(term uint32) bool
}

type PercentBasedStrategy struct {
	numPeers uint32

	percentOfVotesNeeded float64
	votesObtained        uint32
	term                 uint32
	mutex                sync.Mutex
}

func (c *PercentBasedStrategy) NewPercentBasedStrategy(perecentOfVotesNeeded float64, numPeers uint32) *PercentBasedStrategy {
	return &PercentBasedStrategy{
		percentOfVotesNeeded: perecentOfVotesNeeded,
		numPeers:             numPeers,
	}
}

func (c *PercentBasedStrategy) VoteObtained(term uint32) bool {
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

		return c.votesObtained == uint32(math.Ceil(c.percentOfVotesNeeded*float64(c.numPeers)))
	}
}
