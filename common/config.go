package common

type NodeConfig struct {
	Id   string
	Host string
}

type LeaderTimeout struct {
	MaxMillis int64
	MinMillis int64
}
