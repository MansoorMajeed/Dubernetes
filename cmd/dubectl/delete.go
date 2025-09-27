package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Long: `Delete resources by name.

This command deletes the specified resources from the cluster.`,
}

var deletePodsCmd = &cobra.Command{
	Use:   "pods POD_NAME [POD_NAME...]",
	Short: "Delete pods",
	Long: `Delete one or more pods by name.

This command removes the specified pods from the cluster and stops all
their running containers.`,
	Example: `  # Delete a single pod
  dubectl delete pods my-app

  # Delete multiple pods
  dubectl delete pods app1 app2 app3`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDeletePods,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.AddCommand(deletePodsCmd)
}

func runDeletePods(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	verbose, _ := cmd.Flags().GetBool("verbose")

	client := NewAPIClient(serverURL)

	for _, podName := range args {
		if verbose {
			fmt.Printf("Deleting pod: %s\n", podName)
		}

		err := client.DeletePod(podName)
		if err != nil {
			return fmt.Errorf("failed to delete pod %s: %w", podName, err)
		}

		fmt.Printf("pod/%s deleted\n", podName)
	}

	return nil
}