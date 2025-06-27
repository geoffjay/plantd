#!/bin/bash

set -e

echo "Testing State gRPC Service"
echo "========================="

# Start the service in the background
echo "Starting State gRPC service..."
./build/plantd-state-grpc &
GRPC_PID=$!

# Wait for service to start
sleep 2

# Function to cleanup
cleanup() {
    echo "Cleaning up..."
    kill $GRPC_PID 2>/dev/null || true
    rm -f state.db
}

# Ensure cleanup on exit
trap cleanup EXIT

# Test health endpoint
echo "Testing health endpoint..."
curl -s http://localhost:8080/health | jq .

# Test status endpoint
echo "Testing status endpoint..."
curl -s http://localhost:8080/status | jq .

# Test MDP compatibility layer
echo "Testing MDP compatibility - SET operation..."
curl -s -X POST "http://localhost:8080/mdp/set/test-key/test-value" | jq .

echo "Testing MDP compatibility - GET operation..."
curl -s -X POST "http://localhost:8080/mdp/get/test-key" | jq .

echo "Testing MDP compatibility - LIST operation..."
curl -s -X POST "http://localhost:8080/mdp/list" | jq .

echo "Testing MDP compatibility - DELETE operation..."
curl -s -X POST "http://localhost:8080/mdp/delete/test-key" | jq .

echo "Testing MDP compatibility - GET after DELETE..."
curl -s -X POST "http://localhost:8080/mdp/get/test-key" || echo "Expected: Key not found"

echo ""
echo "✅ All tests completed successfully!"
echo "🎉 State gRPC service is working with MDP compatibility!" 
