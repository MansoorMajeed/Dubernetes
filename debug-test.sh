#!/bin/bash

# Debug script to see what's happening with container creation

set -e

echo "🔍 Debug Test - Container Creation"
echo "=================================="

# Configuration
ORCHESTRATOR_PORT="8080"
NGINX_PORT="8081"
TEST_POD_NAME="debug-test-app"

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
    rm -f /tmp/dubernetes-debug.db
    
    echo "✅ Cleanup complete"
}

# Set up cleanup trap
trap cleanup EXIT

echo "📋 Step 1: Start Orchestrator with Verbose Logging"
echo "------------------------------------------------"

# Create config for testing
cat > /tmp/dubernetes-debug.yaml <<EOF
api:
  host: "localhost"
  port: $ORCHESTRATOR_PORT
database:
  path: "/tmp/dubernetes-debug.db"
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

# Start orchestrator in background with verbose output
echo "  - Starting orchestrator with verbose output..."
./bin/orchestrator --config /tmp/dubernetes-debug.yaml --verbose &
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
cat > /tmp/debug-pod.yaml <<EOF
name: $TEST_POD_NAME
image: nginx:alpine
replicas: 1
access:
  host: debug-test.local
EOF

echo "  - Applying test pod configuration..."
./bin/dubectl --server http://localhost:$ORCHESTRATOR_PORT apply -f /tmp/debug-pod.yaml

echo "  - Waiting for reconciliation (10 seconds)..."
sleep 10

echo "📋 Step 3: Check Database State"
echo "------------------------------"

echo "  - Checking database for pod and replicas..."
sqlite3 /tmp/dubernetes-debug.db "SELECT name, image, replicas, desired_state FROM pods;"
echo "  - Replica entries:"
sqlite3 /tmp/dubernetes-debug.db "SELECT pod_name, replica_id, container_id, port, status FROM replicas;"

echo "📋 Step 4: Check Running Containers"
echo "----------------------------------"

echo "  - All containers with dubernetes labels:"
docker ps -a --filter "label=dubernetes.pod" --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Labels}}"

echo "  - All containers with our specific pod name:"
docker ps -a --filter "label=dubernetes.pod=$TEST_POD_NAME" --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Labels}}"

echo "📋 Step 5: Check Logs"
echo "--------------------"

echo "  - Recent docker commands (last 10):"
history | grep docker | tail -10 || echo "No docker commands in history"

echo "📋 Step 6: Manual Docker Test"
echo "----------------------------"

echo "  - Testing if we can create a container manually..."
MANUAL_CONTAINER=$(docker run -d --label="dubernetes.pod=manual-test" nginx:alpine)
echo "  - Created container: $MANUAL_CONTAINER"

echo "  - Verifying manual container is running..."
docker ps --filter "id=$MANUAL_CONTAINER"

echo "  - Cleaning up manual container..."
docker rm -f $MANUAL_CONTAINER

echo ""
echo "🔍 Debug Test Complete!"
echo "======================="
echo "Check the output above for clues about why containers aren't being created."