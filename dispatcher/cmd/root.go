package main

import "github.com/spf13/cobra"

// rootCmd is the root of the command-line application.
var rootCmd = &cobra.Command{
	Use:   "dispatcher",
	Short: "dispatcher",
}

func init() {
	rootCmd.AddCommand(runCmd())
	rootCmd.SilenceUsage = true
}
