#!/bin/bash

# Script to start workout-tracker in CI environment for testing/demo purposes
# This script builds and runs both the backend (with SQLite) and Angular dev server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==> Starting Workout Tracker CI Demo"
echo "==> Root directory: $ROOT_DIR"

# Navigate to root directory
cd "$ROOT_DIR"

# Create necessary directories
mkdir -p data tmp

# Step 1: Build the backend binary with SQLite support
echo ""
echo "==> Building backend binary..."
make build-server

# Check if binary was created
if [ ! -f "tmp/workout-tracker" ]; then
    echo "ERROR: Backend binary not found at tmp/workout-tracker"
    exit 1
fi

# Step 2: Set environment variables for SQLite and development
export WT_DATABASE_DRIVER=sqlite
export WT_DSN=./data/workout-tracker-ci.db
export WT_JWT_ENCRYPTION_KEY=ci-test-key-not-for-production

# Step 3: Start the backend server in the background
echo ""
echo "==> Starting backend server on port 8080..."
./tmp/workout-tracker &
BACKEND_PID=$!

echo "Backend started with PID: $BACKEND_PID"

# Wait for backend to be ready
echo "==> Waiting for backend to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8080/api/v2/app-info > /dev/null 2>&1; then
        echo "Backend is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "ERROR: Backend failed to start within 30 seconds"
        kill $BACKEND_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Step 4: Install Angular dependencies if needed
cd "$ROOT_DIR/client"
if [ ! -d "node_modules" ]; then
    echo ""
    echo "==> Installing Angular dependencies..."
    npm install
fi

# Step 5: Start Angular dev server in the background
echo ""
echo "==> Starting Angular dev server on port 4200..."
npm run start &
ANGULAR_PID=$!

echo "Angular dev server started with PID: $ANGULAR_PID"

# Wait for Angular dev server to be ready
echo "==> Waiting for Angular dev server to be ready..."
for i in {1..60}; do
    if curl -s http://localhost:4200 > /dev/null 2>&1; then
        echo "Angular dev server is ready!"
        break
    fi
    if [ $i -eq 60 ]; then
        echo "ERROR: Angular dev server failed to start within 60 seconds"
        kill $BACKEND_PID $ANGULAR_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# Save PIDs to file for cleanup
echo "$BACKEND_PID" > "$ROOT_DIR/tmp/backend.pid"
echo "$ANGULAR_PID" > "$ROOT_DIR/tmp/angular.pid"

echo ""
echo "==> ✅ Workout Tracker is now running!"
echo ""
echo "Backend API:    http://localhost:8080"
echo "Angular Client: http://localhost:4200"
echo ""
echo "To test the workout creation page, navigate to:"
echo "http://localhost:4200/workouts/add"
echo ""
echo "To stop the servers, run:"
echo "  scripts/stop-ci-demo.sh"
echo ""
echo "Or manually kill processes:"
echo "  kill $BACKEND_PID $ANGULAR_PID"
echo ""

# Keep script running
echo "==> Press Ctrl+C to stop all servers..."
trap "echo ''; echo '==> Stopping servers...'; kill $BACKEND_PID $ANGULAR_PID 2>/dev/null || true; exit 0" INT TERM

# Wait for user interrupt or process to end
wait
