package rpc

import (
	"github.com/Sirupsen/logrus"
	"github.com/xxtommoxx/raft-consensus/common"
	"github.com/xxtommoxx/raft-consensus/rpc/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
)

type RequestHandler interface {
	RequestVote(req *VoteRequest) (<-chan bool, <-chan error)
	KeepAlive(req *KeepAliveRequest) (<-chan struct{}, <-chan error)
}

func NewServer(nodeConfig common.NodeConfig, requestHandler RequestHandler,
	stateStore common.StateStore) common.Service {
	return newGrpcServer(nodeConfig, requestHandler, stateStore)
}

type grpcServer struct {
	*common.SyncService
	nodeConfig common.NodeConfig

	grpcServer *grpc.Server
	listener   net.Listener

	requestHandler RequestHandler
	stateStore     common.StateStore

	log *logrus.Entry
}

func newGrpcServer(nodeConfig common.NodeConfig, requestHandler RequestHandler,
	stateStore common.StateStore) *grpcServer {
	grpcServer := &grpcServer{
		log:            logrus.WithField("id", nodeConfig.Id),
		nodeConfig:     nodeConfig,
		requestHandler: requestHandler,
		stateStore:     stateStore,
	}

	grpcServer.SyncService = common.NewSyncService(grpcServer.syncStart, grpcServer.asyncStart, grpcServer.syncStop)
	return grpcServer
}

func (s *grpcServer) syncStart() error {
	s.log.Infof("Starting grpcServer using %v", s.nodeConfig.Host)

	lis, err := net.Listen("tcp", s.nodeConfig.Host)

	if err != nil {
		s.log.Error("grpcServer failed to start:", err)
		return err
	} else {
		grpcServer := grpc.NewServer()
		s.grpcServer = grpcServer
		s.listener = lis
		proto.RegisterRpcServiceServer(grpcServer, s)

		return nil
	}
}

func (s *grpcServer) asyncStart() {
	s.log.Warn("gRpc serve error:", s.grpcServer.Serve(s.listener))
}

func (s *grpcServer) syncStop() error {
	s.log.Info("Stopping grpcServer")
	s.grpcServer.Stop()
	return nil
}

func (s grpcServer) newResponseHeader() *proto.ResponseHeader {
	return &proto.ResponseHeader{
		Term: s.stateStore.CurrentTerm(),
	}
}

func (s *grpcServer) newKeepAliveResponse() *proto.KeepAliveResponse {
	return &proto.KeepAliveResponse{
		Header: s.newResponseHeader(),
	}
}

func (s *grpcServer) newVoteResponse(voteGranted bool) *proto.VoteResponse {
	return &proto.VoteResponse{
		Header:      s.newResponseHeader(),
		VoteGranted: voteGranted,
	}

}

func (s *grpcServer) KeepAlive(ctx context.Context, req *proto.KeepAliveRequest) (*proto.KeepAliveResponse, error) {
	_, errCh := s.requestHandler.KeepAlive(keepAliveRequestFromProto(req))
	err := <-errCh
	return s.newKeepAliveResponse(), err
}

func (s *grpcServer) ElectLeader(ctx context.Context, req *proto.VoteRequest) (*proto.VoteResponse, error) {
	voteObtainedCh, errCh := s.requestHandler.RequestVote(voteRequestFromProto(req))

	select {
	case voteGranted := <-voteObtainedCh:
		return s.newVoteResponse(voteGranted), nil
	case err := <-errCh:
		return s.newVoteResponse(false), err
	}
}
