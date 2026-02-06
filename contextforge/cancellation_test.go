package contextforge

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestCancellationService_Cancel(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/cancellation/cancel", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"cancelled","requestId":"req-123","reason":"user requested"}`)
	})

	req := &CancellationRequest{
		RequestID: "req-123",
		Reason:    String("user requested"),
	}

	got, _, err := client.Cancel.Cancel(context.Background(), req)
	if err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}

	if got.Status != "cancelled" {
		t.Errorf("Cancel status = %q, want %q", got.Status, "cancelled")
	}
	if got.RequestID != "req-123" {
		t.Errorf("Cancel requestId = %q, want %q", got.RequestID, "req-123")
	}
}

func TestCancellationService_Cancel_NilRequest(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Cancel.Cancel(context.Background(), nil)
	if err == nil {
		t.Fatal("Cancel expected error for nil request, got nil")
	}
}

func TestCancellationService_Status(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/cancellation/status/req-456", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"tool:search","registered_at":1738790000.1,"cancelled":true,"cancelled_at":1738790001.2,"cancel_reason":"timeout"}`)
	})

	got, _, err := client.Cancel.Status(context.Background(), "req-456")
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	if !got.Cancelled {
		t.Errorf("Status cancelled = %v, want true", got.Cancelled)
	}
	if got.CancelReason == nil || *got.CancelReason != "timeout" {
		t.Errorf("Status cancel_reason = %v, want %q", got.CancelReason, "timeout")
	}
}
