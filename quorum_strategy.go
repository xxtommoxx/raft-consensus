package raft

type QuorumStrategy interface {
	NewOp(term uint32) QuorumOp
}

type QuorumOp interface {
	IsObtained() bool
	VoteReceived(term uint32)
}

type MajorityStrategy struct {
	numPeers int
}

func NewMajorityStrategy(numPeers int) QuorumStrategy {
	return &MajorityStrategy{
		numPeers: numPeers,
	}
}

func (c *MajorityStrategy) votesNeeded() int {
	if c.numPeers == 1 {
		return 1
	} else {
		return (c.numPeers / 2) + 1
	}
}

func (c *MajorityStrategy) NewOp(term uint32) QuorumOp {
	votes := 1
	greaterTerm := false

	op := struct{ OpHelper }{}
	op.IsObtainedFn = func() bool {
		return !greaterTerm && (c.votesNeeded() == 1 || c.votesNeeded() <= votes)
	}
	op.VoteReceivedFn = func(vTerm uint32) {
		if vTerm > term {
			greaterTerm = true
		} else {
			votes++
		}
	}

	return op
}

// allows implementing QuorumOp 'anonymously'
type OpHelper struct {
	IsObtainedFn   func() bool
	VoteReceivedFn func(uint32)
}

func (o OpHelper) IsObtained() bool {
	return o.IsObtainedFn()
}

func (o OpHelper) VoteReceived(term uint32) {
	o.VoteReceivedFn(term)
}
