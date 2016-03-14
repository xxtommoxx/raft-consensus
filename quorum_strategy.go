package raft

type QuorumStrategy interface {
	NewOp(term uint32) QuorumOp
}

type QuorumOp interface {
	IsObtained() bool
	VoteReceived(term uint32) QuorumOp
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
	type recFn func(int, bool, recFn) QuorumOp

	// todo make easier to understand
	qFn := func(votes int, greaterTerm bool, fn recFn) QuorumOp {
		op := struct{ OpHelper }{}
		op.IsObtainedFn = func() bool { return !greaterTerm && (c.votesNeeded() == 1 || c.votesNeeded() <= votes) }
		op.VoteReceivedFn = func(vTerm uint32) QuorumOp {
			if greaterTerm || op.IsObtainedFn() {
				return op
			} else {
				return fn(votes+1, vTerm > term, fn)
			}
		}
		return op
	}

	return qFn(0, false, qFn)
}

// allows implementing QuorumOp 'anonymously'
type OpHelper struct {
	IsObtainedFn   func() bool
	VoteReceivedFn func(uint32) QuorumOp
}

func (o OpHelper) IsObtained() bool {
	return o.IsObtainedFn()
}

func (o OpHelper) VoteReceived(term uint32) QuorumOp {
	return o.VoteReceivedFn(term)
}
