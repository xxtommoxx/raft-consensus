// Code generated by protoc-gen-go.
// source: install_snapshot.proto
// DO NOT EDIT!

package rpc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type InstallSnapshotRequest struct {
	Term              uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
	LeaderId          string `protobuf:"bytes,2,opt,name=leader_id" json:"leader_id,omitempty"`
	LastIncludedIndex uint32 `protobuf:"varint,3,opt,name=last_included_index" json:"last_included_index,omitempty"`
	LastIncludedTerm  uint32 `protobuf:"varint,4,opt,name=last_included_term" json:"last_included_term,omitempty"`
	Offset            uint32 `protobuf:"varint,5,opt,name=offset" json:"offset,omitempty"`
	Data              []byte `protobuf:"bytes,6,opt,name=data,proto3" json:"data,omitempty"`
	Done              bool   `protobuf:"varint,7,opt,name=done" json:"done,omitempty"`
}

func (m *InstallSnapshotRequest) Reset()         { *m = InstallSnapshotRequest{} }
func (m *InstallSnapshotRequest) String() string { return proto.CompactTextString(m) }
func (*InstallSnapshotRequest) ProtoMessage()    {}

type InstallSnapshotResponse struct {
	Term uint32 `protobuf:"varint,1,opt,name=term" json:"term,omitempty"`
}

func (m *InstallSnapshotResponse) Reset()         { *m = InstallSnapshotResponse{} }
func (m *InstallSnapshotResponse) String() string { return proto.CompactTextString(m) }
func (*InstallSnapshotResponse) ProtoMessage()    {}