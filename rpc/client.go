package rpc

import (
	"errors"
	"fmt"
	"github.com/xxtommoxx/raft-consensus/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"sync"
	"time"
)

type StreamVoteResponse struct {
	response <-chan struct{}
}

type Client interface {
	SendRequestVote(term uint32) <-chan VoteResponse
	SendKeepAlive(term uint32) <-chan *KeepAliveResponse
}

type gRpcClient struct {
	conn *grpc.ClientConn
	RpcServiceClient
	requestCh chan func()
}

type client struct {
	*common.SyncService
	config common.NodeConfig

	peers      []common.NodeConfig
	rpcClients map[string]*gRpcClient // use SyncService withMutex to read / write
}

func NewClient(config common.NodeConfig, peers ...common.NodeConfig) *client {
	client := &client{peers: peers}
	client.SyncService = common.NewSyncService(client.syncStart, client.asyncStart, client.syncStop)
	return client
}

func (c *client) asyncStart() {
	numClient := len(c.rpcClients)

	var wg sync.WaitGroup
	wg.Add(numClient)

	for _, rpcClient := range c.rpcClients {
		go func() {
			for reqFn := range rpcClient.requestCh {
				reqFn()
			}
		}()
	}
}

func (c *client) syncStop() error {
	return c.closeConnections() // TODO let each individual client handle
}

func (c *client) syncStart() error {
	rpcClientMap := make(map[string]*gRpcClient)

	for _, peer := range c.peers {
		newGRpcClient, err := newGRpcClient(peer)

		if err != nil {
			c.closeConnections()
			return err
		} else {
			rpcClientMap[peer.Id] = newGRpcClient
		}
	}

	c.rpcClients = rpcClientMap

	return nil
}

func (c *client) closeConnections() error {
	someFailed := false

	for _, gRpcClient := range c.rpcClients {

		if gRpcClient != nil {
			if err := gRpcClient.conn.Close(); err != nil {
				someFailed = true
				// log
			}
		}
	}

	if someFailed {
		return nil // TODO return useful error msg
	} else {
		return nil
	}

}

func newGRpcClient(peer common.NodeConfig) (*gRpcClient, error) {
	conn, err := grpc.Dial(peer.Host, grpc.WithInsecure())

	if err != nil {
		return nil, err
	} else {
		for i := 1; i <= 20; i++ {
			s, sErr := conn.State()

			if sErr != nil {
				return nil, sErr
			} else {
				if s == grpc.TransientFailure {
					return nil, errors.New(fmt.Sprintf("Failed to establish initial connection for host %v", peer.Host))
				} else if s == grpc.Ready {
					return &gRpcClient{conn: conn, RpcServiceClient: NewRpcServiceClient(conn), requestCh: make(chan func())}, nil
				}

				time.Sleep(500 * time.Millisecond)
			}
		}
		return nil, errors.New("Initial connection failed to go to state READY")
	}
}

func (c *client) leaderInfo(term uint32) *LeaderInfo {
	return &LeaderInfo{
		Id:   c.config.Id,
		Term: term,
	}
}

func (c *client) SendRequestVote(term uint32) <-chan VoteResponse {

	return nil
}

func (c *client) safeGetRpcClients() map[string]*gRpcClient {
	return c.WithMutexReturning(func() interface{} {
		return c.rpcClients
	}).(map[string]*gRpcClient)
}

type RequestFunc func(*gRpcClient) (interface{}, error)

type FanoutCh <-chan interface{}

func (c FanoutCh) andFoward(chFn func(int) interface{}) interface{} {
	respCap := cap(c)
	forwardCh := chFn(respCap)
	common.FowardChan(c, forwardCh)
	return forwardCh
}

func (c *client) fanoutRequest(handle RequestFunc) FanoutCh {
	rpcClients := c.safeGetRpcClients()
	numClients := len(rpcClients)

	var wg sync.WaitGroup
	wg.Add(numClients)
	respCh := make(chan interface{}, numClients)

	for _, rpcClient := range rpcClients {
		go func() {
			defer close(respCh)

			rpcClient.requestCh <- func() {
				resp, err := handle(rpcClient)

				if err != nil {
					fmt.Println(err)
				}

				respCh <- resp
				wg.Done()
			}

			wg.Wait()
		}()
	}

	return respCh
}

func (c *client) SendKeepAlive(term uint32) <-chan *KeepAliveResponse {
	respCh := c.fanoutRequest(func(r *gRpcClient) (interface{}, error) {
		return r.KeepAlive(context.Background(), &KeepAliveRequest{LeaderInfo: c.leaderInfo(term)})
	}).andFoward(func(respCap int) interface{} {
		return make(chan *KeepAliveResponse, respCap)
	})

	return respCh.(chan *KeepAliveResponse)
}
