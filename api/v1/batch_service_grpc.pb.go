// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: api/v1/batch_service.proto

package v1

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
	BatchService_CreateBatchJob_FullMethodName = "/llmoperator.batch.server.v1.BatchService/CreateBatchJob"
	BatchService_ListBatchJobs_FullMethodName  = "/llmoperator.batch.server.v1.BatchService/ListBatchJobs"
	BatchService_GetBatchJob_FullMethodName    = "/llmoperator.batch.server.v1.BatchService/GetBatchJob"
	BatchService_CancelBatchJob_FullMethodName = "/llmoperator.batch.server.v1.BatchService/CancelBatchJob"
)

// BatchServiceClient is the client API for BatchService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BatchServiceClient interface {
	CreateBatchJob(ctx context.Context, in *CreateBatchJobRequest, opts ...grpc.CallOption) (*BatchJob, error)
	ListBatchJobs(ctx context.Context, in *ListBatchJobsRequest, opts ...grpc.CallOption) (*ListBatchJobsResponse, error)
	GetBatchJob(ctx context.Context, in *GetBatchJobRequest, opts ...grpc.CallOption) (*BatchJob, error)
	CancelBatchJob(ctx context.Context, in *CancelBatchJobRequest, opts ...grpc.CallOption) (*BatchJob, error)
}

type batchServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBatchServiceClient(cc grpc.ClientConnInterface) BatchServiceClient {
	return &batchServiceClient{cc}
}

func (c *batchServiceClient) CreateBatchJob(ctx context.Context, in *CreateBatchJobRequest, opts ...grpc.CallOption) (*BatchJob, error) {
	out := new(BatchJob)
	err := c.cc.Invoke(ctx, BatchService_CreateBatchJob_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *batchServiceClient) ListBatchJobs(ctx context.Context, in *ListBatchJobsRequest, opts ...grpc.CallOption) (*ListBatchJobsResponse, error) {
	out := new(ListBatchJobsResponse)
	err := c.cc.Invoke(ctx, BatchService_ListBatchJobs_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *batchServiceClient) GetBatchJob(ctx context.Context, in *GetBatchJobRequest, opts ...grpc.CallOption) (*BatchJob, error) {
	out := new(BatchJob)
	err := c.cc.Invoke(ctx, BatchService_GetBatchJob_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *batchServiceClient) CancelBatchJob(ctx context.Context, in *CancelBatchJobRequest, opts ...grpc.CallOption) (*BatchJob, error) {
	out := new(BatchJob)
	err := c.cc.Invoke(ctx, BatchService_CancelBatchJob_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BatchServiceServer is the server API for BatchService service.
// All implementations must embed UnimplementedBatchServiceServer
// for forward compatibility
type BatchServiceServer interface {
	CreateBatchJob(context.Context, *CreateBatchJobRequest) (*BatchJob, error)
	ListBatchJobs(context.Context, *ListBatchJobsRequest) (*ListBatchJobsResponse, error)
	GetBatchJob(context.Context, *GetBatchJobRequest) (*BatchJob, error)
	CancelBatchJob(context.Context, *CancelBatchJobRequest) (*BatchJob, error)
	mustEmbedUnimplementedBatchServiceServer()
}

// UnimplementedBatchServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBatchServiceServer struct {
}

func (UnimplementedBatchServiceServer) CreateBatchJob(context.Context, *CreateBatchJobRequest) (*BatchJob, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBatchJob not implemented")
}
func (UnimplementedBatchServiceServer) ListBatchJobs(context.Context, *ListBatchJobsRequest) (*ListBatchJobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListBatchJobs not implemented")
}
func (UnimplementedBatchServiceServer) GetBatchJob(context.Context, *GetBatchJobRequest) (*BatchJob, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBatchJob not implemented")
}
func (UnimplementedBatchServiceServer) CancelBatchJob(context.Context, *CancelBatchJobRequest) (*BatchJob, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelBatchJob not implemented")
}
func (UnimplementedBatchServiceServer) mustEmbedUnimplementedBatchServiceServer() {}

// UnsafeBatchServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BatchServiceServer will
// result in compilation errors.
type UnsafeBatchServiceServer interface {
	mustEmbedUnimplementedBatchServiceServer()
}

func RegisterBatchServiceServer(s grpc.ServiceRegistrar, srv BatchServiceServer) {
	s.RegisterService(&BatchService_ServiceDesc, srv)
}

func _BatchService_CreateBatchJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateBatchJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchServiceServer).CreateBatchJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BatchService_CreateBatchJob_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchServiceServer).CreateBatchJob(ctx, req.(*CreateBatchJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BatchService_ListBatchJobs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListBatchJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchServiceServer).ListBatchJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BatchService_ListBatchJobs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchServiceServer).ListBatchJobs(ctx, req.(*ListBatchJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BatchService_GetBatchJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBatchJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchServiceServer).GetBatchJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BatchService_GetBatchJob_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchServiceServer).GetBatchJob(ctx, req.(*GetBatchJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BatchService_CancelBatchJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelBatchJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchServiceServer).CancelBatchJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BatchService_CancelBatchJob_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchServiceServer).CancelBatchJob(ctx, req.(*CancelBatchJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BatchService_ServiceDesc is the grpc.ServiceDesc for BatchService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BatchService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "llmoperator.batch.server.v1.BatchService",
	HandlerType: (*BatchServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateBatchJob",
			Handler:    _BatchService_CreateBatchJob_Handler,
		},
		{
			MethodName: "ListBatchJobs",
			Handler:    _BatchService_ListBatchJobs_Handler,
		},
		{
			MethodName: "GetBatchJob",
			Handler:    _BatchService_GetBatchJob_Handler,
		},
		{
			MethodName: "CancelBatchJob",
			Handler:    _BatchService_CancelBatchJob_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/batch_service.proto",
}