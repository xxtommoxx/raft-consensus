package raft

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc"
	"reflect"
	"sync"
	"testing"
	"time"
)

const (
	UseLeaderKeepAliveMs  = 10
	UseMinLeaderTimeoutMs = 30
	UseMaxLeaderTimeoutMs = 40
)

func TestOneLeaderActive(t *testing.T) {
	fixture := makeNodes(5)

	defer fixture.stopAll()
	fixture.startAll()

	if leaderObtained, leaderIndex := fixture.waitForLeader(); leaderObtained {
		runForTimes(50, func() {
			for i, n := range fixture.nodes {
				if i == leaderIndex && n.fsm.currentState != leaderState {
					t.Fatal("Assumed leader is no longer leader")
				} else if i != leaderIndex && n.fsm.currentState == leaderState {
					t.Fatal("Multiple leaders")
				}
			}
		})
	} else {
		t.Fail()
	}
}

func runForTimes(times int, fn func()) {
	for i := 0; i < times; i++ {
		fn()
		time.Sleep(time.Millisecond * 10)
	}
}

// func assertStable(t *testing.T, failMsg string, delay time.Duration, tries int, doFn func() bool) {
// 	if !waitForStable(delay, tries, doFn) {
// 		t.Fatal(failMsg)
// 	}
// }

type fixture struct {
	nodes []node
}

func (f fixture) startAll() {
	var wg sync.WaitGroup
	wg.Add(len(f.nodes))

	for _, n := range f.nodes {
		go func(n_ node) {
			n_.start()
			wg.Done()
		}(n)
	}

	wg.Wait()
}

func (f fixture) stopAll() {
	var wg sync.WaitGroup

	wg.Add(len(f.nodes))

	for _, n := range f.nodes {
		go func(n_ node) {
			n_.stop()
			wg.Done()
		}(n)
	}

	wg.Wait()
}

func (f fixture) waitForLeader() (bool, int) {
	index := -1
	isStable := waitForStable(time.Millisecond*UseMinLeaderTimeoutMs, 10+(UseMaxLeaderTimeoutMs/UseMinLeaderTimeoutMs), func() bool {
		for i, n := range f.nodes {

			if n.fsm.currentState == leaderState {
				index = i
				return true
			}
		}
		return false
	})

	return isStable, index
}

func waitForStable(delay time.Duration, tries int, doFn func() bool) bool {
	for tryCount := 0; tryCount < tries; tryCount = tryCount + 1 {
		if doFn() == true {
			return true
		} else {
			time.Sleep(delay)
		}
	}

	log.Errorf("waitForStable not reached -- used delay: %v, tries: %v", delay, tries)

	return false
}

type node struct {
	client *rpc.Client
	server common.Service
	fsm    *NodeFSM
}

func (n node) start() {
	startService(n.server)
	startService(n.client)
	startService(n.fsm)
}

func (n node) stop() {
	stopService(n.client)
	stopService(n.server)
	stopService(n.fsm)
}

func startService(s common.Service) {
	if s.Status() == common.Unstarted {
		panicIfError(s.Start())
	}
}

func stopService(s common.Service) {
	if s.Status() == common.Started || s.Status() == common.Stopped {
		panicIfError(s.Stop())
	}
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func makeNodes(numNodes int) fixture {

	nodes := make([]node, numNodes)

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

		nodes[i] = node{
			client: client,
			server: server,
			fsm:    fsm,
		}
	}

	return fixture{
		nodes: nodes,
	}
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
				KeepAliveMs: UseLeaderKeepAliveMs,
				Timeout: common.LeaderTimeout{
					MaxMillis: UseMaxLeaderTimeoutMs,
					MinMillis: UseMinLeaderTimeoutMs,
				},
			},
			Peers: removeAt(i, nodeConfigs).([]common.NodeConfig),
		}
	}

	return configs
}

func removeAt(index int, s interface{}) interface{} {
	t, v := reflect.TypeOf(s), reflect.ValueOf(s)
	cpy := reflect.MakeSlice(t, v.Len(), v.Len())
	reflect.Copy(cpy, v)
	if index == 0 {
		return cpy.Slice(1, cpy.Len()).Interface()
	} else {
		return reflect.AppendSlice(cpy.Slice(0, index), cpy.Slice(index+1, cpy.Len())).Interface()
	}
}
