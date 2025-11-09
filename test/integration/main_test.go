//go:build integration
// +build integration

package integration

import (
	"os"
	"testing"
)

// TestMain sets up and tears down the mock MCP server for all integration tests
func TestMain(m *testing.M) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		os.Exit(m.Run())
	}

	// Start mock MCP server
	InitMockServer()

	// Run tests
	exitCode := m.Run()

	// Cleanup
	CloseMockServer()

	os.Exit(exitCode)
}
