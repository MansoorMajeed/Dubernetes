package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		want     *Config
		wantErr  bool
	}{
		{
			name: "valid config file",
			yamlData: `
orchestrator:
  port: 8080
  host: "localhost"
proxy:
  port: 80
  host: "localhost"
  config_path: "/tmp/test-nginx.conf"
containers:
  port_range_start: 32000
  port_range_end: 32999
database:
  path: "./test.db"
reconciler:
  interval: "10s"
  max_restart_backoff: "5m"
`,
			want: &Config{
				Orchestrator: OrchestratorConfig{
					Port: 8080,
					Host: "localhost",
				},
				Proxy: ProxyConfig{
					Port:       80,
					Host:       "localhost",
					ConfigPath: "/tmp/test-nginx.conf",
				},
				Containers: ContainerConfig{
					PortRangeStart: 32000,
					PortRangeEnd:   32999,
				},
				Database: DatabaseConfig{
					Path: "./test.db",
				},
				Reconciler: ReconcilerConfig{
					Interval:          10 * time.Second,
					MaxRestartBackoff: 5 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid yaml",
			yamlData: "invalid: yaml: content: [",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.yamlData), 0644)
			if err != nil {
				t.Fatalf("Failed to write test config file: %v", err)
			}

			got, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !configEqual(got, tt.want) {
				t.Errorf("LoadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Test loading config with missing file - should use defaults
	got, err := LoadConfig("nonexistent.yaml")
	if err != nil {
		t.Errorf("LoadConfig() with nonexistent file should not error, got: %v", err)
	}

	// Check that defaults are applied
	if got.Orchestrator.Port != 8080 {
		t.Errorf("Expected default orchestrator port 8080, got %d", got.Orchestrator.Port)
	}
	if got.Proxy.Port != 80 {
		t.Errorf("Expected default proxy port 80, got %d", got.Proxy.Port)
	}
}

func TestEnvironmentVariableOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("DUBERNETES_ORCHESTRATOR_PORT", "9090")
	os.Setenv("DUBERNETES_PROXY_PORT", "8080")
	defer func() {
		os.Unsetenv("DUBERNETES_ORCHESTRATOR_PORT")
		os.Unsetenv("DUBERNETES_PROXY_PORT")
	}()

	config, err := LoadConfig("nonexistent.yaml")
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	if config.Orchestrator.Port != 9090 {
		t.Errorf("Expected orchestrator port 9090 from env var, got %d", config.Orchestrator.Port)
	}
	if config.Proxy.Port != 8080 {
		t.Errorf("Expected proxy port 8080 from env var, got %d", config.Proxy.Port)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Orchestrator: OrchestratorConfig{Port: 8080, Host: "localhost"},
				Proxy:        ProxyConfig{Port: 80, Host: "localhost"},
				Containers:   ContainerConfig{PortRangeStart: 32000, PortRangeEnd: 32999},
				Database:     DatabaseConfig{Path: "./test.db"},
				Reconciler:   ReconcilerConfig{Interval: 10 * time.Second, MaxRestartBackoff: 5 * time.Minute},
			},
			wantErr: false,
		},
		{
			name: "invalid port range",
			config: &Config{
				Orchestrator: OrchestratorConfig{Port: 8080, Host: "localhost"},
				Proxy:        ProxyConfig{Port: 80, Host: "localhost"},
				Containers:   ContainerConfig{PortRangeStart: 33000, PortRangeEnd: 32000},
				Database:     DatabaseConfig{Path: "./test.db"},
				Reconciler:   ReconcilerConfig{Interval: 10 * time.Second, MaxRestartBackoff: 5 * time.Minute},
			},
			wantErr: true,
		},
		{
			name: "port out of valid range",
			config: &Config{
				Orchestrator: OrchestratorConfig{Port: -1, Host: "localhost"},
				Proxy:        ProxyConfig{Port: 80, Host: "localhost"},
				Containers:   ContainerConfig{PortRangeStart: 32000, PortRangeEnd: 32999},
				Database:     DatabaseConfig{Path: "./test.db"},
				Reconciler:   ReconcilerConfig{Interval: 10 * time.Second, MaxRestartBackoff: 5 * time.Minute},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to compare configs
func configEqual(a, b *Config) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Orchestrator.Port == b.Orchestrator.Port &&
		a.Orchestrator.Host == b.Orchestrator.Host &&
		a.Proxy.Port == b.Proxy.Port &&
		a.Proxy.Host == b.Proxy.Host &&
		a.Proxy.ConfigPath == b.Proxy.ConfigPath &&
		a.Containers.PortRangeStart == b.Containers.PortRangeStart &&
		a.Containers.PortRangeEnd == b.Containers.PortRangeEnd &&
		a.Database.Path == b.Database.Path &&
		a.Reconciler.Interval == b.Reconciler.Interval &&
		a.Reconciler.MaxRestartBackoff == b.Reconciler.MaxRestartBackoff
}