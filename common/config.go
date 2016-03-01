package common

import "time"

type Config struct {
	self   NodeConfig
	leader LeaderConfig
	peers  []NodeConfig
}

type NodeConfig struct {
	Id   string
	Host string
}
type LeaderConfig struct {
	keepAliveMs time.Duration
	timeout     LeaderTimeout
}

type LeaderTimeout struct {
	MaxMillis int64
	MinMillis int64
}
