package rpc

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"time"
)

const (
	healthyConn = iota
	unhealthyConn
)

type grpcConnInfo struct {
	id    string
	state int
}

type grpcClient struct {
	id string

	*common.SyncService

	peer       common.NodeConfig
	conn       *grpc.ClientConn
	underlying proto.RpcServiceClient

	connInfoCh chan grpcConnInfo
}

func newGrpcClient(id string, peerConfig common.NodeConfig, connInfoCh chan grpcConnInfo) *grpcClient {
	c := &grpcClient{
		id:         id,
		peer:       peerConfig,
		connInfoCh: connInfoCh,
	}

	c.SyncService = common.NewSyncService(c.syncStart, nil, c.syncStop)

	return c
}

func (p *grpcClient) syncStart() error {
	conn, err := grpc.Dial(p.peer.Host, grpc.WithInsecure())

	if err != nil {
		log.Error(err)
		return err
	} else {
		p.conn = conn

		for i := 1; i <= 20; i++ {
			s, sErr := conn.State()

			if sErr != nil {
				return sErr
			} else if s == grpc.Ready {
				p.underlying = proto.NewRpcServiceClient(conn)
				go p.monitorConnFailure()
				return nil
			} else {
				time.Sleep(500 * time.Millisecond)
			}

		}

		return errors.New(fmt.Sprintf("Failed to establish initial connection for host %v", p.peer.Host))
	}
}

func (p *grpcClient) waitForConn(anyOtherState grpc.ConnectivityState, fn func(grpc.ConnectivityState)) {
	nextState, err := p.conn.WaitForStateChange(context.TODO(), anyOtherState)
	if err != nil {
		log.Error(err) // todo might want to handle this in an elegant way
	} else {
		fn(nextState)
	}
}

func (p *grpcClient) monitorConnReady(currState grpc.ConnectivityState) {
	p.waitForConn(currState, func(state grpc.ConnectivityState) {
		if state == grpc.Ready {
			p.connInfoCh <- grpcConnInfo{
				state: healthyConn,
				id:    p.peer.Id,
			}

			p.monitorConnFailure()

		} else if state == grpc.TransientFailure || state == grpc.Connecting {
			go p.monitorConnReady(state)
		}
	})
}

func (p *grpcClient) monitorConnFailure() {
	p.waitForConn(grpc.Ready, func(state grpc.ConnectivityState) {
		if state == grpc.TransientFailure || state == grpc.Connecting {
			p.connInfoCh <- grpcConnInfo{
				state: unhealthyConn,
				id:    p.peer.Id,
			}

			p.monitorConnReady(state)
		}
	})
}

func (p *grpcClient) syncStop() error {
	p.underlying = nil
	return p.conn.Close()
}

func (p *grpcClient) newRequestHeader(term uint32) *proto.RequestHeader {
	return &proto.RequestHeader{
		Term: term,
		Id:   p.id,
	}
}

func (p *grpcClient) context() context.Context {
	return context.Background()
}

func (p *grpcClient) newLogInfo() *proto.LogInfo {
	return &proto.LogInfo{
		LastLogIndex: 0, // todo
		LastLogTerm:  0,
	}
}

func (p *grpcClient) newLeaderInfo() *proto.LeaderInfo {
	return &proto.LeaderInfo{
		Log:         p.newLogInfo(),
		CommitIndex: 0, // todo
	}
}

func (p *grpcClient) requestVote(term uint32) (*VoteResponse, error) {
	req := &proto.VoteRequest{
		Header: p.newRequestHeader(term),
		Log:    p.newLogInfo(),
	}

	resp, err := p.underlying.ElectLeader(p.context(), req)
	if err != nil {
		return nil, err
	} else {
		return voteResponseFromProto(resp), nil
	}
}

func (p *grpcClient) keepAlive(term uint32) (*KeepAliveResponse, error) {
	req := &proto.KeepAliveRequest{
		Header: p.newRequestHeader(term),
		Leader: p.newLeaderInfo(),
	}

	resp, err := p.underlying.KeepAlive(p.context(), req)

	if err != nil {
		return nil, err
	} else {
		return keepAliveResponseFromProto(resp), nil
	}
}
