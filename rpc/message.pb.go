// Code generated by protoc-gen-go.
// source: message.proto
// DO NOT EDIT!

/*
Package rpc is a generated protocol buffer package.

It is generated from these files:
	message.proto
	rpc_service.proto

It has these top-level messages:
	VoteRequest
	VoteResponse
	LeaderInfo
	KeepAliveRequest
	KeepAliveResponse
	AppendConfigEntryRequest
	AppendLogEntryRequest
	AppendEntryResponse
	InstallSnapshotRequest
	InstallSnapshotResponse
*/
package rpc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type VoteRequest struct {
	Term         uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	LastLogIndex uint32 `protobuf:"varint,2,opt,name=last_log_index,json=lastLogIndex" json:"last_log_index,omitempty"`
	LastLogTerm  uint32 `protobuf:"varint,3,opt,name=last_log_term,json=lastLogTerm" json:"last_log_term,omitempty"`
	Id           string `protobuf:"bytes,4,opt,name=id" json:"id,omitempty"`
}

func (m *VoteRequest) Reset()                    { *m = VoteRequest{} }
func (m *VoteRequest) String() string            { return proto.CompactTextString(m) }
func (*VoteRequest) ProtoMessage()               {}
func (*VoteRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type VoteResponse struct {
	Term        uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	VoteGranted bool   `protobuf:"varint,2,opt,name=vote_granted,json=voteGranted" json:"vote_granted,omitempty"`
}

func (m *VoteResponse) Reset()                    { *m = VoteResponse{} }
func (m *VoteResponse) String() string            { return proto.CompactTextString(m) }
func (*VoteResponse) ProtoMessage()               {}
func (*VoteResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type LeaderInfo struct {
	Term         uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	PrevLogIndex uint32 `protobuf:"varint,2,opt,name=prev_log_index,json=prevLogIndex" json:"prev_log_index,omitempty"`
	PrevLogTerm  uint32 `protobuf:"varint,3,opt,name=prev_log_term,json=prevLogTerm" json:"prev_log_term,omitempty"`
	CommitIndex  uint32 `protobuf:"varint,4,opt,name=commit_index,json=commitIndex" json:"commit_index,omitempty"`
	Id           string `protobuf:"bytes,5,opt,name=id" json:"id,omitempty"`
}

func (m *LeaderInfo) Reset()                    { *m = LeaderInfo{} }
func (m *LeaderInfo) String() string            { return proto.CompactTextString(m) }
func (*LeaderInfo) ProtoMessage()               {}
func (*LeaderInfo) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type KeepAliveRequest struct {
	LeaderInfo *LeaderInfo `protobuf:"bytes,1,opt,name=leader_info,json=leaderInfo" json:"leader_info,omitempty"`
}

func (m *KeepAliveRequest) Reset()                    { *m = KeepAliveRequest{} }
func (m *KeepAliveRequest) String() string            { return proto.CompactTextString(m) }
func (*KeepAliveRequest) ProtoMessage()               {}
func (*KeepAliveRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *KeepAliveRequest) GetLeaderInfo() *LeaderInfo {
	if m != nil {
		return m.LeaderInfo
	}
	return nil
}

type KeepAliveResponse struct {
	Term uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
}

func (m *KeepAliveResponse) Reset()                    { *m = KeepAliveResponse{} }
func (m *KeepAliveResponse) String() string            { return proto.CompactTextString(m) }
func (*KeepAliveResponse) ProtoMessage()               {}
func (*KeepAliveResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

type AppendConfigEntryRequest struct {
	LeaderInfo    *LeaderInfo                               `protobuf:"bytes,1,opt,name=leader_info,json=leaderInfo" json:"leader_info,omitempty"`
	NodeLocations map[string]*AppendConfigEntryRequest_Node `protobuf:"bytes,2,rep,name=node_locations,json=nodeLocations" json:"node_locations,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *AppendConfigEntryRequest) Reset()                    { *m = AppendConfigEntryRequest{} }
func (m *AppendConfigEntryRequest) String() string            { return proto.CompactTextString(m) }
func (*AppendConfigEntryRequest) ProtoMessage()               {}
func (*AppendConfigEntryRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *AppendConfigEntryRequest) GetLeaderInfo() *LeaderInfo {
	if m != nil {
		return m.LeaderInfo
	}
	return nil
}

func (m *AppendConfigEntryRequest) GetNodeLocations() map[string]*AppendConfigEntryRequest_Node {
	if m != nil {
		return m.NodeLocations
	}
	return nil
}

type AppendConfigEntryRequest_Node struct {
	Host string `protobuf:"bytes,1,opt,name=host" json:"host,omitempty"`
	Port uint32 `protobuf:"varint,2,opt,name=port" json:"port,omitempty"`
	Id   string `protobuf:"bytes,3,opt,name=id" json:"id,omitempty"`
}

func (m *AppendConfigEntryRequest_Node) Reset()         { *m = AppendConfigEntryRequest_Node{} }
func (m *AppendConfigEntryRequest_Node) String() string { return proto.CompactTextString(m) }
func (*AppendConfigEntryRequest_Node) ProtoMessage()    {}
func (*AppendConfigEntryRequest_Node) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{5, 1}
}

type AppendLogEntryRequest struct {
	LeaderInfo *LeaderInfo                       `protobuf:"bytes,1,opt,name=leader_info,json=leaderInfo" json:"leader_info,omitempty"`
	Entries    []*AppendLogEntryRequest_LogEntry `protobuf:"bytes,2,rep,name=entries" json:"entries,omitempty"`
}

func (m *AppendLogEntryRequest) Reset()                    { *m = AppendLogEntryRequest{} }
func (m *AppendLogEntryRequest) String() string            { return proto.CompactTextString(m) }
func (*AppendLogEntryRequest) ProtoMessage()               {}
func (*AppendLogEntryRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *AppendLogEntryRequest) GetLeaderInfo() *LeaderInfo {
	if m != nil {
		return m.LeaderInfo
	}
	return nil
}

func (m *AppendLogEntryRequest) GetEntries() []*AppendLogEntryRequest_LogEntry {
	if m != nil {
		return m.Entries
	}
	return nil
}

type AppendLogEntryRequest_LogEntry struct {
	Cmd   string `protobuf:"bytes,1,opt,name=cmd" json:"cmd,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *AppendLogEntryRequest_LogEntry) Reset()         { *m = AppendLogEntryRequest_LogEntry{} }
func (m *AppendLogEntryRequest_LogEntry) String() string { return proto.CompactTextString(m) }
func (*AppendLogEntryRequest_LogEntry) ProtoMessage()    {}
func (*AppendLogEntryRequest_LogEntry) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{6, 0}
}

type AppendEntryResponse struct {
	Term    uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	Success bool   `protobuf:"varint,2,opt,name=success" json:"success,omitempty"`
}

func (m *AppendEntryResponse) Reset()                    { *m = AppendEntryResponse{} }
func (m *AppendEntryResponse) String() string            { return proto.CompactTextString(m) }
func (*AppendEntryResponse) ProtoMessage()               {}
func (*AppendEntryResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

type InstallSnapshotRequest struct {
	Term              uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	Id                string `protobuf:"bytes,2,opt,name=id" json:"id,omitempty"`
	LastIncludedIndex uint32 `protobuf:"varint,3,opt,name=last_included_index,json=lastIncludedIndex" json:"last_included_index,omitempty"`
	LastIncludedTerm  uint32 `protobuf:"varint,4,opt,name=last_included_term,json=lastIncludedTerm" json:"last_included_term,omitempty"`
	Offset            uint32 `protobuf:"varint,5,opt,name=offset" json:"offset,omitempty"`
	Data              []byte `protobuf:"bytes,6,opt,name=data,proto3" json:"data,omitempty"`
	Done              bool   `protobuf:"varint,7,opt,name=done" json:"done,omitempty"`
}

func (m *InstallSnapshotRequest) Reset()                    { *m = InstallSnapshotRequest{} }
func (m *InstallSnapshotRequest) String() string            { return proto.CompactTextString(m) }
func (*InstallSnapshotRequest) ProtoMessage()               {}
func (*InstallSnapshotRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

type InstallSnapshotResponse struct {
	Term uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
}

func (m *InstallSnapshotResponse) Reset()                    { *m = InstallSnapshotResponse{} }
func (m *InstallSnapshotResponse) String() string            { return proto.CompactTextString(m) }
func (*InstallSnapshotResponse) ProtoMessage()               {}
func (*InstallSnapshotResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func init() {
	proto.RegisterType((*VoteRequest)(nil), "rpc.VoteRequest")
	proto.RegisterType((*VoteResponse)(nil), "rpc.VoteResponse")
	proto.RegisterType((*LeaderInfo)(nil), "rpc.LeaderInfo")
	proto.RegisterType((*KeepAliveRequest)(nil), "rpc.KeepAliveRequest")
	proto.RegisterType((*KeepAliveResponse)(nil), "rpc.KeepAliveResponse")
	proto.RegisterType((*AppendConfigEntryRequest)(nil), "rpc.AppendConfigEntryRequest")
	proto.RegisterType((*AppendConfigEntryRequest_Node)(nil), "rpc.AppendConfigEntryRequest.Node")
	proto.RegisterType((*AppendLogEntryRequest)(nil), "rpc.AppendLogEntryRequest")
	proto.RegisterType((*AppendLogEntryRequest_LogEntry)(nil), "rpc.AppendLogEntryRequest.LogEntry")
	proto.RegisterType((*AppendEntryResponse)(nil), "rpc.AppendEntryResponse")
	proto.RegisterType((*InstallSnapshotRequest)(nil), "rpc.InstallSnapshotRequest")
	proto.RegisterType((*InstallSnapshotResponse)(nil), "rpc.InstallSnapshotResponse")
}

var fileDescriptor0 = []byte{
	// 578 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xa4, 0x54, 0xc1, 0x6e, 0xd3, 0x40,
	0x10, 0x95, 0x93, 0xb4, 0x69, 0xc7, 0x49, 0x48, 0xb7, 0x50, 0xac, 0x9c, 0xc0, 0x20, 0xc1, 0x01,
	0xac, 0x2a, 0x5c, 0x2a, 0x24, 0x90, 0xaa, 0x52, 0xa1, 0x88, 0x88, 0x83, 0xa9, 0xe0, 0x18, 0x19,
	0x7b, 0x92, 0x5a, 0x38, 0xbb, 0xc6, 0xde, 0x44, 0x54, 0x7c, 0x0b, 0x7f, 0xc2, 0x1f, 0xf0, 0x07,
	0x7c, 0x0d, 0xbb, 0xb3, 0x6b, 0x63, 0x08, 0x29, 0x12, 0xdc, 0xc6, 0x6f, 0x67, 0xde, 0xcc, 0x7b,
	0xeb, 0x59, 0xe8, 0x2f, 0xb1, 0x2c, 0xa3, 0x05, 0x06, 0x79, 0x21, 0xa4, 0x60, 0xed, 0x22, 0x8f,
	0xfd, 0xcf, 0xe0, 0xbe, 0x15, 0x12, 0x43, 0xfc, 0xb8, 0xc2, 0x52, 0x32, 0x06, 0x1d, 0x89, 0xc5,
	0xd2, 0x73, 0xee, 0x38, 0x0f, 0xfb, 0x21, 0xc5, 0xec, 0x3e, 0x0c, 0xb2, 0xa8, 0x94, 0xb3, 0x4c,
	0x2c, 0x66, 0x29, 0x4f, 0xf0, 0x93, 0xd7, 0xa2, 0xd3, 0x9e, 0x46, 0xa7, 0x62, 0x31, 0xd1, 0x18,
	0xf3, 0xa1, 0x5f, 0x67, 0x11, 0x45, 0x9b, 0x92, 0x5c, 0x9b, 0x74, 0xa1, 0x99, 0x06, 0xd0, 0x4a,
	0x13, 0xaf, 0xa3, 0x0e, 0xf6, 0x43, 0x15, 0xf9, 0xe7, 0xd0, 0x33, 0xcd, 0xcb, 0x5c, 0xf0, 0x12,
	0xff, 0xd8, 0xfd, 0x2e, 0xf4, 0xd6, 0x2a, 0x67, 0xb6, 0x28, 0x22, 0x2e, 0x31, 0xa1, 0xde, 0x7b,
	0xa1, 0xab, 0xb1, 0x97, 0x06, 0xf2, 0xbf, 0x38, 0x00, 0x53, 0x8c, 0x12, 0x2c, 0x26, 0x7c, 0x2e,
	0xb6, 0x69, 0xc8, 0x0b, 0x5c, 0x6f, 0x6a, 0xd0, 0x68, 0x53, 0x43, 0x9d, 0xd5, 0xd4, 0x60, 0x93,
	0x2e, 0xec, 0x3c, 0xb1, 0x58, 0x2e, 0x53, 0x69, 0x79, 0x3a, 0x26, 0xc5, 0x60, 0x86, 0xc6, 0xc8,
	0xdc, 0xa9, 0x65, 0xbe, 0x80, 0xe1, 0x2b, 0xc4, 0xfc, 0x34, 0x4b, 0xd7, 0xb5, 0xd1, 0xc7, 0xe0,
	0x66, 0x34, 0xb2, 0xa2, 0x99, 0x0b, 0x9a, 0xd5, 0x1d, 0xdf, 0x08, 0xd4, 0x95, 0x04, 0x3f, 0xa5,
	0x84, 0x90, 0xd5, 0xb1, 0xff, 0x00, 0x0e, 0x1a, 0x2c, 0xdb, 0x1d, 0xf3, 0xbf, 0xb5, 0xc0, 0x3b,
	0xcd, 0x73, 0xe4, 0xc9, 0x99, 0xe0, 0xf3, 0x74, 0x71, 0xce, 0x65, 0x71, 0xf5, 0xcf, 0x7d, 0xd9,
	0x3b, 0x18, 0x70, 0x91, 0xa0, 0x32, 0x25, 0x8e, 0x64, 0xaa, 0xba, 0x2a, 0xeb, 0xda, 0xaa, 0xe8,
	0x98, 0x8a, 0xb6, 0x35, 0x0a, 0x5e, 0xab, 0x9a, 0x69, 0x55, 0x62, 0x4e, 0xfa, 0xbc, 0x89, 0x8d,
	0x12, 0x60, 0x9b, 0x49, 0x6c, 0x08, 0xed, 0x0f, 0x78, 0x45, 0x83, 0xed, 0x87, 0x3a, 0x64, 0x27,
	0xb0, 0xb3, 0x8e, 0xb2, 0x15, 0xd2, 0x95, 0xb9, 0x63, 0xff, 0xef, 0x7d, 0x43, 0x53, 0xf0, 0xb4,
	0x75, 0xe2, 0x8c, 0x9e, 0x43, 0x47, 0x43, 0xda, 0xa9, 0x4b, 0x51, 0x4a, 0x4b, 0x4c, 0xb1, 0xc6,
	0x72, 0x51, 0x48, 0xfb, 0x2f, 0x50, 0x6c, 0x2f, 0xaf, 0x5d, 0x5f, 0xde, 0x57, 0x07, 0x6e, 0x99,
	0x66, 0xea, 0x0f, 0xf8, 0x4f, 0x2b, 0x9f, 0x41, 0x17, 0x15, 0x43, 0x8a, 0x95, 0x87, 0xf7, 0x1a,
	0x5a, 0x7e, 0xa3, 0x0f, 0xea, 0xef, 0xaa, 0x66, 0x34, 0x86, 0xbd, 0x0a, 0xd4, 0x36, 0xc5, 0xcb,
	0xa4, 0xb2, 0x49, 0x85, 0xec, 0x66, 0xd3, 0xa6, 0x9e, 0xb5, 0xc0, 0x3f, 0x83, 0x43, 0x43, 0x6f,
	0xb9, 0xaf, 0xd9, 0x34, 0x0f, 0xba, 0xe5, 0x2a, 0x8e, 0xd5, 0x1b, 0x61, 0x97, 0xac, 0xfa, 0xf4,
	0xbf, 0x3b, 0x70, 0x34, 0xe1, 0xa5, 0x8c, 0xb2, 0xec, 0x0d, 0x8f, 0xf2, 0xf2, 0x52, 0xc8, 0xeb,
	0x1e, 0x0c, 0x63, 0x61, 0xab, 0xb2, 0x90, 0x05, 0x70, 0x48, 0x4f, 0x43, 0xca, 0xe3, 0x6c, 0x95,
	0x60, 0x62, 0x37, 0xc7, 0x2c, 0xd7, 0x81, 0x3e, 0x9a, 0xd8, 0x13, 0xb3, 0x3f, 0x8f, 0x80, 0xfd,
	0x9a, 0x4f, 0x1d, 0xcc, 0xa2, 0x0d, 0x9b, 0xe9, 0xb4, 0x90, 0x47, 0xb0, 0x2b, 0xe6, 0xf3, 0x12,
	0x25, 0x6d, 0x5c, 0x3f, 0xb4, 0x5f, 0x7a, 0xb2, 0x24, 0x92, 0x91, 0xb7, 0x4b, 0x76, 0x50, 0x4c,
	0x98, 0xe0, 0xe8, 0x75, 0x49, 0x1f, 0xc5, 0xfe, 0x63, 0xb8, 0xbd, 0xa1, 0x6d, 0xbb, 0x4b, 0xef,
	0x77, 0xe9, 0xf1, 0x7c, 0xf2, 0x23, 0x00, 0x00, 0xff, 0xff, 0x5b, 0xfe, 0xcd, 0x21, 0x4d, 0x05,
	0x00, 0x00,
}