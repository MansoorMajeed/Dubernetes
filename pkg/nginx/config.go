package nginx

import (
	"fmt"
	"strings"

	"github.com/mansoormajeed/dubernetes/pkg/database"
)

// PodWithReplicas represents a pod with its associated replicas for nginx config generation
type PodWithReplicas struct {
	Pod      database.Pod
	Replicas []database.Replica
}

// GenerateNginxConfig generates nginx configuration from pod and replica data
func GenerateNginxConfig(pods []PodWithReplicas) (string, error) {
	// Start with basic nginx config structure
	var config strings.Builder
	config.WriteString("events {\n")
	config.WriteString("    worker_connections 1024;\n")
	config.WriteString("}\n\n")
	config.WriteString("http {\n")
	
	if len(pods) == 0 {
		// Return basic config with default server
		config.WriteString("    server {\n")
		config.WriteString("        listen 80 default_server;\n")
		config.WriteString("        location / {\n")
		config.WriteString("            return 404 \"No services available\";\n")
		config.WriteString("        }\n")
		config.WriteString("    }\n")
		config.WriteString("}\n")
		return config.String(), nil
	}

	var upstreams []string
	var servers []string

	for _, podWithReplicas := range pods {
		pod := podWithReplicas.Pod
		replicas := podWithReplicas.Replicas

		// Skip pods without ingress configuration
		if pod.Host == "" {
			continue
		}

		// Filter running replicas only
		var runningReplicas []database.Replica
		for _, replica := range replicas {
			if replica.Status == "running" {
				runningReplicas = append(runningReplicas, replica)
			}
		}

		// Skip pods with no running replicas
		if len(runningReplicas) == 0 {
			continue
		}

		// Generate upstream block
		upstream := generateUpstreamBlock(pod.Name, runningReplicas)
		upstreams = append(upstreams, upstream)

		// Generate server block
		server := generateServerBlock(pod.Name, pod.Host)
		servers = append(servers, server)
	}

	// Add upstreams to http block
	for i, upstream := range upstreams {
		config.WriteString("    ")
		config.WriteString(strings.ReplaceAll(upstream, "\n", "\n    "))
		if i < len(upstreams)-1 {
			config.WriteString("\n\n")
		}
	}

	if len(upstreams) > 0 && len(servers) > 0 {
		config.WriteString("\n\n")
	}

	// Add servers to http block
	for i, server := range servers {
		config.WriteString("    ")
		config.WriteString(strings.ReplaceAll(server, "\n", "\n    "))
		if i < len(servers)-1 {
			config.WriteString("\n\n")
		}
	}

	// Close http block
	config.WriteString("\n}\n")

	return config.String(), nil
}

// generateUpstreamBlock creates an nginx upstream block for a pod
func generateUpstreamBlock(podName string, replicas []database.Replica) string {
	var upstream strings.Builder
	upstream.WriteString(fmt.Sprintf("upstream %s {\n", podName))

	for _, replica := range replicas {
		if replica.IPAddress != "" {
			upstream.WriteString(fmt.Sprintf("    server %s:80;\n", replica.IPAddress))
		}
	}

	upstream.WriteString("}")
	return upstream.String()
}

// generateServerBlock creates an nginx server block for a pod
func generateServerBlock(podName, host string) string {
	var server strings.Builder
	server.WriteString("server {\n")
	server.WriteString("    listen 80;\n")
	server.WriteString(fmt.Sprintf("    server_name %s;\n", host))
	server.WriteString("    \n")
	server.WriteString("    location / {\n")
	server.WriteString(fmt.Sprintf("        proxy_pass http://%s;\n", podName))
	server.WriteString("        proxy_set_header Host $host;\n")
	server.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	server.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	server.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	server.WriteString("    }\n")
	server.WriteString("}")
	return server.String()
}

// ValidateNginxConfig validates nginx configuration syntax
func ValidateNginxConfig(config string) error {
	if config == "" {
		return nil
	}

	// Basic syntax validation
	lines := strings.Split(config, "\n")
	var braceStack []string
	
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for missing semicolons on directive lines
		if !strings.Contains(line, "{") && !strings.Contains(line, "}") {
			if !strings.HasSuffix(line, ";") {
				return fmt.Errorf("line %d: missing semicolon at end of directive: %s", lineNum+1, line)
			}
		}

		// Track braces for balance checking
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		
		for i := 0; i < openBraces; i++ {
			braceStack = append(braceStack, "{")
		}
		
		for i := 0; i < closeBraces; i++ {
			if len(braceStack) == 0 {
				return fmt.Errorf("line %d: unexpected closing brace", lineNum+1)
			}
			braceStack = braceStack[:len(braceStack)-1]
		}
	}

	// Check for unbalanced braces
	if len(braceStack) > 0 {
		return fmt.Errorf("unbalanced braces: %d unclosed braces", len(braceStack))
	}

	return nil
}