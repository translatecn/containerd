// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.20.1
// source: demo/api/services/sandbox/v1/sandbox.proto

package sandbox

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

// StoreClient is the client API for Store service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StoreClient interface {
	Create(ctx context.Context, in *StoreCreateRequest, opts ...grpc.CallOption) (*StoreCreateResponse, error)
	Update(ctx context.Context, in *StoreUpdateRequest, opts ...grpc.CallOption) (*StoreUpdateResponse, error)
	Delete(ctx context.Context, in *StoreDeleteRequest, opts ...grpc.CallOption) (*StoreDeleteResponse, error)
	List(ctx context.Context, in *StoreListRequest, opts ...grpc.CallOption) (*StoreListResponse, error)
	Get(ctx context.Context, in *StoreGetRequest, opts ...grpc.CallOption) (*StoreGetResponse, error)
}

type storeClient struct {
	cc grpc.ClientConnInterface
}

func NewStoreClient(cc grpc.ClientConnInterface) StoreClient {
	return &storeClient{cc}
}

func (c *storeClient) Create(ctx context.Context, in *StoreCreateRequest, opts ...grpc.CallOption) (*StoreCreateResponse, error) {
	out := new(StoreCreateResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Store/Create", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *storeClient) Update(ctx context.Context, in *StoreUpdateRequest, opts ...grpc.CallOption) (*StoreUpdateResponse, error) {
	out := new(StoreUpdateResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Store/Update", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *storeClient) Delete(ctx context.Context, in *StoreDeleteRequest, opts ...grpc.CallOption) (*StoreDeleteResponse, error) {
	out := new(StoreDeleteResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Store/Delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *storeClient) List(ctx context.Context, in *StoreListRequest, opts ...grpc.CallOption) (*StoreListResponse, error) {
	out := new(StoreListResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Store/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *storeClient) Get(ctx context.Context, in *StoreGetRequest, opts ...grpc.CallOption) (*StoreGetResponse, error) {
	out := new(StoreGetResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Store/Get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StoreServer is the server API for Store service.
// All implementations must embed UnimplementedStoreServer
// for forward compatibility
type StoreServer interface {
	Create(context.Context, *StoreCreateRequest) (*StoreCreateResponse, error)
	Update(context.Context, *StoreUpdateRequest) (*StoreUpdateResponse, error)
	Delete(context.Context, *StoreDeleteRequest) (*StoreDeleteResponse, error)
	List(context.Context, *StoreListRequest) (*StoreListResponse, error)
	Get(context.Context, *StoreGetRequest) (*StoreGetResponse, error)
	mustEmbedUnimplementedStoreServer()
}

// UnimplementedStoreServer must be embedded to have forward compatible implementations.
type UnimplementedStoreServer struct {
}

func (UnimplementedStoreServer) Create(context.Context, *StoreCreateRequest) (*StoreCreateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedStoreServer) Update(context.Context, *StoreUpdateRequest) (*StoreUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedStoreServer) Delete(context.Context, *StoreDeleteRequest) (*StoreDeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedStoreServer) List(context.Context, *StoreListRequest) (*StoreListResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedStoreServer) Get(context.Context, *StoreGetRequest) (*StoreGetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedStoreServer) mustEmbedUnimplementedStoreServer() {}

// UnsafeStoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StoreServer will
// result in compilation errors.
type UnsafeStoreServer interface {
	mustEmbedUnimplementedStoreServer()
}

func RegisterStoreServer(s grpc.ServiceRegistrar, srv StoreServer) {
	s.RegisterService(&Store_ServiceDesc, srv)
}

func _Store_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreCreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StoreServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Store/Create",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StoreServer).Create(ctx, req.(*StoreCreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Store_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreUpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StoreServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Store/Update",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StoreServer).Update(ctx, req.(*StoreUpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Store_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreDeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StoreServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Store/Delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StoreServer).Delete(ctx, req.(*StoreDeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Store_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StoreServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Store/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StoreServer).List(ctx, req.(*StoreListRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Store_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreGetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StoreServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Store/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StoreServer).Get(ctx, req.(*StoreGetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Store_ServiceDesc is the grpc.ServiceDesc for Store service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Store_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "containerd.services.sandbox.v1.Store",
	HandlerType: (*StoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Create",
			Handler:    _Store_Create_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Store_Update_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _Store_Delete_Handler,
		},
		{
			MethodName: "List",
			Handler:    _Store_List_Handler,
		},
		{
			MethodName: "Get",
			Handler:    _Store_Get_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "demo/over/api/services/sandbox/v1/sandbox.proto",
}

// ControllerClient is the client API for Controller service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ControllerClient interface {
	Create(ctx context.Context, in *ControllerCreateRequest, opts ...grpc.CallOption) (*ControllerCreateResponse, error)
	Start(ctx context.Context, in *ControllerStartRequest, opts ...grpc.CallOption) (*ControllerStartResponse, error)
	Platform(ctx context.Context, in *ControllerPlatformRequest, opts ...grpc.CallOption) (*ControllerPlatformResponse, error)
	Stop(ctx context.Context, in *ControllerStopRequest, opts ...grpc.CallOption) (*ControllerStopResponse, error)
	Wait(ctx context.Context, in *ControllerWaitRequest, opts ...grpc.CallOption) (*ControllerWaitResponse, error)
	Status(ctx context.Context, in *ControllerStatusRequest, opts ...grpc.CallOption) (*ControllerStatusResponse, error)
	Shutdown(ctx context.Context, in *ControllerShutdownRequest, opts ...grpc.CallOption) (*ControllerShutdownResponse, error)
}

type controllerClient struct {
	cc grpc.ClientConnInterface
}

func NewControllerClient(cc grpc.ClientConnInterface) ControllerClient {
	return &controllerClient{cc}
}

func (c *controllerClient) Create(ctx context.Context, in *ControllerCreateRequest, opts ...grpc.CallOption) (*ControllerCreateResponse, error) {
	out := new(ControllerCreateResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Create", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) Start(ctx context.Context, in *ControllerStartRequest, opts ...grpc.CallOption) (*ControllerStartResponse, error) {
	out := new(ControllerStartResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) Platform(ctx context.Context, in *ControllerPlatformRequest, opts ...grpc.CallOption) (*ControllerPlatformResponse, error) {
	out := new(ControllerPlatformResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Platform", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) Stop(ctx context.Context, in *ControllerStopRequest, opts ...grpc.CallOption) (*ControllerStopResponse, error) {
	out := new(ControllerStopResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) Wait(ctx context.Context, in *ControllerWaitRequest, opts ...grpc.CallOption) (*ControllerWaitResponse, error) {
	out := new(ControllerWaitResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Wait", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) Status(ctx context.Context, in *ControllerStatusRequest, opts ...grpc.CallOption) (*ControllerStatusResponse, error) {
	out := new(ControllerStatusResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *controllerClient) Shutdown(ctx context.Context, in *ControllerShutdownRequest, opts ...grpc.CallOption) (*ControllerShutdownResponse, error) {
	out := new(ControllerShutdownResponse)
	err := c.cc.Invoke(ctx, "/containerd.services.sandbox.v1.Controller/Shutdown", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ControllerServer is the server API for Controller service.
// All implementations must embed UnimplementedControllerServer
// for forward compatibility
type ControllerServer interface {
	Create(context.Context, *ControllerCreateRequest) (*ControllerCreateResponse, error)
	Start(context.Context, *ControllerStartRequest) (*ControllerStartResponse, error)
	Platform(context.Context, *ControllerPlatformRequest) (*ControllerPlatformResponse, error)
	Stop(context.Context, *ControllerStopRequest) (*ControllerStopResponse, error)
	Wait(context.Context, *ControllerWaitRequest) (*ControllerWaitResponse, error)
	Status(context.Context, *ControllerStatusRequest) (*ControllerStatusResponse, error)
	Shutdown(context.Context, *ControllerShutdownRequest) (*ControllerShutdownResponse, error)
	mustEmbedUnimplementedControllerServer()
}

// UnimplementedControllerServer must be embedded to have forward compatible implementations.
type UnimplementedControllerServer struct {
}

func (UnimplementedControllerServer) Create(context.Context, *ControllerCreateRequest) (*ControllerCreateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedControllerServer) Start(context.Context, *ControllerStartRequest) (*ControllerStartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedControllerServer) Platform(context.Context, *ControllerPlatformRequest) (*ControllerPlatformResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Platform not implemented")
}
func (UnimplementedControllerServer) Stop(context.Context, *ControllerStopRequest) (*ControllerStopResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedControllerServer) Wait(context.Context, *ControllerWaitRequest) (*ControllerWaitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Wait not implemented")
}
func (UnimplementedControllerServer) Status(context.Context, *ControllerStatusRequest) (*ControllerStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedControllerServer) Shutdown(context.Context, *ControllerShutdownRequest) (*ControllerShutdownResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Shutdown not implemented")
}
func (UnimplementedControllerServer) mustEmbedUnimplementedControllerServer() {}

// UnsafeControllerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ControllerServer will
// result in compilation errors.
type UnsafeControllerServer interface {
	mustEmbedUnimplementedControllerServer()
}

func RegisterControllerServer(s grpc.ServiceRegistrar, srv ControllerServer) {
	s.RegisterService(&Controller_ServiceDesc, srv)
}

func _Controller_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerCreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Create",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Create(ctx, req.(*ControllerCreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerStartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Start(ctx, req.(*ControllerStartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_Platform_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerPlatformRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Platform(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Platform",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Platform(ctx, req.(*ControllerPlatformRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerStopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Stop(ctx, req.(*ControllerStopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_Wait_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerWaitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Wait(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Wait",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Wait(ctx, req.(*ControllerWaitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Status(ctx, req.(*ControllerStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Controller_Shutdown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ControllerShutdownRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ControllerServer).Shutdown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/containerd.services.sandbox.v1.Controller/Shutdown",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ControllerServer).Shutdown(ctx, req.(*ControllerShutdownRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Controller_ServiceDesc is the grpc.ServiceDesc for Controller service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Controller_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "containerd.services.sandbox.v1.Controller",
	HandlerType: (*ControllerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Create",
			Handler:    _Controller_Create_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _Controller_Start_Handler,
		},
		{
			MethodName: "Platform",
			Handler:    _Controller_Platform_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _Controller_Stop_Handler,
		},
		{
			MethodName: "Wait",
			Handler:    _Controller_Wait_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _Controller_Status_Handler,
		},
		{
			MethodName: "Shutdown",
			Handler:    _Controller_Shutdown_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "demo/over/api/services/sandbox/v1/sandbox.proto",
}