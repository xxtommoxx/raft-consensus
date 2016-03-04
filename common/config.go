package common

type Config struct {
	Self   NodeConfig
	Leader LeaderConfig
	Peers  []NodeConfig
}

type NodeConfig struct {
	Id   string
	Host string
}
type LeaderConfig struct {
	KeepAliveMs uint32
	Timeout     LeaderTimeout
}

type LeaderTimeout struct {
	MaxMillis int64
	MinMillis int64
}
