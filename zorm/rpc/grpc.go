package rpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

type GrpcServer struct {
	listen   net.Listener
	g        *grpc.Server
	register []func(g *grpc.Server)
	ops      []grpc.ServerOption
}

func NewGrpcServer(addr string, ops ...GrpcOption) (*GrpcServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	grpcServer := &GrpcServer{}
	grpcServer.listen = listener
	for _, v := range ops {
		v.Apply(grpcServer)
	}
	server := grpc.NewServer(grpcServer.ops...)
	grpcServer.g = server
	return grpcServer, nil
}

func (s *GrpcServer) Run() error {
	for _, f := range s.register {
		f(s.g)
	}
	return s.g.Serve(s.listen)
}

func (s *GrpcServer) Stop() {
	s.g.Stop()
}

func (s *GrpcServer) Register(f func(g *grpc.Server)) {
	s.register = append(s.register, f)
}

type GrpcOption interface {
	Apply(s *GrpcServer)
}

type DefaultGrpcOption struct {
	f func(s *GrpcServer)
}

func (d *DefaultGrpcOption) Apply(s *GrpcServer) {
	d.f(s)
}

func WithGrpcOptions(ops ...grpc.ServerOption) GrpcOption {
	return &DefaultGrpcOption{
		f: func(s *GrpcServer) {
			s.ops = append(s.ops, ops...)
		},
	}
}

type GrpcClient struct {
	Conn *grpc.ClientConn
}

type GrpcClientConfig struct {
	Address     string
	Block       bool
	DialTimeout time.Duration
	ReadTimeout time.Duration
	Direct      bool
	KeepAlive   *keepalive.ClientParameters
	dialOptions []grpc.DialOption
}

func NewGrpcClient(config *GrpcClientConfig) (*GrpcClient, error) {
	var ctx = context.Background()
	var dialOptions = config.dialOptions

	if config.Block {
		if config.DialTimeout > time.Duration(0) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.DialTimeout)
			defer cancel()
		}
		dialOptions = append(dialOptions, grpc.WithBlock())
	}
	if config.KeepAlive != nil {
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(*config.KeepAlive))
	}
	conn, err := grpc.DialContext(ctx, config.Address, dialOptions...)
	if err != nil {
		return nil, err
	}
	return &GrpcClient{
		Conn: conn,
	}, nil
}

func DefaultGrpcClientConfig() *GrpcClientConfig {
	return &GrpcClientConfig{
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
		DialTimeout: time.Second * 3,
		ReadTimeout: time.Second * 3,
		Block:       true,
	}
}
