package rpc

import (
	"errors"
	"fmt"
	"github.com/xxtommoxx/raft-consensus/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"time"
)

type StreamVoteResponse struct {
	response <-chan struct{}
}

type Client interface {
	SendRequestVote() <-chan VoteResponse
	SendKeepAlive() <-chan KeepAliveResponse
}

type gRpcClient struct {
	conn *grpc.ClientConn
	RpcServiceClient
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
		return nil // TODO return useful error msg
	} else {
		return nil
	}

}

func newGRpcClient(host string) (*gRpcClient, error) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())

	if err != nil {
		return nil, err
	} else {
		for i := 1; i <= 20; i++ {
			s, sErr := conn.State()

			if sErr != nil {
				return nil, sErr
			} else {
				if s == grpc.TransientFailure {
					return nil, errors.New(fmt.Sprintf("Failed to establish initial connection for host %v", host))
				} else if s == grpc.Ready {
					return &gRpcClient{conn: conn, RpcServiceClient: NewRpcServiceClient(conn)}, nil
				}

				time.Sleep(500 * time.Millisecond)
			}
		}
		return nil, errors.New("Initial connection failed to go to state READY")
	}
}

func (c *client) SendRequestVote(cancel <-chan struct{}) <-chan VoteResponse {
	// create a channel of size 4
	// put on each host of the host channel
	// if all 4 have finished executing close the response chan

	return nil
}

func (c *client) SendKeepAlive(cancel <-chan struct{}) <-chan KeepAliveResponse {
	for _, gRpcClient := range c.rpcClients {

		k := context.Background()

		resp, err := gRpcClient.KeepAlive(k, &KeepAliveRequest{})
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(resp.Term)
		}
	}
	return nil
}
