package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-logr/stdr"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/llmariner/api-usage/pkg/sender"
	"github.com/llmariner/common/pkg/aws"
	"github.com/llmariner/common/pkg/db"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/server/internal/cache"
	"github.com/llmariner/job-manager/server/internal/config"
	"github.com/llmariner/job-manager/server/internal/k8s"
	"github.com/llmariner/job-manager/server/internal/scheduler"
	"github.com/llmariner/job-manager/server/internal/server"
	"github.com/llmariner/job-manager/server/internal/store"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

func runCmd() *cobra.Command {
	var path string
	var logLevel int
	cmd := &cobra.Command{
		Use:   "run",
		Short: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Parse(path)
			if err != nil {
				return err
			}
			if err := c.Validate(); err != nil {
				return err
			}
			stdr.SetVerbosity(logLevel)
			if err := run(cmd.Context(), &c); err != nil {
				return err
			}
			return nil
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

	dbInst, err := db.OpenDB(c.Database)
	if err != nil {
		return err
	}

	st := store.New(dbInst)
	if err := st.AutoMigrate(); err != nil {
		return err
	}

	// Initialize KMS and data key when KMS is enabled
	var dataKey []byte
	if c.KMSConfig.Enable {
		log.Info("KMS encryption is enabled")
		awsConfig := aws.NewConfigOptions{
			Region: c.KMSConfig.Region,
		}
		if ar := c.KMSConfig.AssumeRole; ar != nil {
			awsConfig.AssumeRole = &aws.AssumeRole{
				RoleARN:    ar.RoleARN,
				ExternalID: ar.ExternalID,
			}
		}
		kmsClient, err := aws.NewKMSClient(ctx, awsConfig, c.KMSConfig.KeyAlias)
		if err != nil {
			return err
		}

		dk, err := st.GetDataKey(ctx, kmsClient)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			log.Info("Creating a data key")
			dk, err = st.CreateDataKey(ctx, kmsClient)
			if err != nil {
				return err
			}
		}
		dataKey = dk
	}

	addr := fmt.Sprintf("localhost:%d", c.GRPCPort)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return err
	}
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			// Do not use the camel case for JSON fields to follow OpenAI API.
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:     true,
				EmitDefaultValues: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
		runtime.WithIncomingHeaderMatcher(auth.HeaderMatcher),
		runtime.WithHealthzEndpoint(grpc_health_v1.NewHealthClient(conn)),
	)
	if err := v1.RegisterFineTuningServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		return err
	}
	if err := v1.RegisterWorkspaceServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		return err
	}
	if err := v1.RegisterBatchServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		return err
	}
	if err := v1.RegisterJobServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		log.Info("Starting HTTP server...", "port", c.HTTPPort)
		errCh <- http.ListenAndServe(fmt.Sprintf(":%d", c.HTTPPort), mux)
	}()

	conn, err = grpc.NewClient(c.FileManagerServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	fclient := fv1.NewFilesServiceClient(conn)

	conn, err = grpc.NewClient(c.ModelManagerServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	mclient := mv1.NewModelsServiceClient(conn)

	k8sClientFactory := k8s.NewClientFactory(c.SessionManagerServerEndpoint)

	var usageSetter sender.UsageSetter
	if c.UsageSender.Enable {
		usage, err := sender.New(ctx, c.UsageSender, grpc.WithTransportCredentials(insecure.NewCredentials()), logger)
		if err != nil {
			return err
		}
		go func() { usage.Run(ctx) }()
		usageSetter = usage
	} else {
		usageSetter = sender.NoopUsageSetter{}
	}

	// TODO(aya): stop the lazy loading and populate cache data from DB at the startup time.
	cache := cache.NewStore(st, logger.WithName("cache"))
	sched := scheduler.New(cache, logger.WithName("scheduler"))

	srv := server.New(
		st,
		fclient,
		mclient,
		k8sClientFactory,
		sched,
		cache,
		c.NotebookConfig.ImageTypes,
		c.BatchJobConfig.Images,
		logger,
		dataKey,
	)
	go func() {
		errCh <- srv.Run(ctx, c.GRPCPort, c.AuthConfig, usageSetter)
	}()

	go func() {
		// TODO(guangrui): Make the interval and maxQueuedTime configurable.
		const (
			defaultMaxQueuedTime       = 3 * time.Minute
			defaultReschedulerInterval = 1 * time.Minute
		)
		errCh <- srv.RunRescheduler(ctx, defaultReschedulerInterval, defaultMaxQueuedTime)
	}()

	go func() {
		s := server.NewWorkerServiceServer(st, cache, logger)
		errCh <- s.Run(ctx, c.WorkerServiceGRPCPort, c.AuthConfig)
	}()

	go func() {
		s := server.NewSyncerServiceServer(logger, k8sClientFactory, sched, cache)
		errCh <- s.Run(ctx, c.SyncerServiceGRPCPort, c.AuthConfig)
	}()

	return <-errCh
}
