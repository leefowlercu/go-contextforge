//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

// TestCancellationService_Basic verifies cancellation endpoints are reachable
// and behave as expected for an unknown request ID.
func TestCancellationService_Basic(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	requestID := fmt.Sprintf("integration-cancel-%d", time.Now().UnixNano())
	reason := "integration-test"

	result, _, err := client.Cancel.Cancel(ctx, &contextforge.CancellationRequest{
		RequestID: requestID,
		Reason:    contextforge.String(reason),
	})
	if err != nil {
		t.Fatalf("Cancel request failed: %v", err)
	}
	if result == nil {
		t.Fatal("Cancel returned nil response")
	}
	if result.RequestID != requestID {
		t.Errorf("Expected requestId %q, got %q", requestID, result.RequestID)
	}
	if result.Status != "queued" && result.Status != "cancelled" {
		t.Errorf("Expected status queued|cancelled, got %q", result.Status)
	}

	status, _, err := client.Cancel.Status(ctx, requestID)
	if err == nil {
		if status == nil {
			t.Fatal("Status returned nil response without error")
		}
		t.Logf("Cancellation status exists for %s (cancelled=%v)", requestID, status.Cancelled)
		return
	}

	// Unknown request IDs may return 404 (expected for queued remote cancellation).
	if apiErr, ok := err.(*contextforge.ErrorResponse); ok && apiErr.Response.StatusCode == http.StatusNotFound {
		t.Logf("Status correctly returned 404 for unknown run %s", requestID)
		return
	}

	t.Fatalf("Status returned unexpected error: %v", err)
}
