package rpc

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"reflect"
	"sync"
)

type Client interface {
	common.Service
	SendRequestVote(term uint32, cancelCh chan struct{}) <-chan *VoteResponse
	SendKeepAlive(term uint32, cancelCh chan struct{}) <-chan *KeepAliveResponse
}

type grpcClientCh struct {
	gClient   *grpcClient
	requestCh chan func()
}

func (g *grpcClientCh) stop() error {
	close(g.requestCh)
	return g.gClient.Stop()
}

type client struct {
	*common.SyncService
	config common.NodeConfig

	listener common.EventListener

	peers    []common.NodeConfig
	gClients map[string]*grpcClientCh
}

func NewClient(listener common.EventListener, config common.NodeConfig,
	peers ...common.NodeConfig) Client {

	client := &client{
		peers:    peers,
		listener: listener,
		config:   config,
	}

	client.SyncService = common.NewSyncService(client.syncStart, client.asyncStart, client.syncStop)
	return client
}

func (c *client) asyncStart() {
	gClients := c.safeGetRpcClients()

	numClient := len(gClients)

	var wg sync.WaitGroup
	wg.Add(numClient)

	for _, r := range gClients {
		go func(gClientCh *grpcClientCh) {
			for reqFn := range gClientCh.requestCh {
				reqFn()
			}

			wg.Done()
		}(r)
	}

	wg.Wait()
}

func (c *client) syncStop() error {
	return c.stopGrpcClients()
}

func (c *client) syncStart() error {
	log.Info("Start client")

	gClients := make(map[string]*grpcClientCh)

	for _, peer := range c.peers {
		grpcClient := newGrpcClient(c.config.Id, peer)

		if err := grpcClient.Start(); err != nil {
			log.Errorf("Failed to start for client: %v - error: %v", peer, err)
			return c.stopGrpcClients()
		} else {
			gClients[peer.Id] = &grpcClientCh{
				gClient:   grpcClient,
				requestCh: make(chan func()),
			}
		}
	}

	c.gClients = gClients

	log.Info("Start client complete")

	return nil
}

func (c *client) stopGrpcClients() error {
	someFailed := false

	for _, gClient := range c.gClients {
		if err := gClient.stop(); err != nil {
			log.Error("Error stopping: %v", err)
			someFailed = true
		}
	}

	if someFailed {
		return errors.New("Failed to stop some of the gRpc clients -- see logs")
	} else {
		return nil
	}
}

func (c *client) safeGetRpcClients() map[string]*grpcClientCh {
	return c.WithMutexReturning(func() interface{} {
		return c.gClients
	}).(map[string]*grpcClientCh)
}

func (c *client) SendRequestVote(term uint32, cancelCh chan struct{}) <-chan *VoteResponse {
	fanOutFn := func(cap int) *fanout {
		reqFn := func(r *grpcClientCh) (Response, error) {
			return r.gClient.requestVote(term)
		}

		return newFanout(reqFn, make(chan *VoteResponse, cap))
	}

	return c.fanoutRequest(fanOutFn, cancelCh).(chan *VoteResponse)
}

func (c *client) SendKeepAlive(term uint32, cancelCh chan struct{}) <-chan *KeepAliveResponse {
	fanOutFn := func(cap int) *fanout {
		reqFn := func(r *grpcClientCh) (Response, error) {
			return r.gClient.keepAlive(term)
		}

		return newFanout(reqFn, make(chan *KeepAliveResponse, cap))
	}

	return c.fanoutRequest(fanOutFn, cancelCh).(chan *KeepAliveResponse)
}

// Generic code that fans out requests to the peers.
type fanout struct {
	reqFn  func(*grpcClientCh) (Response, error)
	respCh reflect.Value
}

func newFanout(reqFn func(*grpcClientCh) (Response, error), respCh interface{}) *fanout {
	return &fanout{
		reqFn:  reqFn,
		respCh: reflect.ValueOf(respCh),
	}
}

func (c *client) fanoutRequest(fanoutFn func(int) *fanout, cancelCh chan struct{}) interface{} {
	rpcClients := c.safeGetRpcClients()
	numClients := len(rpcClients)

	f := fanoutFn(numClients)

	var wg sync.WaitGroup
	wg.Add(numClients)

	doneCh := make(chan struct{})

	go func() {
		select {
		case <-cancelCh:
			f.respCh.Close()
		case <-doneCh:
			f.respCh.Close()
		}
	}()

	go func() {
		defer close(doneCh)
		wg.Wait()
	}()

	for _, r := range rpcClients {
		go func(gClientCh *grpcClientCh) {
			defer wg.Done()

			gClientCh.requestCh <- func() {
				resp, err := f.reqFn(gClientCh)

				if err != nil {
					log.Warn(err)
				} else {
					c.listener.HandleEvent(
						common.Event{
							Term:      resp.Term(),
							EventType: common.ResponseReceived,
						})
					f.respCh.Send(reflect.ValueOf(resp))
				}
			}
		}(r)
	}

	return f.respCh
}
