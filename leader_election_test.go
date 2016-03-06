package raft

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"reflect"
	"testing"
	"time"
)

// todo: replace with raft bootstrap class
type fixture struct {
	client rpc.Client
	server *rpc.Server
	fsm    *NodeFSM
}

func (f fixture) start() {
	f.client.Start()
	f.server.Start()
	f.fsm.Start()
}

func (f fixture) stop() {
	f.client.Stop()
	f.server.Stop()
	f.fsm.Stop()
}

func removeAt(index int, slice interface{}) interface{} {
	s := reflect.ValueOf(slice)

	if index == 0 {
		return s.Slice(1, s.Len()).Interface()
	} else {
		return reflect.AppendSlice(s.Slice(0, index), s.Slice(index+1, s.Len())).Interface()
	}
}

func makeNodes(numNodes int) []fixture {

	fixtures := make([]fixture, numNodes)

	nodeConfigs := makeNodeConfigs(numNodes)

	for i, cfg := range makeConfigs(nodeConfigs) {
		peerConfigs := removeAt(i, nodeConfigs).([]common.NodeConfig)

		eventDispatcher := common.NewEventListenerDispatcher()
		client := rpc.NewClient(eventDispatcher, cfg.Self, peerConfigs...)

		stateStore := NewInMemoryStateStore()

		follower := NewFollower(stateStore, eventDispatcher, cfg.Leader.Timeout)
		leader := NewLeader(cfg.Leader.KeepAliveMs, client, stateStore)
		candidate := NewCandidate(stateStore, client, eventDispatcher, NewMajorityStrategyOp(numNodes))

		fsm := NewNodeFSM(stateStore, eventDispatcher, follower, candidate, leader)

		server := rpc.NewServer(cfg.Self.Host, fsm)

		fixtures[i] = fixture{
			client: client,
			server: server,
			fsm:    fsm,
		}
	}

	return fixtures
}

func makeNodeConfigs(numNodes int) []common.NodeConfig {
	idPrefix := "node"
	startPort := 8080
	nodeConfigs := make([]common.NodeConfig, numNodes)

	for x := 0; x < numNodes; x++ {
		nodeConfigs[x] = common.NodeConfig{
			Id:   fmt.Sprintf("%v-%v", idPrefix, x),
			Host: fmt.Sprintf("localhost:%v", x+startPort),
		}
	}

	return nodeConfigs
}

func makeConfigs(nodeConfigs []common.NodeConfig) []common.Config {
	configs := make([]common.Config, len(nodeConfigs))

	for i, nodeCfg := range nodeConfigs {
		configs[i] = common.Config{
			Self: nodeCfg,
			Leader: common.LeaderConfig{
				KeepAliveMs: 10,
				Timeout: common.LeaderTimeout{
					MaxMillis: 30,
					MinMillis: 20,
				},
			},
			Peers: removeAt(i, nodeConfigs).([]common.NodeConfig),
		}
	}

	return configs
}

func TestOneLeaderActive(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	n := makeNodes(1)
	n[0].start()

	time.Sleep(10 * time.Second)

	n[0].stop()
}
