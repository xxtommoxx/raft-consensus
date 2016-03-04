package rpc

import (
	log "github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
)

type RequestHandler interface {
	RequestVote(req *VoteRequest) (<-chan *VoteResponse, <-chan error)
	KeepAlive(req *KeepAliveRequest) (<-chan *KeepAliveResponse, <-chan error)
}

type Server struct {
	grpcServer *grpcServer
}

func (s *Server) Stop() error {
	return s.grpcServer.Stop()
}

func (s *Server) Start() error {
	return s.grpcServer.Start()
}

func NewServer(host string, requestHandler RequestHandler) *Server {
	return &Server{newGrpcServer(host, requestHandler)}
}

type grpcServer struct {
	*common.SyncService
	host string

	grpcServer *grpc.Server
	listener   net.Listener

	requestHandler RequestHandler
}

func newGrpcServer(host string, requestHandler RequestHandler) *grpcServer {
	grpcServer := &grpcServer{host: host, requestHandler: requestHandler}
	grpcServer.SyncService = common.NewSyncService(grpcServer.syncStart, grpcServer.asyncStart, grpcServer.syncStop)
	return grpcServer
}

func (s *grpcServer) syncStart() error {
	log.Info("Starting rpc server using host:", s.host)

	lis, err := net.Listen("tcp", s.host)

	if err != nil {
		log.Error("rpc server failed to start:", err)
		return err
	} else {
		grpcServer := grpc.NewServer()
		s.grpcServer = grpcServer
		s.listener = lis
		RegisterRpcServiceServer(grpcServer, s)

		return nil
	}
}

func (s *grpcServer) asyncStart() {
	log.Error("Serve error:", s.grpcServer.Serve(s.listener))
}

func (s *grpcServer) syncStop() error {
	log.Info("Stopping rpc server")
	s.grpcServer.Stop()
	return nil
}

func (s *grpcServer) KeepAlive(ctx context.Context, req *KeepAliveRequest) (*KeepAliveResponse, error) {
	respCh, errCh := s.requestHandler.KeepAlive(req)

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

func (s *grpcServer) ElectLeader(ctx context.Context, req *VoteRequest) (*VoteResponse, error) {
	respCh, errCh := s.requestHandler.RequestVote(req)

	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

func (s *grpcServer) AppendLogEntries(ctx context.Context, req *AppendLogEntryRequest) (*AppendEntryResponse, error) {
	return nil, nil
}

func (s *grpcServer) UpdateConfiguration(ctx context.Context, req *AppendConfigEntryRequest) (*AppendEntryResponse, error) {
	return nil, nil
}

func (s *grpcServer) InstallSnapshot(ctx context.Context, req *InstallSnapshotRequest) (*InstallSnapshotResponse, error) {
	return nil, nil
}
