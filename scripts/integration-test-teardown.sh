#!/bin/bash
set -e

echo "🧹 Cleaning up Context Forge test environment..."

# Get the project root directory (one level up from scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Stop gateway process
if [ -f "$PROJECT_ROOT/tmp/contextforge-test.pid" ]; then
    PID=$(cat "$PROJECT_ROOT/tmp/contextforge-test.pid")
    if ps -p $PID > /dev/null 2>&1; then
        echo "🛑 Stopping gateway (PID: $PID)..."
        kill $PID 2>/dev/null || true
        sleep 2
        # Force kill if still running
        kill -9 $PID 2>/dev/null || true
    fi
fi

# Clean up test artifacts directory
echo "🗑️  Removing test artifacts..."
rm -rf "$PROJECT_ROOT/tmp"

echo "✅ Test environment cleaned up!"
