package raft

type Id string

type Config struct {
	minElectionTimeout    uint32
	maxElectionTimeoutMax uint32
	peers                 []Location
}

type Location struct {
	id   Id
	host string
	port uint32
}

type Node struct {
	config Config
}

func NewNode(config Config, stateStore *StateStore) (node *Node) {
	&Node{config: config}
}

func (n *Node) bootstrap() {

}
