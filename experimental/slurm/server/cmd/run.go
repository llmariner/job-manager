package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-logr/stdr"

	v40 "github.com/llmariner/job-manager/experimental/slurm/api/v0040"
	"github.com/llmariner/job-manager/experimental/slurm/server/internal/config"
	"github.com/llmariner/job-manager/experimental/slurm/server/internal/server"
	"github.com/spf13/cobra"
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

	log.Info("Starting the server", "port", c.HTTPPort)

	proxy := server.NewProxy(c.BaseURL, c.AuthToken)
	s := server.New(proxy, logger.WithName("server"))

	hs := &http.Server{
		Handler: v40.HandlerFromMux(s, http.NewServeMux()),
		Addr:    fmt.Sprintf("0.0.0.0:%d", c.HTTPPort),
	}
	return hs.ListenAndServe()
}
