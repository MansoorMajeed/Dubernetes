package nginx

import (
	"testing"

	"github.com/mansoormajeed/dubernetes/pkg/database"
)

func TestGenerateNginxConfig(t *testing.T) {
	tests := []struct {
		name     string
		pods     []PodWithReplicas
		expected string
		wantErr  bool
	}{
		{
			name: "single pod with access",
			pods: []PodWithReplicas{
				{
					Pod: database.Pod{
						Name:     "test-app",
						Image:    "nginx:latest",
						Replicas: 1,
						Host:     "app.local",
					},
					Replicas: []database.Replica{
						{PodName: "test-app", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "running"},
					},
				},
			},
			expected: `events {
    worker_connections 1024;
}

http {
    upstream test-app {
        server 172.17.0.2:80;
    }

    server {
        listen 80;
        server_name app.local;
        
        location / {
            proxy_pass http://test-app;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
`,
			wantErr: false,
		},
		{
			name: "multiple replicas with load balancing",
			pods: []PodWithReplicas{
				{
					Pod: database.Pod{
						Name:     "multi-app",
						Image:    "nginx:latest",
						Replicas: 3,
						Host:     "multi.local",
					},
					Replicas: []database.Replica{
						{PodName: "multi-app", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "running"},
						{PodName: "multi-app", ContainerID: "container2", IPAddress: "172.17.0.3", Port: 32002, Status: "running"},
						{PodName: "multi-app", ContainerID: "container3", IPAddress: "172.17.0.4", Port: 32003, Status: "running"},
					},
				},
			},
			expected: `events {
    worker_connections 1024;
}

http {
    upstream multi-app {
        server 172.17.0.2:80;
        server 172.17.0.3:80;
        server 172.17.0.4:80;
    }

    server {
        listen 80;
        server_name multi.local;
        
        location / {
            proxy_pass http://multi-app;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
`,
			wantErr: false,
		},
		{
			name: "multiple pods with different hosts",
			pods: []PodWithReplicas{
				{
					Pod: database.Pod{
						Name:     "app1",
						Image:    "nginx:latest",
						Replicas: 1,
						Host:     "app1.local",
					},
					Replicas: []database.Replica{
						{PodName: "app1", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "running"},
					},
				},
				{
					Pod: database.Pod{
						Name:     "app2",
						Image:    "nginx:latest",
						Replicas: 2,
						Host:     "app2.local",
					},
					Replicas: []database.Replica{
						{PodName: "app2", ContainerID: "container2", IPAddress: "172.17.0.3", Port: 32002, Status: "running"},
						{PodName: "app2", ContainerID: "container3", IPAddress: "172.17.0.4", Port: 32003, Status: "running"},
					},
				},
			},
			expected: `events {
    worker_connections 1024;
}

http {
    upstream app1 {
        server 172.17.0.2:80;
    }

    upstream app2 {
        server 172.17.0.3:80;
        server 172.17.0.4:80;
    }

    server {
        listen 80;
        server_name app1.local;
        
        location / {
            proxy_pass http://app1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }

    server {
        listen 80;
        server_name app2.local;
        
        location / {
            proxy_pass http://app2;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
`,
			wantErr: false,
		},
		{
			name:     "empty pods list",
			pods:     []PodWithReplicas{},
			expected: `events {
    worker_connections 1024;
}

http {
    server {
        listen 80 default_server;
        location / {
            return 404 "No services available";
        }
    }
}
`,
			wantErr:  false,
		},
		{
			name: "pod without host (no ingress)",
			pods: []PodWithReplicas{
				{
					Pod: database.Pod{
						Name:     "no-ingress",
						Image:    "nginx:latest",
						Replicas: 1,
						Host:     "",
					},
					Replicas: []database.Replica{
						{PodName: "no-ingress", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "running"},
					},
				},
			},
			expected: `events {
    worker_connections 1024;
}

http {

}
`,
			wantErr:  false,
		},
		{
			name: "pod with only failed replicas",
			pods: []PodWithReplicas{
				{
					Pod: database.Pod{
						Name:     "failed-app",
						Image:    "nginx:latest",
						Replicas: 1,
						Host:     "failed.local",
					},
					Replicas: []database.Replica{
						{PodName: "failed-app", ContainerID: "container1", IPAddress: "172.17.0.2", Port: 32001, Status: "failed"},
					},
				},
			},
			expected: `events {
    worker_connections 1024;
}

http {

}
`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateNginxConfig(tt.pods)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateNginxConfig() expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("GenerateNginxConfig() unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("GenerateNginxConfig() =\n%s\n\nwant:\n%s", result, tt.expected)
			}
		})
	}
}

func TestValidateNginxConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid config",
			config: `upstream app {
    server 172.17.0.2:80;
}

server {
    listen 80;
    server_name app.local;
    
    location / {
        proxy_pass http://app;
    }
}`,
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  "",
			wantErr: false,
		},
		{
			name: "invalid syntax",
			config: `upstream app {
    server 172.17.0.2:80
}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNginxConfig(tt.config)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateNginxConfig() expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ValidateNginxConfig() unexpected error: %v", err)
			}
		})
	}
}