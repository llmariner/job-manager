package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{Use: "syncer"}
	cmd.AddCommand(runCmd())
	cmd.SilenceUsage = true

	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
