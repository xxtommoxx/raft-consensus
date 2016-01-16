package raft

type LeaderTimeout struct {
	MaxMillis int64
	MinMillis int64
}

// type Location struct {
// 	host string
// 	port uint32
// }
//
// type Config struct {
// 	MinElectionTimeoutMillis   int
// 	MaxElectionTimeoutMaxMllis int
// 	peers                      []Location
// }

// type Node struct {
// 	config Config
// }
//
// func NewNode(config Config, stateStore *StateStore) *Node {
// 	return &Node{config: config}
// }

/*********** NODE FSM **************/
