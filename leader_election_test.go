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
	client *rpc.Client
	server common.Service
	fsm    *NodeFSM
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}

}

func (f fixture) start() {
	panicIfError(f.server.Start())
	panicIfError(f.client.Start())
	panicIfError(f.fsm.Start())
}

func (f fixture) stop() {
	panicIfError(f.client.Stop())
	log.Info("Stopped client")
	panicIfError(f.server.Stop())
	log.Info("Stopped server")
	panicIfError(f.fsm.Stop())
	log.Info("Stopped fsm")
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

		stateStore := common.NewInMemoryStateStore()

		follower := NewFollower(stateStore, eventDispatcher, cfg.Leader.Timeout)
		leader := NewLeader(cfg.Leader.KeepAliveMs, client, stateStore)
		candidate := NewCandidate(stateStore, client, eventDispatcher, NewMajorityStrategy(numNodes))

		fsm := NewNodeFSM(stateStore, eventDispatcher, follower, candidate, leader, cfg.Self.Id)

		server := rpc.NewServer(cfg.Self, fsm, stateStore)

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
					MaxMillis: 1000,
					MinMillis: 500,
				},
			},
			Peers: removeAt(i, nodeConfigs).([]common.NodeConfig),
		}
	}

	return configs
}

func TestOneLeaderActive(t *testing.T) {
	log.SetLevel(log.InfoLevel)

	n := makeNodes(2)

	go func() {
		n[0].start()

	}()

	go func() {
		n[1].start()

	}()

	// go func() {
	// 	n[2].start()
	//
	// }()

	time.Sleep(100 * time.Second)
	// n[1].stop()
	//
	// for i := 0; i < 1000; i++ {
	// 	n[0].client.SendKeepAlive(120)
	// }
	//
	// n[0].stop()

	// time.Sleep(10 * time.Second)
	// log.Info("Starting....")
	// n[0].start()
	// time.Sleep(1000 * time.Second)

}
