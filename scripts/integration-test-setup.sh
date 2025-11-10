#!/bin/bash
set -e

echo "🚀 Starting Context Forge test environment..."

# Get the project root directory (one level up from scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Check for uvx
if ! command -v uvx &> /dev/null; then
    echo "❌ uvx is required but not installed"
    echo "   Install with: brew install uv"
    exit 1
fi

echo "✓ Using uvx ($(uvx --version 2>&1))"

echo "🔧 Starting Context Forge gateway..."

# Clean up old database to ensure fresh state
echo "🗑️  Cleaning up old database files..."
rm -rfv "$PROJECT_ROOT/tmp" || true
mkdir -p "$PROJECT_ROOT/tmp"
echo "✓ Database cleanup complete"

# Change to project root to ensure database is created in the right place
cd "$PROJECT_ROOT"

# Set environment variables and start gateway in background
export MCPGATEWAY_ADMIN_API_ENABLED=true
export MCPGATEWAY_UI_ENABLED=true
export PLATFORM_ADMIN_EMAIL=admin@test.local
export PLATFORM_ADMIN_PASSWORD=testpassword123
export PLATFORM_ADMIN_FULL_NAME="Platform Administrator"
export DATABASE_URL=sqlite:///tmp/contextforge-test.db
export LOG_LEVEL=INFO
export REDIS_ENABLED=false
export OTEL_ENABLE_OBSERVABILITY=false
export AUTH_REQUIRED=false
export MCP_CLIENT_AUTH_ENABLED=false
export BASIC_AUTH_USER=admin
export BASIC_AUTH_PASSWORD=testpassword123
export JWT_SECRET_KEY="test-secret-key-for-integration-testing"
export SECURE_COOKIES=false

# Start gateway in background
uvx --from mcp-contextforge-gateway mcpgateway --host 0.0.0.0 --port 8000 > "$PROJECT_ROOT/tmp/contextforge-test.log" 2>&1 &
GATEWAY_PID=$!
echo $GATEWAY_PID > "$PROJECT_ROOT/tmp/contextforge-test.pid"

echo "⏳ Waiting for Context Forge to be ready..."

# Wait for health check
MAX_RETRIES=30
RETRY_COUNT=0

until curl -f http://localhost:8000/health > /dev/null 2>&1; do
  RETRY_COUNT=$((RETRY_COUNT + 1))
  if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
    echo "❌ Context Forge failed to start within timeout"
    cat "$PROJECT_ROOT/tmp/contextforge-test.log"
    kill $GATEWAY_PID 2>/dev/null || true
    exit 1
  fi
  echo "Waiting for Context Forge... (attempt $RETRY_COUNT/$MAX_RETRIES)"
  sleep 2
done

echo "✅ Context Forge is ready!"
echo ""

# Generate JWT token for integration tests
echo "🔑 Generating JWT token for integration tests..."
uvx --from mcp-contextforge-gateway python3 -m mcpgateway.utils.create_jwt_token \
  --username admin@test.local \
  --exp 10080 \
  --secret test-secret-key-for-integration-testing > "$PROJECT_ROOT/tmp/contextforge-test-token.txt" 2>&1

if [ $? -eq 0 ]; then
  # Extract just the token (remove any extra output)
  TOKEN=$(cat "$PROJECT_ROOT/tmp/contextforge-test-token.txt" | grep -o 'eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*' || cat "$PROJECT_ROOT/tmp/contextforge-test-token.txt")
  echo "$TOKEN" > "$PROJECT_ROOT/tmp/contextforge-test-token.txt"
  export CONTEXTFORGE_TEST_TOKEN="$TOKEN"
  echo "✅ JWT token generated successfully"
else
  echo "⚠️  Failed to generate JWT token, tests may fail"
  cat "$PROJECT_ROOT/tmp/contextforge-test-token.txt"
fi
echo ""

echo "📊 Gateway Information:"
echo "   Address: http://localhost:8000"
echo "   Admin Email: admin@test.local"
echo "   Admin Password: testpassword123"
echo "   PID: $GATEWAY_PID"
echo "   Token File: $PROJECT_ROOT/tmp/contextforge-test-token.txt"
echo ""

# Get version info
echo "🔍 Gateway Version:"
curl -s -u admin@test.local:testpassword123 http://localhost:8000/version | jq . || echo "Version endpoint unavailable"
echo ""

echo "✨ Test environment is ready for integration tests!"
