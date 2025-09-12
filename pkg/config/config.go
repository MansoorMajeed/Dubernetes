package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
)

// Config represents the complete Dubernetes configuration
type Config struct {
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	Proxy        ProxyConfig        `yaml:"proxy"`
	Containers   ContainerConfig    `yaml:"containers"`
	Database     DatabaseConfig     `yaml:"database"`
	Reconciler   ReconcilerConfig   `yaml:"reconciler"`
}

// OrchestratorConfig contains settings for the orchestrator API server
type OrchestratorConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// ProxyConfig contains settings for the nginx proxy
type ProxyConfig struct {
	Port       int    `yaml:"port"`
	Host       string `yaml:"host"`
	ConfigPath string `yaml:"config_path"`
}

// ContainerConfig contains settings for container port allocation
type ContainerConfig struct {
	PortRangeStart int `yaml:"port_range_start"`
	PortRangeEnd   int `yaml:"port_range_end"`
}

// DatabaseConfig contains settings for the SQLite database
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// ReconcilerConfig contains settings for the reconciliation loop
type ReconcilerConfig struct {
	Interval          time.Duration `yaml:"interval"`
	MaxRestartBackoff time.Duration `yaml:"max_restart_backoff"`
}

// LoadConfig loads configuration from a YAML file with environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	// Start with default configuration
	config := defaultConfig()

	// Try to load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// defaultConfig returns a configuration with sensible defaults
func defaultConfig() *Config {
	return &Config{
		Orchestrator: OrchestratorConfig{
			Port: 8080,
			Host: "localhost",
		},
		Proxy: ProxyConfig{
			Port:       80,
			Host:       "localhost",
			ConfigPath: "/tmp/dubernetes-nginx.conf",
		},
		Containers: ContainerConfig{
			PortRangeStart: 32000,
			PortRangeEnd:   32999,
		},
		Database: DatabaseConfig{
			Path: "./dubernetes.db",
		},
		Reconciler: ReconcilerConfig{
			Interval:          10 * time.Second,
			MaxRestartBackoff: 5 * time.Minute,
		},
	}
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *Config) {
	// Orchestrator overrides
	if port := os.Getenv("DUBERNETES_ORCHESTRATOR_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Orchestrator.Port = p
		}
	}
	if host := os.Getenv("DUBERNETES_ORCHESTRATOR_HOST"); host != "" {
		config.Orchestrator.Host = host
	}

	// Proxy overrides
	if port := os.Getenv("DUBERNETES_PROXY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Proxy.Port = p
		}
	}
	if host := os.Getenv("DUBERNETES_PROXY_HOST"); host != "" {
		config.Proxy.Host = host
	}
	if path := os.Getenv("DUBERNETES_PROXY_CONFIG_PATH"); path != "" {
		config.Proxy.ConfigPath = path
	}

	// Container overrides
	if start := os.Getenv("DUBERNETES_CONTAINER_PORT_START"); start != "" {
		if p, err := strconv.Atoi(start); err == nil {
			config.Containers.PortRangeStart = p
		}
	}
	if end := os.Getenv("DUBERNETES_CONTAINER_PORT_END"); end != "" {
		if p, err := strconv.Atoi(end); err == nil {
			config.Containers.PortRangeEnd = p
		}
	}

	// Database overrides
	if path := os.Getenv("DUBERNETES_DATABASE_PATH"); path != "" {
		config.Database.Path = path
	}

	// Reconciler overrides
	if interval := os.Getenv("DUBERNETES_RECONCILER_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.Reconciler.Interval = d
		}
	}
	if backoff := os.Getenv("DUBERNETES_RECONCILER_MAX_BACKOFF"); backoff != "" {
		if d, err := time.ParseDuration(backoff); err == nil {
			config.Reconciler.MaxRestartBackoff = d
		}
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate orchestrator port
	if c.Orchestrator.Port < 1 || c.Orchestrator.Port > 65535 {
		return fmt.Errorf("orchestrator port must be between 1 and 65535, got %d", c.Orchestrator.Port)
	}

	// Validate proxy port
	if c.Proxy.Port < 1 || c.Proxy.Port > 65535 {
		return fmt.Errorf("proxy port must be between 1 and 65535, got %d", c.Proxy.Port)
	}

	// Validate container port range
	if c.Containers.PortRangeStart < 1024 || c.Containers.PortRangeStart > 65535 {
		return fmt.Errorf("container port range start must be between 1024 and 65535, got %d", c.Containers.PortRangeStart)
	}
	if c.Containers.PortRangeEnd < 1024 || c.Containers.PortRangeEnd > 65535 {
		return fmt.Errorf("container port range end must be between 1024 and 65535, got %d", c.Containers.PortRangeEnd)
	}
	if c.Containers.PortRangeStart >= c.Containers.PortRangeEnd {
		return fmt.Errorf("container port range start must be less than end, got start=%d end=%d", 
			c.Containers.PortRangeStart, c.Containers.PortRangeEnd)
	}

	// Validate hosts are not empty
	if c.Orchestrator.Host == "" {
		return fmt.Errorf("orchestrator host cannot be empty")
	}
	if c.Proxy.Host == "" {
		return fmt.Errorf("proxy host cannot be empty")
	}

	// Validate database path is not empty
	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	// Validate reconciler settings
	if c.Reconciler.Interval <= 0 {
		return fmt.Errorf("reconciler interval must be positive, got %v", c.Reconciler.Interval)
	}
	if c.Reconciler.MaxRestartBackoff <= 0 {
		return fmt.Errorf("reconciler max restart backoff must be positive, got %v", c.Reconciler.MaxRestartBackoff)
	}

	return nil
}