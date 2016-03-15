package rpc

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"reflect"
	"sync"
)

type ClientSession interface {
	SendRequestVote(term uint32, cancelCh chan struct{}) <-chan *VoteResponse
	SendKeepAlive(term uint32, cancelCh chan struct{}) <-chan *KeepAliveResponse
	Terminate()
}

type Client struct {
	*common.SyncService
	config common.NodeConfig

	listener common.EventListener

	peers       []common.NodeConfig
	grpcClients map[string]*grpcClient

	sessions []ClientSession
}

func NewClient(listener common.EventListener, config common.NodeConfig,
	peers ...common.NodeConfig) *Client {

	client := &Client{
		peers:    peers,
		listener: listener,
		config:   config,
	}

	client.SyncService = common.NewSyncService(client.syncStart, nil, client.syncStop)
	return client
}

func (c *Client) syncStop() error {
	return c.stopGrpcClients()
}

func (c *Client) syncStart() error {
	log.Info("Start client")

	grpcClients := make(map[string]*grpcClient)

	for _, peer := range c.peers {
		grpcClient := newGrpcClient(c.config.Id, peer)

		if err := grpcClient.Start(); err != nil {
			log.Errorf("Failed to start for client: %v - error: %v", peer, err)
			return c.stopGrpcClients()
		} else {
			grpcClients[peer.Id] = grpcClient
		}
	}

	c.grpcClients = grpcClients

	log.Info("Start client complete")

	return nil
}

func (c *Client) stopGrpcClients() error {
	someFailed := false

	for _, g := range c.grpcClients {
		if err := g.Stop(); err != nil {
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

func (c *Client) NewSession() ClientSession {
	return newClientSession(c.grpcClients, c.listener)
}

type peerClient struct {
	*grpcClient
	requestCh chan func()
}

type clientSession struct {
	terminateCh chan struct{}
	peerClients map[string]*peerClient
	listener    common.EventListener
	requestCh   chan func()
}

func newClientSession(grpcClients map[string]*grpcClient, listener common.EventListener) *clientSession {
	peerClients := make(map[string]*peerClient)
	requestCh := make(chan func())

	for id, gClient := range grpcClients {
		peerClients[id] = newPeerClient(gClient, requestCh)
	}

	c := &clientSession{
		terminateCh: make(chan struct{}),
		listener:    listener,
		peerClients: peerClients,
		requestCh:   requestCh,
	}

	go c.processPeerRequests()
	go c.watchForTerminate()

	return c
}

func newPeerClient(gClient *grpcClient, requestCh chan func()) *peerClient {
	return &peerClient{
		grpcClient: gClient,
		requestCh:  requestCh,
	}
}

func (c *clientSession) watchForTerminate() {
	<-c.terminateCh

	log.Debug("Terminating session")

	for _, r := range c.peerClients {
		close(r.requestCh)
	}
}

func (c *clientSession) processPeerRequests() {
	for reqFn := range c.requestCh {
		reqFn()
	}
	for _, r := range c.peerClients {
		go func(p *peerClient) {
			for reqFn := range c.requestCh {
				reqFn()
			}
		}(r)
	}

}

func (c *clientSession) Terminate() {
	close(c.terminateCh)
}

func (c *clientSession) SendRequestVote(term uint32, cancelCh chan struct{}) <-chan *VoteResponse {
	fanOutFn := func(cap int) *fanout {
		reqFn := func(r *peerClient) (Response, error) {
			return r.requestVote(term)
		}

		return newFanout(reqFn, make(chan *VoteResponse, cap))
	}

	return c.fanoutRequest(fanOutFn, cancelCh).(chan *VoteResponse)
}

func (c *clientSession) SendKeepAlive(term uint32, cancelCh chan struct{}) <-chan *KeepAliveResponse {
	fanOutFn := func(cap int) *fanout {
		reqFn := func(r *peerClient) (Response, error) {
			return r.keepAlive(term)
		}

		return newFanout(reqFn, make(chan *KeepAliveResponse, cap))
	}

	return c.fanoutRequest(fanOutFn, cancelCh).(chan *KeepAliveResponse)
}

// Generic code that fans out requests to the peers.
type fanout struct {
	reqFn  func(*peerClient) (Response, error)
	respCh reflect.Value
}

func newFanout(reqFn func(*peerClient) (Response, error), respCh interface{}) *fanout {
	return &fanout{
		reqFn:  reqFn,
		respCh: reflect.ValueOf(respCh),
	}
}

func (c *clientSession) fanoutRequest(fanoutFn func(int) *fanout, cancelCh chan struct{}) interface{} {
	rpcClients := c.peerClients
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
		go func(p *peerClient) {
			defer common.NoopRecoverLog() // requestCh might be closed because of session being terminated

			p.requestCh <- func() {
				defer wg.Done()
				defer common.NoopRecoverLog() // responseCh might be closed because of cancelled
				resp, err := f.reqFn(p)
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

	return f.respCh.Interface()
}
