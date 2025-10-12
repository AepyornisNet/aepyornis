#!/bin/bash

# Script to stop workout-tracker CI demo servers

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==> Stopping Workout Tracker CI Demo"

# Read PIDs from file if they exist
BACKEND_PID_FILE="$ROOT_DIR/tmp/backend.pid"
ANGULAR_PID_FILE="$ROOT_DIR/tmp/angular.pid"

if [ -f "$BACKEND_PID_FILE" ]; then
    BACKEND_PID=$(cat "$BACKEND_PID_FILE")
    if kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo "==> Stopping backend server (PID: $BACKEND_PID)..."
        kill "$BACKEND_PID"
    fi
    rm -f "$BACKEND_PID_FILE"
fi

if [ -f "$ANGULAR_PID_FILE" ]; then
    ANGULAR_PID=$(cat "$ANGULAR_PID_FILE")
    if kill -0 "$ANGULAR_PID" 2>/dev/null; then
        echo "==> Stopping Angular dev server (PID: $ANGULAR_PID)..."
        kill "$ANGULAR_PID"
    fi
    rm -f "$ANGULAR_PID_FILE"
fi

# Also try to kill by process name as backup
pkill -f "tmp/workout-tracker" 2>/dev/null || true
pkill -f "ng serve" 2>/dev/null || true

echo "==> ✅ Servers stopped"
