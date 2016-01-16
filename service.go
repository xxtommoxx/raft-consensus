package raft

type Service interface {
	Stop() error
	Start(term uint32) error
}
