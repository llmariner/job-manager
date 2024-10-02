// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package legacy

import (
	context "context"
	v1 "github.com/llmariner/job-manager/api/v1"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BatchWorkerServiceClient is the client API for BatchWorkerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BatchWorkerServiceClient interface {
	ListQueuedInternalBatchJobs(ctx context.Context, in *v1.ListQueuedInternalBatchJobsRequest, opts ...grpc.CallOption) (*v1.ListQueuedInternalBatchJobsResponse, error)
	GetInternalBatchJob(ctx context.Context, in *v1.GetInternalBatchJobRequest, opts ...grpc.CallOption) (*v1.InternalBatchJob, error)
	UpdateBatchJobState(ctx context.Context, in *v1.UpdateBatchJobStateRequest, opts ...grpc.CallOption) (*v1.UpdateBatchJobStateResponse, error)
}

type batchWorkerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBatchWorkerServiceClient(cc grpc.ClientConnInterface) BatchWorkerServiceClient {
	return &batchWorkerServiceClient{cc}
}

func (c *batchWorkerServiceClient) ListQueuedInternalBatchJobs(ctx context.Context, in *v1.ListQueuedInternalBatchJobsRequest, opts ...grpc.CallOption) (*v1.ListQueuedInternalBatchJobsResponse, error) {
	out := new(v1.ListQueuedInternalBatchJobsResponse)
	err := c.cc.Invoke(ctx, "/llmoperator.batch.server.v1.BatchWorkerService/ListQueuedInternalBatchJobs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *batchWorkerServiceClient) GetInternalBatchJob(ctx context.Context, in *v1.GetInternalBatchJobRequest, opts ...grpc.CallOption) (*v1.InternalBatchJob, error) {
	out := new(v1.InternalBatchJob)
	err := c.cc.Invoke(ctx, "/llmoperator.batch.server.v1.BatchWorkerService/GetInternalBatchJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *batchWorkerServiceClient) UpdateBatchJobState(ctx context.Context, in *v1.UpdateBatchJobStateRequest, opts ...grpc.CallOption) (*v1.UpdateBatchJobStateResponse, error) {
	out := new(v1.UpdateBatchJobStateResponse)
	err := c.cc.Invoke(ctx, "/llmoperator.batch.server.v1.BatchWorkerService/UpdateBatchJobState", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BatchWorkerServiceServer is the server API for BatchWorkerService service.
// All implementations must embed UnimplementedBatchWorkerServiceServer
// for forward compatibility
type BatchWorkerServiceServer interface {
	ListQueuedInternalBatchJobs(context.Context, *v1.ListQueuedInternalBatchJobsRequest) (*v1.ListQueuedInternalBatchJobsResponse, error)
	GetInternalBatchJob(context.Context, *v1.GetInternalBatchJobRequest) (*v1.InternalBatchJob, error)
	UpdateBatchJobState(context.Context, *v1.UpdateBatchJobStateRequest) (*v1.UpdateBatchJobStateResponse, error)
	mustEmbedUnimplementedBatchWorkerServiceServer()
}

// UnimplementedBatchWorkerServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBatchWorkerServiceServer struct {
}

func (UnimplementedBatchWorkerServiceServer) ListQueuedInternalBatchJobs(context.Context, *v1.ListQueuedInternalBatchJobsRequest) (*v1.ListQueuedInternalBatchJobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListQueuedInternalBatchJobs not implemented")
}
func (UnimplementedBatchWorkerServiceServer) GetInternalBatchJob(context.Context, *v1.GetInternalBatchJobRequest) (*v1.InternalBatchJob, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInternalBatchJob not implemented")
}
func (UnimplementedBatchWorkerServiceServer) UpdateBatchJobState(context.Context, *v1.UpdateBatchJobStateRequest) (*v1.UpdateBatchJobStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateBatchJobState not implemented")
}
func (UnimplementedBatchWorkerServiceServer) mustEmbedUnimplementedBatchWorkerServiceServer() {}

// UnsafeBatchWorkerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BatchWorkerServiceServer will
// result in compilation errors.
type UnsafeBatchWorkerServiceServer interface {
	mustEmbedUnimplementedBatchWorkerServiceServer()
}

func RegisterBatchWorkerServiceServer(s grpc.ServiceRegistrar, srv BatchWorkerServiceServer) {
	s.RegisterService(&BatchWorkerService_ServiceDesc, srv)
}

func _BatchWorkerService_ListQueuedInternalBatchJobs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(v1.ListQueuedInternalBatchJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchWorkerServiceServer).ListQueuedInternalBatchJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmoperator.batch.server.v1.BatchWorkerService/ListQueuedInternalBatchJobs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchWorkerServiceServer).ListQueuedInternalBatchJobs(ctx, req.(*v1.ListQueuedInternalBatchJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BatchWorkerService_GetInternalBatchJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(v1.GetInternalBatchJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchWorkerServiceServer).GetInternalBatchJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmoperator.batch.server.v1.BatchWorkerService/GetInternalBatchJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchWorkerServiceServer).GetInternalBatchJob(ctx, req.(*v1.GetInternalBatchJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BatchWorkerService_UpdateBatchJobState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(v1.UpdateBatchJobStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchWorkerServiceServer).UpdateBatchJobState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmoperator.batch.server.v1.BatchWorkerService/UpdateBatchJobState",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchWorkerServiceServer).UpdateBatchJobState(ctx, req.(*v1.UpdateBatchJobStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BatchWorkerService_ServiceDesc is the grpc.ServiceDesc for BatchWorkerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BatchWorkerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "llmoperator.batch.server.v1.BatchWorkerService",
	HandlerType: (*BatchWorkerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListQueuedInternalBatchJobs",
			Handler:    _BatchWorkerService_ListQueuedInternalBatchJobs_Handler,
		},
		{
			MethodName: "GetInternalBatchJob",
			Handler:    _BatchWorkerService_GetInternalBatchJob_Handler,
		},
		{
			MethodName: "UpdateBatchJobState",
			Handler:    _BatchWorkerService_UpdateBatchJobState_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/legacy/batch_service.proto",
}