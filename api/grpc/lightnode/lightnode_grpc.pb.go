// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.23.4
// source: lightnode/lightnode.proto

package lightnode

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	LightNode_StreamBlobAvailability_FullMethodName = "/lightnode.LightNode/StreamBlobAvailability"
)

// LightNodeClient is the client API for LightNode service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LightNodeClient interface {
	// StreamBlobAvailability streams the availability status blobs from the light node's perspective.
	// A light node considers a blob to be available if all chunks it wants to sample are available.
	// This API is for use by a DA node for monitoring the availability of chunks through its
	// constellation of agent light nodes.
	StreamBlobAvailability(ctx context.Context, in *StreamChunkAvailabilityRequest, opts ...grpc.CallOption) (LightNode_StreamBlobAvailabilityClient, error)
}

type lightNodeClient struct {
	cc grpc.ClientConnInterface
}

func NewLightNodeClient(cc grpc.ClientConnInterface) LightNodeClient {
	return &lightNodeClient{cc}
}

func (c *lightNodeClient) StreamBlobAvailability(ctx context.Context, in *StreamChunkAvailabilityRequest, opts ...grpc.CallOption) (LightNode_StreamBlobAvailabilityClient, error) {
	stream, err := c.cc.NewStream(ctx, &LightNode_ServiceDesc.Streams[0], LightNode_StreamBlobAvailability_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	x := &lightNodeStreamBlobAvailabilityClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type LightNode_StreamBlobAvailabilityClient interface {
	Recv() (*StreamChunkAvailabilityReply, error)
	grpc.ClientStream
}

type lightNodeStreamBlobAvailabilityClient struct {
	grpc.ClientStream
}

func (x *lightNodeStreamBlobAvailabilityClient) Recv() (*StreamChunkAvailabilityReply, error) {
	m := new(StreamChunkAvailabilityReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// LightNodeServer is the server API for LightNode service.
// All implementations must embed UnimplementedLightNodeServer
// for forward compatibility
type LightNodeServer interface {
	// StreamBlobAvailability streams the availability status blobs from the light node's perspective.
	// A light node considers a blob to be available if all chunks it wants to sample are available.
	// This API is for use by a DA node for monitoring the availability of chunks through its
	// constellation of agent light nodes.
	StreamBlobAvailability(*StreamChunkAvailabilityRequest, LightNode_StreamBlobAvailabilityServer) error
	mustEmbedUnimplementedLightNodeServer()
}

// UnimplementedLightNodeServer must be embedded to have forward compatible implementations.
type UnimplementedLightNodeServer struct {
}

func (UnimplementedLightNodeServer) StreamBlobAvailability(*StreamChunkAvailabilityRequest, LightNode_StreamBlobAvailabilityServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamBlobAvailability not implemented")
}
func (UnimplementedLightNodeServer) mustEmbedUnimplementedLightNodeServer() {}

// UnsafeLightNodeServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LightNodeServer will
// result in compilation errors.
type UnsafeLightNodeServer interface {
	mustEmbedUnimplementedLightNodeServer()
}

func RegisterLightNodeServer(s grpc.ServiceRegistrar, srv LightNodeServer) {
	s.RegisterService(&LightNode_ServiceDesc, srv)
}

func _LightNode_StreamBlobAvailability_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamChunkAvailabilityRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(LightNodeServer).StreamBlobAvailability(m, &lightNodeStreamBlobAvailabilityServer{stream})
}

type LightNode_StreamBlobAvailabilityServer interface {
	Send(*StreamChunkAvailabilityReply) error
	grpc.ServerStream
}

type lightNodeStreamBlobAvailabilityServer struct {
	grpc.ServerStream
}

func (x *lightNodeStreamBlobAvailabilityServer) Send(m *StreamChunkAvailabilityReply) error {
	return x.ServerStream.SendMsg(m)
}

// LightNode_ServiceDesc is the grpc.ServiceDesc for LightNode service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LightNode_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "lightnode.LightNode",
	HandlerType: (*LightNodeServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamBlobAvailability",
			Handler:       _LightNode_StreamBlobAvailability_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "lightnode/lightnode.proto",
}
