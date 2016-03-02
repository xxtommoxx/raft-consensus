package raft

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/common"
	"testing"
)

func makeConfigs(numNodes int) []common.Config {
	idPrefix := "node"
	startPort := 8080
	nodeConfigs := make([]common.NodeConfig, numNodes)

	for x := 0; x < numNodes; x++ {
		nodeConfigs[x] = common.NodeConfig{
			Id:   fmt.Sprintf("%v-%v", idPrefix, x),
			Host: fmt.Sprintf("localhost:%v", x+startPort),
		}
	}

	configs := make([]common.Config, numNodes)

	for x := 0; x < numNodes; x++ {
		var peers []common.NodeConfig

		if x == 0 {
			peers = nodeConfigs[1:numNodes]
		} else {
			peers = append(nodeConfigs[0:x-1], nodeConfigs[x+1:numNodes]...)
		}

		configs[x] = common.Config{
			Self: common.NodeConfig{
				Id:   "n1",
				Host: "localhost:8081",
			},
			Leader: common.LeaderConfig{
				KeepAliveMs: 100,
				Timeout: common.LeaderTimeout{
					MaxMillis: 400,
					MinMillis: 200,
				},
			},
			Peers: peers,
			PerecentOfVotesNeeded: 0.5,
		}
	}

	return nil

}

func TestOneLeaderActive(t *testing.T) {
}
