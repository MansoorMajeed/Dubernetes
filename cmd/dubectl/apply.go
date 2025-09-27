package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a configuration to a resource",
	Long: `Apply a configuration to a resource by filename.

This command creates or updates resources based on the configuration
defined in YAML files.`,
	Example: `  # Apply a pod configuration
  dubectl apply -f pod.yaml

  # Apply multiple configurations
  dubectl apply -f pod1.yaml -f pod2.yaml`,
	RunE: runApply,
}

var applyFlags struct {
	filenames []string
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringSliceVarP(&applyFlags.filenames, "filename", "f", []string{}, "Filename or directory to apply (can be specified multiple times)")
	applyCmd.MarkFlagRequired("filename")
}

func runApply(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	verbose, _ := cmd.Flags().GetBool("verbose")

	client := NewAPIClient(serverURL)

	for _, filename := range applyFlags.filenames {
		if verbose {
			fmt.Printf("Applying configuration from %s...\n", filename)
		}

		// Parse YAML file
		podSpec, err := parseYAMLFile(filename)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", filename, err)
		}

		if verbose {
			fmt.Printf("Parsed pod: %s (image: %s, replicas: %d)\n", 
				podSpec.Name, podSpec.Image, podSpec.Replicas)
		}

		// Apply the configuration
		if err := client.CreatePod(podSpec); err != nil {
			return fmt.Errorf("Error applying pod: %w", err)
		}

		fmt.Printf("pod/%s created\n", podSpec.Name)
	}

	return nil
}