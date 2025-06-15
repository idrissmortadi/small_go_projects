#!/bin/bash

set -e

# Compile the Go server
echo "Compiling server..."
go build -o server main.go

# Start server in background
echo "Starting server..."
./server &
SERVER_PID=$!
sleep 1

# Function to cleanup server on exit
cleanup() {
  echo "Stopping server..."
  kill $SERVER_PID
}
trap cleanup EXIT

# Test 1: Single request (should succeed)
echo "Test 1: Single request"
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/1

# Test 2: Rapid requests from same IP (should trigger rate limit)
echo "Test 2: Rapid requests (rate limit)"
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/2
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/2

# Test 3: Wait and request again (should succeed)
echo "Test 3: Wait for rate limit, then request"
sleep 1
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/2

# Test 4: Cache test (should hit cache on second request)
echo "Test 4: Cache test"
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/3
sleep 1
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/3

# Test 5: Cache expiry (wait for cache to expire)
echo "Test 5: Cache expiry"
sleep 5
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/todos/3

echo "All tests done."
