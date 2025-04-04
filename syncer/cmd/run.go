package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"

	"github.com/go-logr/stdr"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/syncer/internal/config"
	"github.com/llmariner/job-manager/syncer/internal/controller"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func runCmd() *cobra.Command {
	var path string
	var logLevel int
	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Parse(path)
			if err != nil {
				return err
			}
			if err := c.Validate(); err != nil {
				return err
			}
			stdr.SetVerbosity(logLevel)
			return run(cmd.Context(), &c)
		},
	}
	cmd.Flags().StringVar(&path, "config", "", "Path to the config file")
	cmd.Flags().IntVar(&logLevel, "v", 0, "Log level")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

func run(ctx context.Context, c *config.Config) error {
	logger := stdr.New(log.Default())
	log := logger.WithName("boot")
	ctx = ctrl.LoggerInto(ctx, log)
	ctrl.SetLogger(logger)

	conn, err := grpc.NewClient(c.JobManagerServerSyncerServiceAddr, grpcOption(c))
	if err != nil {
		return fmt.Errorf("failed to create job grpc client: %s", err)
	}
	ssc := v1.NewSyncerServiceClient(conn)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           controller.Scheme,
		LeaderElection:   c.KubernetesManager.EnableLeaderElection,
		LeaderElectionID: c.KubernetesManager.LeaderElectionID,
		Metrics: metricsserver.Options{
			BindAddress: c.KubernetesManager.MetricsBindAddress,
		},
		HealthProbeBindAddress: c.KubernetesManager.HealthBindAddress,
		PprofBindAddress:       c.KubernetesManager.PprofBindAddress,
	})
	if err != nil {
		return fmt.Errorf("create manager: %s", err)
	}

	if c.SyncedKinds.Jobs {
		if err := (&controller.JobController{}).SetupWithManager(mgr, ssc); err != nil {
			return fmt.Errorf("setup job controller: %s", err)
		}
	}

	if c.SyncedKinds.JobSets {
		if err := (&controller.JobSetController{}).SetupWithManager(mgr, ssc); err != nil {
			return fmt.Errorf("setup job-set controller: %s", err)
		}
	}

	if err := (&controller.RemoteSyncerManager{}).SetupWithManager(mgr, ssc, c.SessionManagerEndpoint, c.SyncedKinds); err != nil {
		return fmt.Errorf("setup remote syncer manager: %s", err)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return err
	}

	return mgr.Start(ctx)
}

func grpcOption(c *config.Config) grpc.DialOption {
	if c.Tenant.TLS.Enable {
		return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	}
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}
