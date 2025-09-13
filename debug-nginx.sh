#!/bin/bash

# Debug nginx configuration generation

set -e

echo "🔍 Debug Nginx Configuration"
echo "==========================="

# Configuration
ORCHESTRATOR_PORT="8080"
NGINX_PORT="8081"
TEST_POD_NAME="nginx-debug-app"

# Clean up function
cleanup() {
    echo "🧹 Cleaning up..."
    
    # Stop orchestrator if running
    if [ ! -z "$ORCHESTRATOR_PID" ]; then
        echo "  - Stopping orchestrator (PID: $ORCHESTRATOR_PID)"
        kill $ORCHESTRATOR_PID 2>/dev/null || true
        wait $ORCHESTRATOR_PID 2>/dev/null || true
    fi
    
    # Clean up any containers
    echo "  - Cleaning up containers..."
    docker ps -a --filter "label=dubernetes.pod=$TEST_POD_NAME" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
    docker ps -a --filter "name=dubernetes-nginx" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
    
    # Clean up database
    rm -f /tmp/dubernetes-nginx-debug.db
    
    echo "✅ Cleanup complete"
}

# Set up cleanup trap
trap cleanup EXIT

echo "📋 Step 1: Start Orchestrator"
echo "----------------------------"

# Create config for testing
cat > /tmp/dubernetes-nginx-debug.yaml <<EOF
api:
  host: "localhost"
  port: $ORCHESTRATOR_PORT
database:
  path: "/tmp/dubernetes-nginx-debug.db"
nginx:
  port: $NGINX_PORT
  config_path: "/tmp/dubernetes-nginx-debug.conf"
docker:
  network: "bridge"
  port_range:
    start: 32000
    end: 32999
logging:
  level: "debug"
reconciler:
  interval: 5
EOF

# Start orchestrator in background
echo "  - Starting orchestrator..."
./bin/orchestrator --config /tmp/dubernetes-nginx-debug.yaml &
ORCHESTRATOR_PID=$!

# Wait for orchestrator to start
echo "  - Waiting for orchestrator to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:$ORCHESTRATOR_PORT/health > /dev/null 2>&1; then
        echo "  ✅ Orchestrator is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "  ❌ Orchestrator failed to start"
        exit 1
    fi
    sleep 1
done

echo "📋 Step 2: Deploy Test Pod"
echo "-------------------------"

# Create test pod YAML
cat > /tmp/nginx-debug-pod.yaml <<EOF
name: $TEST_POD_NAME
image: nginx:alpine
replicas: 1
access:
  host: nginx-debug.local
EOF

echo "  - Applying test pod configuration..."
./bin/dubectl --server http://localhost:$ORCHESTRATOR_PORT apply -f /tmp/nginx-debug-pod.yaml

echo "  - Waiting for deployment (15 seconds)..."
sleep 15

echo "📋 Step 3: Check Nginx Configuration"
echo "-----------------------------------"

echo "  - Checking if nginx config file exists..."
if [ -f "/tmp/dubernetes-nginx-debug.conf" ]; then
    echo "  ✅ Config file exists"
    echo "  - Config file contents:"
    echo "    ----------------------------------------"
    cat /tmp/dubernetes-nginx-debug.conf
    echo "    ----------------------------------------"
else
    echo "  ❌ Config file does not exist"
fi

echo "📋 Step 4: Check Nginx Container"
echo "-------------------------------"

NGINX_CONTAINER=$(docker ps --filter "name=dubernetes-nginx" --format "{{.ID}}")
if [ ! -z "$NGINX_CONTAINER" ]; then
    echo "  ✅ Nginx container found: $NGINX_CONTAINER"
    
    echo "  - Testing nginx config syntax in container..."
    docker exec $NGINX_CONTAINER nginx -t || echo "  ❌ Config test failed"
    
    echo "  - Checking nginx error logs..."
    docker logs $NGINX_CONTAINER --tail 20
else
    echo "  ❌ Nginx container not found"
fi

echo "📋 Step 5: Database Check"
echo "-----------------------"

echo "  - Checking database for pods and replicas..."
echo "  - Pods:"
sqlite3 /tmp/dubernetes-nginx-debug.db "SELECT name, image, replicas, desired_state, host FROM pods;"
echo "  - Replicas:"
sqlite3 /tmp/dubernetes-nginx-debug.db "SELECT pod_name, replica_id, container_id, port, status FROM replicas;"

echo ""
echo "🔍 Debug Complete!"
echo "=================="