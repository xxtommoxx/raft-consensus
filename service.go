package raft

type Service interface {
	Stop() error
	Start() error
}
