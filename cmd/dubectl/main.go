package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dubectl",
	Short: "Dubernetes CLI - A simple container orchestration client",
	Long: `dubectl is the command-line interface for Dubernetes, 
a simplified container orchestration system.

Use dubectl to deploy, manage, and monitor containerized applications
in your Dubernetes cluster.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "Dubernetes orchestrator server URL")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
}