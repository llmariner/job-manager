package main

import (
	"context"
	"fmt"
	"log"

	iv1 "github.com/llm-operator/inference-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/db"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
	"github.com/llm-operator/job-manager/dispatcher/internal/dispatcher"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const flagConfig = "config"

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

	k8sClient, err := newK8sClient(c.Debug.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("new k8s client: %s", err)
	}

	pc := dispatcher.NewPodCreator(k8sClient, c.JobNamespace, &c.ModelStore, c.Debug.UseFakeJob)

	var iclient dispatcher.ModelRegisterClient
	if c.Debug.Standalone {
		iclient = &dispatcher.NoopModelRegisterClient{}
	} else {
		conn, err := grpc.Dial(c.InferenceManagerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		iclient = iv1.NewInferenceEngineInternalServiceClient(conn)
	}

	d := dispatcher.New(st, pc, iclient)
	return d.Run(ctx, c.JobPollingInterval)
}

func newK8sClient(kubeconfigPath string) (kubernetes.Interface, error) {
	var config *rest.Config
	var err error
	if kubeconfigPath != "" {
		log.Printf("Using kubeconfig at %q", kubeconfigPath)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
