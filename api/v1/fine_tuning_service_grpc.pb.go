// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

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

// FineTuningServiceClient is the client API for FineTuningService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FineTuningServiceClient interface {
	CreateJob(ctx context.Context, in *CreateJobRequest, opts ...grpc.CallOption) (*Job, error)
	ListJobs(ctx context.Context, in *ListJobsRequest, opts ...grpc.CallOption) (*ListJobsResponse, error)
	GetJob(ctx context.Context, in *GetJobRequest, opts ...grpc.CallOption) (*Job, error)
	CancelJob(ctx context.Context, in *CancelJobRequest, opts ...grpc.CallOption) (*Job, error)
}

type fineTuningServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFineTuningServiceClient(cc grpc.ClientConnInterface) FineTuningServiceClient {
	return &fineTuningServiceClient{cc}
}

func (c *fineTuningServiceClient) CreateJob(ctx context.Context, in *CreateJobRequest, opts ...grpc.CallOption) (*Job, error) {
	out := new(Job)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningService/CreateJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fineTuningServiceClient) ListJobs(ctx context.Context, in *ListJobsRequest, opts ...grpc.CallOption) (*ListJobsResponse, error) {
	out := new(ListJobsResponse)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningService/ListJobs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fineTuningServiceClient) GetJob(ctx context.Context, in *GetJobRequest, opts ...grpc.CallOption) (*Job, error) {
	out := new(Job)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningService/GetJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fineTuningServiceClient) CancelJob(ctx context.Context, in *CancelJobRequest, opts ...grpc.CallOption) (*Job, error) {
	out := new(Job)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningService/CancelJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FineTuningServiceServer is the server API for FineTuningService service.
// All implementations must embed UnimplementedFineTuningServiceServer
// for forward compatibility
type FineTuningServiceServer interface {
	CreateJob(context.Context, *CreateJobRequest) (*Job, error)
	ListJobs(context.Context, *ListJobsRequest) (*ListJobsResponse, error)
	GetJob(context.Context, *GetJobRequest) (*Job, error)
	CancelJob(context.Context, *CancelJobRequest) (*Job, error)
	mustEmbedUnimplementedFineTuningServiceServer()
}

// UnimplementedFineTuningServiceServer must be embedded to have forward compatible implementations.
type UnimplementedFineTuningServiceServer struct {
}

func (UnimplementedFineTuningServiceServer) CreateJob(context.Context, *CreateJobRequest) (*Job, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateJob not implemented")
}
func (UnimplementedFineTuningServiceServer) ListJobs(context.Context, *ListJobsRequest) (*ListJobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListJobs not implemented")
}
func (UnimplementedFineTuningServiceServer) GetJob(context.Context, *GetJobRequest) (*Job, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetJob not implemented")
}
func (UnimplementedFineTuningServiceServer) CancelJob(context.Context, *CancelJobRequest) (*Job, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelJob not implemented")
}
func (UnimplementedFineTuningServiceServer) mustEmbedUnimplementedFineTuningServiceServer() {}

// UnsafeFineTuningServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FineTuningServiceServer will
// result in compilation errors.
type UnsafeFineTuningServiceServer interface {
	mustEmbedUnimplementedFineTuningServiceServer()
}

func RegisterFineTuningServiceServer(s grpc.ServiceRegistrar, srv FineTuningServiceServer) {
	s.RegisterService(&FineTuningService_ServiceDesc, srv)
}

func _FineTuningService_CreateJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningServiceServer).CreateJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningService/CreateJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningServiceServer).CreateJob(ctx, req.(*CreateJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FineTuningService_ListJobs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningServiceServer).ListJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningService/ListJobs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningServiceServer).ListJobs(ctx, req.(*ListJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FineTuningService_GetJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningServiceServer).GetJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningService/GetJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningServiceServer).GetJob(ctx, req.(*GetJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FineTuningService_CancelJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CancelJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningServiceServer).CancelJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningService/CancelJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningServiceServer).CancelJob(ctx, req.(*CancelJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// FineTuningService_ServiceDesc is the grpc.ServiceDesc for FineTuningService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FineTuningService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "llmariner.fine_tuning.server.v1.FineTuningService",
	HandlerType: (*FineTuningServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateJob",
			Handler:    _FineTuningService_CreateJob_Handler,
		},
		{
			MethodName: "ListJobs",
			Handler:    _FineTuningService_ListJobs_Handler,
		},
		{
			MethodName: "GetJob",
			Handler:    _FineTuningService_GetJob_Handler,
		},
		{
			MethodName: "CancelJob",
			Handler:    _FineTuningService_CancelJob_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/fine_tuning_service.proto",
}

// FineTuningWorkerServiceClient is the client API for FineTuningWorkerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FineTuningWorkerServiceClient interface {
	ListQueuedInternalJobs(ctx context.Context, in *ListQueuedInternalJobsRequest, opts ...grpc.CallOption) (*ListQueuedInternalJobsResponse, error)
	GetInternalJob(ctx context.Context, in *GetInternalJobRequest, opts ...grpc.CallOption) (*InternalJob, error)
	// UpdateJobPhase updates the job status depending on the phase.
	UpdateJobPhase(ctx context.Context, in *UpdateJobPhaseRequest, opts ...grpc.CallOption) (*UpdateJobPhaseResponse, error)
}

type fineTuningWorkerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFineTuningWorkerServiceClient(cc grpc.ClientConnInterface) FineTuningWorkerServiceClient {
	return &fineTuningWorkerServiceClient{cc}
}

func (c *fineTuningWorkerServiceClient) ListQueuedInternalJobs(ctx context.Context, in *ListQueuedInternalJobsRequest, opts ...grpc.CallOption) (*ListQueuedInternalJobsResponse, error) {
	out := new(ListQueuedInternalJobsResponse)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningWorkerService/ListQueuedInternalJobs", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fineTuningWorkerServiceClient) GetInternalJob(ctx context.Context, in *GetInternalJobRequest, opts ...grpc.CallOption) (*InternalJob, error) {
	out := new(InternalJob)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningWorkerService/GetInternalJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *fineTuningWorkerServiceClient) UpdateJobPhase(ctx context.Context, in *UpdateJobPhaseRequest, opts ...grpc.CallOption) (*UpdateJobPhaseResponse, error) {
	out := new(UpdateJobPhaseResponse)
	err := c.cc.Invoke(ctx, "/llmariner.fine_tuning.server.v1.FineTuningWorkerService/UpdateJobPhase", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FineTuningWorkerServiceServer is the server API for FineTuningWorkerService service.
// All implementations must embed UnimplementedFineTuningWorkerServiceServer
// for forward compatibility
type FineTuningWorkerServiceServer interface {
	ListQueuedInternalJobs(context.Context, *ListQueuedInternalJobsRequest) (*ListQueuedInternalJobsResponse, error)
	GetInternalJob(context.Context, *GetInternalJobRequest) (*InternalJob, error)
	// UpdateJobPhase updates the job status depending on the phase.
	UpdateJobPhase(context.Context, *UpdateJobPhaseRequest) (*UpdateJobPhaseResponse, error)
	mustEmbedUnimplementedFineTuningWorkerServiceServer()
}

// UnimplementedFineTuningWorkerServiceServer must be embedded to have forward compatible implementations.
type UnimplementedFineTuningWorkerServiceServer struct {
}

func (UnimplementedFineTuningWorkerServiceServer) ListQueuedInternalJobs(context.Context, *ListQueuedInternalJobsRequest) (*ListQueuedInternalJobsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListQueuedInternalJobs not implemented")
}
func (UnimplementedFineTuningWorkerServiceServer) GetInternalJob(context.Context, *GetInternalJobRequest) (*InternalJob, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInternalJob not implemented")
}
func (UnimplementedFineTuningWorkerServiceServer) UpdateJobPhase(context.Context, *UpdateJobPhaseRequest) (*UpdateJobPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateJobPhase not implemented")
}
func (UnimplementedFineTuningWorkerServiceServer) mustEmbedUnimplementedFineTuningWorkerServiceServer() {
}

// UnsafeFineTuningWorkerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FineTuningWorkerServiceServer will
// result in compilation errors.
type UnsafeFineTuningWorkerServiceServer interface {
	mustEmbedUnimplementedFineTuningWorkerServiceServer()
}

func RegisterFineTuningWorkerServiceServer(s grpc.ServiceRegistrar, srv FineTuningWorkerServiceServer) {
	s.RegisterService(&FineTuningWorkerService_ServiceDesc, srv)
}

func _FineTuningWorkerService_ListQueuedInternalJobs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListQueuedInternalJobsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningWorkerServiceServer).ListQueuedInternalJobs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningWorkerService/ListQueuedInternalJobs",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningWorkerServiceServer).ListQueuedInternalJobs(ctx, req.(*ListQueuedInternalJobsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FineTuningWorkerService_GetInternalJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetInternalJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningWorkerServiceServer).GetInternalJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningWorkerService/GetInternalJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningWorkerServiceServer).GetInternalJob(ctx, req.(*GetInternalJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FineTuningWorkerService_UpdateJobPhase_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateJobPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FineTuningWorkerServiceServer).UpdateJobPhase(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/llmariner.fine_tuning.server.v1.FineTuningWorkerService/UpdateJobPhase",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FineTuningWorkerServiceServer).UpdateJobPhase(ctx, req.(*UpdateJobPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// FineTuningWorkerService_ServiceDesc is the grpc.ServiceDesc for FineTuningWorkerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FineTuningWorkerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "llmariner.fine_tuning.server.v1.FineTuningWorkerService",
	HandlerType: (*FineTuningWorkerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListQueuedInternalJobs",
			Handler:    _FineTuningWorkerService_ListQueuedInternalJobs_Handler,
		},
		{
			MethodName: "GetInternalJob",
			Handler:    _FineTuningWorkerService_GetInternalJob_Handler,
		},
		{
			MethodName: "UpdateJobPhase",
			Handler:    _FineTuningWorkerService_UpdateJobPhase_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/fine_tuning_service.proto",
}
