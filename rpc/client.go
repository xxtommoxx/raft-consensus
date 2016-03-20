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

	peers []common.NodeConfig

	grpcConnInfoCh       chan grpcConnInfo
	grpcClients          map[string]*grpcClient
	unhealthyGrpcClients map[string]*grpcClient

	sessions map[*clientSession]*struct{}
}

func NewClient(listener common.EventListener, config common.NodeConfig,
	peers ...common.NodeConfig) *Client {

	client := &Client{
		peers:                peers,
		listener:             listener,
		config:               config,
		grpcConnInfoCh:       make(chan grpcConnInfo),
		sessions:             make(map[*clientSession]*struct{}),
		unhealthyGrpcClients: make(map[string]*grpcClient),
	}

	client.SyncService = common.NewSyncService(client.syncStart, client.manageSessions, client.syncStop)
	return client
}

func (c *Client) syncStop() error {
	return c.stopGrpcClients()
}

func (c *Client) manageSessions() {
	for {
		select {
		case <-c.StopCh:
			return
		case connInfo := <-c.grpcConnInfoCh:
			c.WithMutex(func() {
				switch connInfo.state {
				case healthyConn:
					log.Infof("Adding back now healthy conn id: %v", connInfo.id)

					c.grpcClients[connInfo.id] = c.unhealthyGrpcClients[connInfo.id]
					delete(c.unhealthyGrpcClients, connInfo.id)

					for s, _ := range c.sessions {
						s.addGrpcClient(connInfo.id, c.grpcClients[connInfo.id])
					}

				case unhealthyConn:
					log.Infof("Removing unhealthy conn id: %v", connInfo.id)

					c.unhealthyGrpcClients[connInfo.id] = c.grpcClients[connInfo.id]
					delete(c.grpcClients, connInfo.id)

					for s, _ := range c.sessions {
						s.removeGrpcClient(connInfo.id)
					}
				}

			})
		}
	}
}

func (c *Client) syncStart() error {
	log.Info("Start client")

	grpcClients := make(map[string]*grpcClient)

	for _, peer := range c.peers {
		grpcClient := newGrpcClient(c.config.Id, peer, c.grpcConnInfoCh)

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
	var s *clientSession

	c.WithMutex(func() {
		s = newClientSession(c.grpcClients, c.listener, c.config.Id, c)
		c.sessions[s] = nil
	})

	return s
}

func (c *Client) removeSession(s *clientSession) {
	c.WithMutex(func() {
		delete(c.sessions, s)
	})
}

type peerClient struct {
	*grpcClient
	requestCh chan func()
}

type clientSession struct {
	client *Client // TODO client dependency should be removed; only used to remove session

	terminateCh chan struct{}
	peerClients map[string]*peerClient
	listener    common.EventListener

	mutex sync.RWMutex
}

func newClientSession(grpcClients map[string]*grpcClient,
	listener common.EventListener, id string, client *Client) *clientSession {
	peerClients := make(map[string]*peerClient)

	for id, gClient := range grpcClients {
		peerClients[id] = newPeerClient(gClient)
	}

	c := &clientSession{
		terminateCh: make(chan struct{}),
		listener:    listener,
		client:      client,
		peerClients: peerClients,
	}

	go c.watchForTerminate()

	return c
}

func (c *clientSession) addGrpcClient(id string, gClient *grpcClient) {
	defer c.mutex.Unlock()
	c.mutex.Lock()

	if peerClient := c.peerClients[id]; peerClient == nil {
		c.peerClients[id] = newPeerClient(gClient)
	}
}

func (c *clientSession) removeGrpcClient(id string) {
	defer c.mutex.Unlock()
	defer common.NoopRecoverLog() // session is terminating and already closed requestCh and this was invoked concurrently
	c.mutex.Lock()

	if peerClient := c.peerClients[id]; peerClient != nil {
		close(peerClient.requestCh)
		delete(c.peerClients, id)
	}
}

func newPeerClient(gClient *grpcClient) *peerClient {
	p := &peerClient{
		grpcClient: gClient,
		requestCh:  make(chan func()),
	}

	go processPeerRequests(p)
	return p
}

func (c *clientSession) watchForTerminate() {
	defer c.mutex.RUnlock()
	defer common.NoopRecoverLog() // requestCh might be closed by an unhealthy connection in removeGrpcClient

	<-c.terminateCh

	c.mutex.RLock()

	log.Debug("Terminating session")

	for _, r := range c.peerClients {
		close(r.requestCh)
	}

	c.client.removeSession(c)
}

func processPeerRequests(p *peerClient) {
	for reqFn := range p.requestCh {
		reqFn()
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
	defer c.mutex.RUnlock()
	c.mutex.RLock()

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

			// requestCh might be closed because of session being terminated
			// let go of the wait
			defer common.RecoverAndDo(func() { wg.Done() })

			p.requestCh <- func() {
				defer wg.Done()
				defer common.NoopRecoverLog() // respCh might be closed because of session being terminated

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
