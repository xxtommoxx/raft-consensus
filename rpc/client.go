package rpc

import (
	"github.com/xxtommoxx/raft-consensus/common"
	"google.golang.org/grpc"
)

type Client interface {
	SendRequestVote(cancel <-chan struct{}) <-chan VoteResponse
	SendKeepAlive(cancel <-chan struct{}) <-chan KeepAliveResponse
}

type gRpcClient struct {
	conn      *grpc.ClientConn
	rpcClient RpcServiceClient
}

type client struct {
	*common.SyncService
	rpcClients map[string]*gRpcClient // use SyncService withMutex to read / write
}

func NewClient(hosts ...string) *client {
	rpcClients := make(map[string]*gRpcClient)

	for _, h := range hosts {
		rpcClients[h] = nil

	}
	client := &client{rpcClients: rpcClients}

	client.SyncService = common.NewSyncService(client.syncStart, nil, client.syncStop)

	return client
}
func (c *client) syncStop() error {
	return c.closeConnections()
}

func (c *client) syncStart() error {
	for h, _ := range c.rpcClients {
		newGRpcClient, err := newGRpcClient(h)

		if err != nil {
			c.closeConnections()
			return err
		} else {
			c.rpcClients[h] = newGRpcClient
		}
	}

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
		return nil // TODO return useful error mes
	} else {
		return nil
	}

}

func newGRpcClient(host string) (*gRpcClient, error) {
	opts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithBlock()}

	conn, err := grpc.Dial(host, opts...)

	if err != nil {
		return nil, err
	} else {
		return &gRpcClient{conn: conn, rpcClient: NewRpcServiceClient(conn)}, nil
	}
}
