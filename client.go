package raft

type Client interface {
	sendRequestVote(cancel <-chan struct{}) <-chan VoteResponse
	sendKeepAlive(cancel <-chan struct{})
}
