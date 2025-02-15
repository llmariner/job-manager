package main

import (
	"context"
	"crypto/tls"
	"log"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/llmariner/cluster-manager/pkg/status"
	fv1 "github.com/llmariner/file-manager/api/v1"
	v1 "github.com/llmariner/job-manager/api/v1"
	"github.com/llmariner/job-manager/dispatcher/internal/clusterstatus"
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	"github.com/llmariner/job-manager/dispatcher/internal/dispatcher"
	"github.com/llmariner/job-manager/dispatcher/internal/s3"
	mv1 "github.com/llmariner/model-manager/api/v1"
	"github.com/llmariner/rbac-manager/pkg/auth"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
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
	ctx = ctrl.LoggerInto(ctx, log)
	ctrl.SetLogger(logger)

	if err := auth.ValidateClusterRegistrationKey(); err != nil {
		return err
	}

	restConfig, err := newRestConfig(log, c.Debug.KubeconfigPath)
	if err != nil {
		return err
	}
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		LeaderElection:   c.KubernetesManager.EnableLeaderElection,
		LeaderElectionID: c.KubernetesManager.LeaderElectionID,
		Metrics: metricsserver.Options{
			BindAddress: c.KubernetesManager.MetricsBindAddress,
		},
		HealthProbeBindAddress: c.KubernetesManager.HealthBindAddress,
		PprofBindAddress:       c.KubernetesManager.PprofBindAddress,
	})
	if err != nil {
		return err
	}

	jc := dispatcher.NewJobClient(
		mgr.GetClient(),
		c.Job,
		c.KueueIntegration,
	)

	option := grpcOption(c)

	if c.ComponentStatusSender.Enable {
		ss, err := status.NewBeaconSender(c.ComponentStatusSender, grpcOption(c), logger)
		if err != nil {
			return err
		}
		go func() {
			ss.Run(logr.NewContext(ctx, logger))
		}()
	}

	fconn, err := grpc.NewClient(c.FileManagerServerWorkerServiceAddr, option)
	if err != nil {
		return err
	}
	fclient := fv1.NewFilesWorkerServiceClient(fconn)

	mconn, err := grpc.NewClient(c.ModelManagerServerWorkerServiceAddr, option)
	if err != nil {
		return err
	}
	mclient := mv1.NewModelsWorkerServiceClient(mconn)
	s3Client, err := s3.NewClient(ctx, c.ObjectStore.S3)
	if err != nil {
		return err
	}

	jconn, err := grpc.NewClient(c.JobManagerServerWorkerServiceAddr, option)
	if err != nil {
		return err
	}
	ftClient := v1.NewFineTuningWorkerServiceClient(jconn)
	wsClient := v1.NewWorkspaceWorkerServiceClient(jconn)
	bwClient := v1.NewBatchWorkerServiceClient(jconn)

	nbm := dispatcher.NewNotebookManager(mgr.GetClient(), wsClient, c.Notebook)
	if err := nbm.SetupWithManager(mgr); err != nil {
		return err
	}

	bjm := dispatcher.NewBatchJobManager(dispatcher.BatchJobManagerOptions{
		K8sClient:   mgr.GetClient(),
		S3Client:    s3Client,
		FileClient:  fclient,
		BwClient:    bwClient,
		LlmaBaseURL: c.Notebook.LLMarinerBaseURL,
		WandbConfig: c.Job.WandbAPIKeySecret,
		KueueConfig: c.KueueIntegration,
	})
	if err := bjm.SetupWithManager(mgr); err != nil {
		return err
	}

	csm := clusterstatus.NewManager(v1.NewJobWorkerServiceClient(jconn), c.ClusterStatusUpdateInterval)
	if err := csm.SetupWithManager(mgr); err != nil {
		return err
	}

	var preProcessor dispatcher.PreProcessorI
	var postProcessor dispatcher.PostProcessorI
	if c.Debug.Standalone {
		preProcessor = &dispatcher.NoopPreProcessor{}
		postProcessor = &dispatcher.NoopPostProcessor{}
	} else {
		preProcessor = dispatcher.NewPreProcessor(fclient, mclient, s3Client)
		postProcessor = dispatcher.NewPostProcessor(mclient)
	}
	if err := dispatcher.New(ftClient, wsClient, bwClient, jc, preProcessor, nbm, bjm, c.PollingInterval).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := dispatcher.NewLifecycleManager(ftClient, mgr.GetClient(), postProcessor).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return err
	}

	return mgr.Start(ctx)
}

func newRestConfig(log logr.Logger, kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		log.Info("Using kubeconfig at", "path", kubeconfigPath)
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}

func grpcOption(c *config.Config) grpc.DialOption {
	if c.Worker.TLS.Enable {
		return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	}
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}
