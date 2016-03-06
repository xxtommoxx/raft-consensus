package raft

// represents an immutable quorum operation on a request of some sort
type QuorumStrategyOp interface {
	Accepted(term uint32) QuorumStrategyOp
	IsObtained() bool
}

type MajorityStrategyOp struct {
	numPeers int

	votesObtained int
	term          uint32
}

func NewMajorityStrategyOp(numPeers int) QuorumStrategyOp {
	return &MajorityStrategyOp{
		numPeers: numPeers,
	}
}

func (c *MajorityStrategyOp) IsObtained() bool {
	return (c.numPeers == 1 && c.votesObtained == 1) || (c.votesObtained == (c.numPeers/2)+1)
}

func (c *MajorityStrategyOp) Accepted(term uint32) QuorumStrategyOp {
	if c.term == 0 || c.term >= term {
		return &MajorityStrategyOp{
			numPeers:      c.numPeers,
			term:          term,
			votesObtained: c.votesObtained + 1,
		}
	} else {
		return alwaysFalseStrategy{}
	}

}

type alwaysFalseStrategy struct{}

func (alwaysFalseStrategy) Accepted(term uint32) QuorumStrategyOp {
	return alwaysFalseStrategy{}
}

func (alwaysFalseStrategy) IsObtained() bool {
	return false
}
