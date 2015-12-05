package raft

type Location struct {
	id   Id
	host string
	port uint32
}

type Config struct {
	minElectionTimeout    uint32
	maxElectionTimeoutMax uint32
	peers                 []Location
}

type node struct {
	config Config
}

func NewNode(config Config, stateStore *StateStore) (node *node) {
	return &node{config: config}
}

func (n *node) bootstrap() {

}
