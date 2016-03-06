package rpc

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
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
	common.Service
	SendRequestVote(term uint32) <-chan *VoteResponse
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

	listener common.EventListener

	peers      []common.NodeConfig
	rpcClients map[string]*gRpcClient // use SyncService withMutex to read / write
}

func NewClient(listener common.EventListener, config common.NodeConfig, peers ...common.NodeConfig) Client {
	client := &client{peers: peers, listener: listener, config: config}
	client.SyncService = common.NewSyncService(client.syncStart, client.asyncStart, client.syncStop)
	return client
}

func (c *client) asyncStart() {
	numClient := len(c.rpcClients)

	var wg sync.WaitGroup
	wg.Add(numClient)

	for _, r := range c.rpcClients {
		go func(rpcClient *gRpcClient) {
			for reqFn := range rpcClient.requestCh {
				reqFn()
			}

			wg.Done()
		}(r)
	}

	wg.Wait()
}

func (c *client) syncStop() error {
	return c.closeConnections()
}

func (c *client) syncStart() error {
	log.Info("Start rpc client")
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
		close(gRpcClient.requestCh)
		if err := gRpcClient.conn.Close(); err != nil {
			log.Error("Error closing client connection: ", err)
			someFailed = true
		}
	}

	if someFailed {
		return errors.New("Failed to close connection")
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

func (c *client) safeGetRpcClients() map[string]*gRpcClient {
	return c.WithMutexReturning(func() interface{} {
		return c.rpcClients
	}).(map[string]*gRpcClient)
}

func (c *client) SendRequestVote(term uint32) <-chan *VoteResponse {
	return c.fanoutRequest(func(r *gRpcClient) response {
		res, err := r.ElectLeader(context.Background(), &VoteRequest{Term: term, Id: c.config.Id})
		return response{res.Term, res, err}
	}).andFoward(func(respCap int) interface{} {
		return make(chan *VoteResponse, respCap)
	}).(chan *VoteResponse)
}

func (c *client) SendKeepAlive(term uint32) <-chan *KeepAliveResponse {
	return c.fanoutRequest(func(r *gRpcClient) response {
		res, err := r.KeepAlive(context.Background(), &KeepAliveRequest{LeaderInfo: c.leaderInfo(term)})
		return response{res.Term, res, err}
	}).andFoward(func(respCap int) interface{} {
		return make(chan *KeepAliveResponse, respCap)
	}).(chan *KeepAliveResponse)
}

// Generic code that fans out requests to the peers.
type requestFunc func(*gRpcClient) response

type response struct {
	term   uint32
	result interface{}
	err    error
}

type fanoutCh <-chan interface{}

func (f fanoutCh) andFoward(chFn func(int) interface{}) interface{} {
	respCap := cap(f)
	forwardCh := chFn(respCap)
	common.FowardChan(f, forwardCh)
	return forwardCh
}

func (c *client) fanoutRequest(handle requestFunc) fanoutCh {
	rpcClients := c.safeGetRpcClients()
	numClients := len(rpcClients)

	var wg sync.WaitGroup
	wg.Add(numClients)
	respCh := make(chan interface{}, numClients)

	go func() {
		defer close(respCh)
		wg.Wait()
	}()

	for _, r := range rpcClients {
		go func(rpcClient *gRpcClient) {
			rpcClient.requestCh <- func() {
				resp := handle(rpcClient)

				if resp.err != nil {
					log.Error(resp.err)
				} else {
					c.listener.HandleEvent(common.Event{Term: resp.term, EventType: common.ResponseReceived})
					respCh <- resp.result
				}

				wg.Done()
			}
		}(r)
	}

	return respCh
}
