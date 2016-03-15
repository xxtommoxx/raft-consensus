package rpc

import (
	"fmt"
	"github.com/xxtommoxx/raft-consensus/rpc/proto"
)

// translated protobuf 3 messages* to support embedded types
type Request interface {
	Term() uint32
	Id() string
}

type RequestHeader struct {
	term uint32
	id   string
}

func (r *RequestHeader) String() string {
	return fmt.Sprintf("RequestHeader { term: %v, id: %v }", r.term, r.id)
}

func (r *RequestHeader) Term() uint32 {
	return r.term
}

func (r *RequestHeader) Id() string {
	return r.id
}

func requestHeaderFromProto(r *proto.RequestHeader) *RequestHeader {
	return &RequestHeader{
		term: r.Term,
		id:   r.Id,
	}
}

type LogInfo struct {
	LastLogIndex uint32
	LastLogTerm  uint32
}

func (r *LogInfo) String() string {
	return fmt.Sprintf("LogInfo { LastLogIndex: %v, LastLogTerm: %v }", r.LastLogIndex, r.LastLogTerm)
}

func logInfoFromProto(r *proto.LogInfo) *LogInfo {
	return &LogInfo{
		LastLogIndex: r.LastLogIndex,
		LastLogTerm:  r.LastLogTerm,
	}
}

type LeaderInfo struct {
	*LogInfo
	CommitIndex uint32
}

func (r *LeaderInfo) String() string {
	return fmt.Sprintf("LeaderInfo { %v, CommitIndex: %v }", r.LogInfo, r.CommitIndex)
}

func leaderInfoFromProto(r *proto.LeaderInfo) *LeaderInfo {
	return &LeaderInfo{
		LogInfo:     logInfoFromProto(r.Log),
		CommitIndex: r.CommitIndex,
	}
}

type VoteRequest struct {
	*RequestHeader
	*LogInfo
}

func (r *VoteRequest) String() string {
	return fmt.Sprintf("VoteRequest { %v, %v }", r.RequestHeader, r.LogInfo)
}

func voteRequestFromProto(r *proto.VoteRequest) *VoteRequest {
	return &VoteRequest{
		RequestHeader: requestHeaderFromProto(r.Header),
		LogInfo:       logInfoFromProto(r.Log),
	}
}

type KeepAliveRequest struct {
	*RequestHeader
	*LeaderInfo
}

func keepAliveRequestFromProto(r *proto.KeepAliveRequest) *KeepAliveRequest {
	return &KeepAliveRequest{
		RequestHeader: requestHeaderFromProto(r.Header),
		LeaderInfo:    leaderInfoFromProto(r.Leader),
	}
}

func (r *KeepAliveRequest) String() string {
	return fmt.Sprintf("KeepAliveRequest { %v, %v }", r.RequestHeader, r.LeaderInfo)
}

type Response interface {
	Term() uint32
}

type ResponseHeader struct {
	term uint32
}

func (r *ResponseHeader) Term() uint32 {
	return r.term
}

func responseHeaderFromProto(r *proto.ResponseHeader) *ResponseHeader {
	return &ResponseHeader{
		term: r.Term,
	}
}

type KeepAliveResponse struct {
	*ResponseHeader
}

func keepAliveResponseFromProto(r *proto.KeepAliveResponse) *KeepAliveResponse {
	return &KeepAliveResponse{
		ResponseHeader: responseHeaderFromProto(r.Header),
	}
}

type VoteResponse struct {
	*ResponseHeader
	VoteGranted bool
}

func voteResponseFromProto(r *proto.VoteResponse) *VoteResponse {
	return &VoteResponse{
		ResponseHeader: responseHeaderFromProto(r.Header),
		VoteGranted:    r.VoteGranted,
	}
}
