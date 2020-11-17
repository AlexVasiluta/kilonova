// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package grpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// EvalClient is the client API for Eval service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EvalClient interface {
	// Compile compiles a program, to be used for later execution
	Compile(ctx context.Context, in *CompileRequest, opts ...grpc.CallOption) (*CompileResponse, error)
	// Execute runs a stream of tests, returning their output
	// warning: the executable for the ID will be deleted after this is finished
	Execute(ctx context.Context, opts ...grpc.CallOption) (Eval_ExecuteClient, error)
}

type evalClient struct {
	cc grpc.ClientConnInterface
}

func NewEvalClient(cc grpc.ClientConnInterface) EvalClient {
	return &evalClient{cc}
}

func (c *evalClient) Compile(ctx context.Context, in *CompileRequest, opts ...grpc.CallOption) (*CompileResponse, error) {
	out := new(CompileResponse)
	err := c.cc.Invoke(ctx, "/eval.Eval/Compile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *evalClient) Execute(ctx context.Context, opts ...grpc.CallOption) (Eval_ExecuteClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Eval_serviceDesc.Streams[0], "/eval.Eval/Execute", opts...)
	if err != nil {
		return nil, err
	}
	x := &evalExecuteClient{stream}
	return x, nil
}

type Eval_ExecuteClient interface {
	Send(*Test) error
	Recv() (*TestResponse, error)
	grpc.ClientStream
}

type evalExecuteClient struct {
	grpc.ClientStream
}

func (x *evalExecuteClient) Send(m *Test) error {
	return x.ClientStream.SendMsg(m)
}

func (x *evalExecuteClient) Recv() (*TestResponse, error) {
	m := new(TestResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// EvalServer is the server API for Eval service.
// All implementations must embed UnimplementedEvalServer
// for forward compatibility
type EvalServer interface {
	// Compile compiles a program, to be used for later execution
	Compile(context.Context, *CompileRequest) (*CompileResponse, error)
	// Execute runs a stream of tests, returning their output
	// warning: the executable for the ID will be deleted after this is finished
	Execute(Eval_ExecuteServer) error
	mustEmbedUnimplementedEvalServer()
}

// UnimplementedEvalServer must be embedded to have forward compatible implementations.
type UnimplementedEvalServer struct {
}

func (UnimplementedEvalServer) Compile(context.Context, *CompileRequest) (*CompileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Compile not implemented")
}
func (UnimplementedEvalServer) Execute(Eval_ExecuteServer) error {
	return status.Errorf(codes.Unimplemented, "method Execute not implemented")
}
func (UnimplementedEvalServer) mustEmbedUnimplementedEvalServer() {}

// UnsafeEvalServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EvalServer will
// result in compilation errors.
type UnsafeEvalServer interface {
	mustEmbedUnimplementedEvalServer()
}

func RegisterEvalServer(s grpc.ServiceRegistrar, srv EvalServer) {
	s.RegisterService(&_Eval_serviceDesc, srv)
}

func _Eval_Compile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CompileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EvalServer).Compile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/eval.Eval/Compile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EvalServer).Compile(ctx, req.(*CompileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Eval_Execute_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(EvalServer).Execute(&evalExecuteServer{stream})
}

type Eval_ExecuteServer interface {
	Send(*TestResponse) error
	Recv() (*Test, error)
	grpc.ServerStream
}

type evalExecuteServer struct {
	grpc.ServerStream
}

func (x *evalExecuteServer) Send(m *TestResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *evalExecuteServer) Recv() (*Test, error) {
	m := new(Test)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _Eval_serviceDesc = grpc.ServiceDesc{
	ServiceName: "eval.Eval",
	HandlerType: (*EvalServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Compile",
			Handler:    _Eval_Compile_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Execute",
			Handler:       _Eval_Execute_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "eval.proto",
}
