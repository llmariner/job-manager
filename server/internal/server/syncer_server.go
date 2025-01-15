package server

import (
	"context"
	"fmt"
	"net"

	"github.com/go-logr/logr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewSyncerServiceServer creates a new syncer service server.
func NewSyncerServiceServer(logger logr.Logger, k8sClientFactory k8s.ClientFactory, scheduler schedulerI) *SS {
	return &SS{
		logger:           logger.WithName("syncer"),
		k8sClientFactory: k8sClientFactory,
		scheduler:        scheduler,
	}
}

// SS is a server for syncer services.
type SS struct {
	v1.UnimplementedSyncerServiceServer

	srv              *grpc.Server
	k8sClientFactory k8s.ClientFactory
	scheduler        schedulerI
	logger           logr.Logger
}

// Run runs the syncer service server.
func (ss *SS) Run(ctx context.Context, port int) error {
	ss.logger.Info("Starting syncer service server...", "port", port)

	// TODO: support auth
	fakeAuth := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(fakeAuthInto(ctx), req)
	}
	opt := grpc.ChainUnaryInterceptor(fakeAuth)

	srv := grpc.NewServer(opt)
	v1.RegisterSyncerServiceServer(srv, ss)
	reflection.Register(srv)
	ss.srv = srv

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := srv.Serve(l); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
