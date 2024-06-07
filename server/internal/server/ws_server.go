package server

import (
	"context"
	"fmt"
	"log"
	"net"

	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// NewWorkerServiceServer creates a new worker service server.
func NewWorkerServiceServer(s *store.S) *WS {
	return &WS{
		store: s,
	}
}

// WS is a server for worker services.
type WS struct {
	v1.UnimplementedFineTuningWorkerServiceServer
	v1.UnimplementedWorkspaceWorkerServiceServer

	srv   *grpc.Server
	store *store.S
}

// Run runs the worker service server.
func (ws *WS) Run(ctx context.Context, port int) error {
	log.Printf("Starting worker service server on port %d", port)

	// TODO(aya): configure request authN/Z

	srv := grpc.NewServer()
	v1.RegisterFineTuningWorkerServiceServer(srv, ws)
	v1.RegisterWorkspaceWorkerServiceServer(srv, ws)
	reflection.Register(srv)

	ws.srv = srv

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := srv.Serve(l); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Stop stops the worker service server.
func (ws *WS) Stop() {
	ws.srv.Stop()
}
