#!/bin/bash
set -e

echo "🧹 Cleaning up Context Forge test environment..."

# Stop gateway process
if [ -f /tmp/contextforge-test.pid ]; then
    PID=$(cat /tmp/contextforge-test.pid)
    if ps -p $PID > /dev/null 2>&1; then
        echo "🛑 Stopping gateway (PID: $PID)..."
        kill $PID 2>/dev/null || true
        sleep 2
        # Force kill if still running
        kill -9 $PID 2>/dev/null || true
    fi
    rm -f /tmp/contextforge-test.pid
fi

# Clean up test artifacts
rm -f /tmp/contextforge-test.db*
rm -f /tmp/contextforge-test.log
rm -f /tmp/contextforge-test-token.txt

echo "✅ Test environment cleaned up!"
