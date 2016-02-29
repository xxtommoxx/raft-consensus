package rpc

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
)

type server struct {
	*common.SyncService
	host string

	counter uint32

	grpcServer *grpc.Server
	listener   net.Listener
}

func NewServer(host string) *server {
	server := &server{host: host}
	server.SyncService = common.NewSyncService(server.syncStart, server.asyncStart, server.syncStop)
	return server
}

func (s *server) syncStart() error {
	log.Info("Starting rpc server using host:", s.host)

	lis, err := net.Listen("tcp", s.host)

	if err != nil {
		log.Error("GRpc server failed to start:", err)
		return err
	} else {
		grpcServer := grpc.NewServer()
		s.grpcServer = grpcServer
		s.listener = lis
		RegisterRpcServiceServer(grpcServer, s)

		return nil
	}
}

func (s *server) asyncStart() {
	log.Error("Serve error:", s.grpcServer.Serve(s.listener))
}

func (s *server) syncStop() error {
	log.Info("Stopping rpc server")
	s.grpcServer.Stop()
	return nil
}

func (s *server) KeepAlive(ctx context.Context, req *KeepAliveRequest) (*KeepAliveResponse, error) {
	count := s.counter + 1

	s.counter++

	return &KeepAliveResponse{count}, nil

}

func (s *server) AppendLogEntries(ctx context.Context, req *AppendLogEntryRequest) (*AppendEntryResponse, error) {
	return nil, nil
}

func (s *server) UpdateConfiguration(ctx context.Context, req *AppendConfigEntryRequest) (*AppendEntryResponse, error) {
	return nil, nil
}

func (s *server) ElectLeader(ctx context.Context, req *VoteRequest) (*VoteResponse, error) {
	return nil, nil
}

func (s *server) InstallSnapshot(ctx context.Context, req *InstallSnapshotRequest) (*InstallSnapshotResponse, error) {
	return nil, nil
}
