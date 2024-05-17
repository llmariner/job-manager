package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/llm-operator/common/pkg/db"
	fv1 "github.com/llm-operator/file-manager/api/v1"
	v1 "github.com/llm-operator/job-manager/api/v1"
	"github.com/llm-operator/job-manager/common/pkg/store"
	"github.com/llm-operator/job-manager/server/internal/config"
	"github.com/llm-operator/job-manager/server/internal/server"
	mv1 "github.com/llm-operator/model-manager/api/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
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
	dbInst, err := db.OpenDB(c.Database)
	if err != nil {
		return err
	}

	st := store.New(dbInst)
	if err := st.AutoMigrate(); err != nil {
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
	)
	addr := fmt.Sprintf("localhost:%d", c.GRPCPort)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := v1.RegisterFineTuningServiceHandlerFromEndpoint(ctx, mux, addr, opts); err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		log.Printf("Starting HTTP server on port %d", c.HTTPPort)
		errCh <- http.ListenAndServe(fmt.Sprintf(":%d", c.HTTPPort), mux)
	}()

	conn, err := grpc.Dial(c.FileManagerServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	fclient := fv1.NewFilesServiceClient(conn)

	conn, err = grpc.Dial(c.ModelManagerServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	mclient := mv1.NewModelsServiceClient(conn)

	restConfig, err := newRestConfig(c.Debug.KubeconfigPath)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	k8sJobClient := server.NewK8sJobClient(kubeClient, c.JobNamespace)

	go func() {
		s := server.New(st, fclient, mclient, k8sJobClient)
		errCh <- s.Run(ctx, c.GRPCPort, c.AuthConfig)
	}()

	return <-errCh
}

func newRestConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		log.Printf("Using kubeconfig at %s", kubeconfigPath)
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	return rest.InClusterConfig()
}

func init() {
	runCmd.Flags().StringP(flagConfig, "c", "", "Configuration file path")
	_ = runCmd.MarkFlagRequired(flagConfig)
}
