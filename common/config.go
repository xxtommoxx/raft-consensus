package common

type Config struct {
	Self                  NodeConfig
	Leader                LeaderConfig
	Peers                 []NodeConfig
	PerecentOfVotesNeeded float64
}

type NodeConfig struct {
	Id   string
	Host string
}
type LeaderConfig struct {
	KeepAliveMs int64
	Timeout     LeaderTimeout
}

type LeaderTimeout struct {
	MaxMillis int64
	MinMillis int64
}
