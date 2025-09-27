package main

import (
	"fmt"
	"text/tabwriter"
	"os"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Display one or many resources",
	Long: `Display one or many resources.

Prints a table of the most important information about the specified resources.`,
}

var getPodsCmd = &cobra.Command{
	Use:   "pods [POD_NAME]",
	Short: "List pods",
	Long: `List all pods in the current cluster, or get details about a specific pod.

This command shows running pods and their current status.`,
	Example: `  # List all pods
  dubectl get pods

  # Get details about a specific pod
  dubectl get pods my-app`,
	RunE: runGetPods,
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getPodsCmd)
}

func runGetPods(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	verbose, _ := cmd.Flags().GetBool("verbose")

	client := NewAPIClient(serverURL)

	if len(args) == 0 {
		// List all pods
		if verbose {
			fmt.Println("Fetching pod list...")
		}

		pods, err := client.ListPods()
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		if len(pods) == 0 {
			fmt.Println("No pods found.")
			return nil
		}

		// Print pods in table format
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tIMAGE\tREPLICAS\tSTATUS")

		for _, pod := range pods {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", 
				pod.Name, pod.Image, pod.Replicas, pod.Status)
		}

		w.Flush()
		return nil

	} else if len(args) == 1 {
		// Get specific pod
		podName := args[0]

		if verbose {
			fmt.Printf("Fetching details for pod: %s\n", podName)
		}

		pod, err := client.GetPod(podName)
		if err != nil {
			return fmt.Errorf("failed to get pod %s: %w", podName, err)
		}

		// Print detailed pod information
		fmt.Printf("Name:       %s\n", pod.Name)
		fmt.Printf("Image:      %s\n", pod.Image)
		fmt.Printf("Replicas:   %d\n", pod.Replicas)
		fmt.Printf("Status:     %s\n", pod.Status)
		fmt.Printf("Created:    %s\n", pod.CreatedAt)

		if pod.Access != nil && pod.Access.Host != "" {
			fmt.Printf("Host:       %s\n", pod.Access.Host)
		}

		return nil

	} else {
		return fmt.Errorf("too many arguments: expected 0 or 1, got %d", len(args))
	}
}