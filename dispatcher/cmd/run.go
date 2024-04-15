package main

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
	fv1 "github.com/llm-operator/file-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/db"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	"github.com/llm-operator/job-manager/dispatcher/internal/dispatcher"
	"github.com/llm-operator/job-manager/dispatcher/internal/s3"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const flagConfig = "config"

var setupLog = ctrl.Log.WithName("setup")

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Flags().GetString(flagConfig)
		if err != nil {
			return err
		}

		c, err := config.Parse(path)
		if err != nil {
			return err
		}

		if err := c.Validate(); err != nil {
			return err
		}

		if err := run(cmd.Context(), &c); err != nil {
			return err
		}
		return nil
	},
}

func run(ctx context.Context, c *config.Config) error {
	ctrl.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))

	var st *store.S
	if c.Debug.Standalone {
		dbInst, err := gorm.Open(sqlite.Open(c.Debug.SqlitePath), &gorm.Config{})
		if err != nil {
			return err
		}
		st = store.New(dbInst)
		if err := st.AutoMigrate(); err != nil {
			return err
		}
	} else {
		dbInst, err := db.OpenDB(c.Database)
		if err != nil {
			return err
		}
		st = store.New(dbInst)
	}

	restConfig, err := newRestConfig(c.Debug.KubeconfigPath)
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
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				c.JobNamespace: cache.Config{},
			},
		},
	})
	if err != nil {
		return err
	}

	jc := dispatcher.NewJobClient(
		mgr.GetClient(),
		c.JobNamespace,
		&c.ModelStore,
		c.Debug.UseFakeJob,
		c.Debug.HuggingFaceAccessToken,
	)

	var preProcessor dispatcher.PostProcessorI
	var postProcessor dispatcher.PostProcessorI
	if c.Debug.Standalone {
		preProcessor = &dispatcher.NoopPreProcessor{}
		postProcessor = &dispatcher.NoopPostProcessor{}
	} else {
		conn, err := grpc.Dial(c.FileManagerInternalServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		fclient := fv1.NewFilesInternalServiceClient(conn)

		conn, err = grpc.Dial(c.ModelManagerInternalServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		mclient := mv1.NewModelsInternalServiceClient(conn)
		s3Client := s3.NewClient(c.ObjectStore.S3)

		preProcessor = dispatcher.NewPreProcessor(fclient, mclient)
		postProcessor = dispatcher.NewPostProcessor(mclient, s3Client)
	}

	if err := dispatcher.New(st, jc, preProcessor, c.JobPollingInterval).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := dispatcher.NewLifecycleManager(st, mgr.GetClient(), postProcessor).
		SetupWithManager(mgr); err != nil {
		return err
	}
	return mgr.Start(ctx)
}

func newRestConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		setupLog.Info("Using kubeconfig at", "path", kubeconfigPath)
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
