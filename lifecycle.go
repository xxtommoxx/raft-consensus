package raft

type Lifecycle interface {
	Stop() error
	Start(term uint32) error
}
