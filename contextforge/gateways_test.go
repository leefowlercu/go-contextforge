package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestGatewaysService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Next-Cursor", "next456")
		fmt.Fprint(w, `[{"id":"1","name":"test-gateway","url":"https://example.com","enabled":true}]`)
	})

	ctx := context.Background()
	gateways, resp, err := client.Gateways.List(ctx, nil)

	if err != nil {
		t.Errorf("Gateways.List returned error: %v", err)
	}

	if len(gateways) != 1 {
		t.Errorf("Gateways.List returned %d gateways, want 1", len(gateways))
	}

	if gateways[0].Name != "test-gateway" {
		t.Errorf("Gateways.List returned gateway name %q, want %q", gateways[0].Name, "test-gateway")
	}

	if resp.NextCursor != "next456" {
		t.Errorf("Response.NextCursor = %q, want %q", resp.NextCursor, "next456")
	}
}

func TestGatewaysService_List_PaginatedEnvelope(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		if got := r.URL.Query().Get("include_pagination"); got != "true" {
			t.Errorf("include_pagination = %q, want %q", got, "true")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"gateways":[{"id":"1","name":"gw-1","url":"https://example.com","enabled":true}],"nextCursor":"gw-cursor-1"}`)
	})

	gateways, resp, err := client.Gateways.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("Gateways.List returned error: %v", err)
	}
	if len(gateways) != 1 {
		t.Fatalf("Gateways.List returned %d gateways, want 1", len(gateways))
	}
	if resp.NextCursor != "gw-cursor-1" {
		t.Errorf("Response.NextCursor = %q, want %q", resp.NextCursor, "gw-cursor-1")
	}
}

func TestGatewaysService_ListWithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameters
		includeInactive := r.URL.Query().Get("include_inactive")
		if includeInactive != "true" {
			t.Errorf("Expected include_inactive=true query parameter, got %q", includeInactive)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","name":"test-gateway","url":"https://example.com","enabled":false}]`)
	})

	ctx := context.Background()
	opts := &GatewayListOptions{
		IncludeInactive: true,
	}
	gateways, _, err := client.Gateways.List(ctx, opts)

	if err != nil {
		t.Errorf("Gateways.List returned error: %v", err)
	}

	if len(gateways) != 1 {
		t.Errorf("Gateways.List returned %d gateways, want 1", len(gateways))
	}
}

func TestGatewaysService_Get(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/abc123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"abc123","name":"test-gateway","url":"https://example.com","enabled":true}`)
	})

	ctx := context.Background()
	gateway, _, err := client.Gateways.Get(ctx, "abc123")

	if err != nil {
		t.Errorf("Gateways.Get returned error: %v", err)
	}

	if *gateway.ID != "abc123" {
		t.Errorf("Gateways.Get returned gateway ID %q, want %q", *gateway.ID, "abc123")
	}

	if gateway.Name != "test-gateway" {
		t.Errorf("Gateways.Get returned gateway name %q, want %q", gateway.Name, "test-gateway")
	}
}

func TestGatewaysService_Get_QueryParamAuthAndRefreshFields(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/qp1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"qp1","name":"query-auth-gateway","url":"https://example.com","authType":"query_param","authQueryParamKey":"api_key","authQueryParamValueMasked":"******","refreshIntervalSeconds":600,"enabled":true}`)
	})

	gateway, _, err := client.Gateways.Get(context.Background(), "qp1")
	if err != nil {
		t.Fatalf("Gateways.Get returned error: %v", err)
	}

	if gateway.AuthQueryParamKey == nil || *gateway.AuthQueryParamKey != "api_key" {
		t.Errorf("AuthQueryParamKey = %v, want %q", gateway.AuthQueryParamKey, "api_key")
	}
	if gateway.AuthQueryParamValueMasked == nil || *gateway.AuthQueryParamValueMasked != "******" {
		t.Errorf("AuthQueryParamValueMasked = %v, want %q", gateway.AuthQueryParamValueMasked, "******")
	}
	if gateway.RefreshIntervalSeconds == nil || *gateway.RefreshIntervalSeconds != 600 {
		t.Errorf("RefreshIntervalSeconds = %v, want %d", gateway.RefreshIntervalSeconds, 600)
	}
}

func TestGatewaysService_Create(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &Gateway{
		Name:        "new-gateway",
		URL:         "https://newgateway.com",
		Description: String("A new gateway"),
		Transport:   "SSE",
	}

	mux.HandleFunc("/gateways", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has gateway fields at top level (NOT wrapped)
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Check that gateway fields are at top level (not wrapped in "gateway" key)
		if _, ok := body["name"]; !ok {
			t.Error("Expected request body to have 'name' field at top level")
		}
		if _, ok := body["url"]; !ok {
			t.Error("Expected request body to have 'url' field at top level")
		}
		// Ensure NOT wrapped
		if _, ok := body["gateway"]; ok {
			t.Error("Request body should NOT have 'gateway' wrapper")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"def456","name":"new-gateway","url":"https://newgateway.com","description":"A new gateway","enabled":true}`)
	})

	ctx := context.Background()
	gateway, _, err := client.Gateways.Create(ctx, input, nil)

	if err != nil {
		t.Errorf("Gateways.Create returned error: %v", err)
	}

	if *gateway.ID != "def456" {
		t.Errorf("Gateways.Create returned gateway ID %q, want %q", *gateway.ID, "def456")
	}

	if gateway.Name != "new-gateway" {
		t.Errorf("Gateways.Create returned gateway name %q, want %q", gateway.Name, "new-gateway")
	}
}

func TestGatewaysService_Update(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &Gateway{
		Name:        "updated-gateway",
		URL:         "https://updated.com",
		Description: String("An updated gateway"),
	}

	mux.HandleFunc("/gateways/abc123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		// Verify the request body is NOT wrapped (different from tools)
		var body Gateway
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body.Name != "updated-gateway" {
			t.Errorf("Expected request body to have name 'updated-gateway', got %q", body.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"abc123","name":"updated-gateway","url":"https://updated.com","description":"An updated gateway","enabled":true}`)
	})

	ctx := context.Background()
	gateway, _, err := client.Gateways.Update(ctx, "abc123", input)

	if err != nil {
		t.Errorf("Gateways.Update returned error: %v", err)
	}

	if gateway.Name != "updated-gateway" {
		t.Errorf("Gateways.Update returned gateway name %q, want %q", gateway.Name, "updated-gateway")
	}
}

func TestGatewaysService_Delete(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/abc123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	_, err := client.Gateways.Delete(ctx, "abc123")

	if err != nil {
		t.Errorf("Gateways.Delete returned error: %v", err)
	}
}

func TestGatewaysService_Toggle(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/abc123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "true" {
			t.Errorf("Expected activate=true query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","message":"Gateway toggled","gateway":{"id":"abc123","name":"test-gateway","url":"https://example.com","enabled":true}}`)
	})

	ctx := context.Background()
	gateway, _, err := client.Gateways.Toggle(ctx, "abc123", true)

	if err != nil {
		t.Errorf("Gateways.Toggle returned error: %v", err)
	}

	if !gateway.Enabled {
		t.Errorf("Gateways.Toggle returned gateway with enabled=%v, want true", gateway.Enabled)
	}
}

func TestGatewaysService_ToggleDeactivate(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/abc123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "false" {
			t.Errorf("Expected activate=false query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","message":"Gateway toggled","gateway":{"id":"abc123","name":"test-gateway","url":"https://example.com","enabled":false}}`)
	})

	ctx := context.Background()
	gateway, _, err := client.Gateways.Toggle(ctx, "abc123", false)

	if err != nil {
		t.Errorf("Gateways.Toggle returned error: %v", err)
	}

	if gateway.Enabled {
		t.Errorf("Gateways.Toggle returned gateway with enabled=%v, want false", gateway.Enabled)
	}
}

func TestGatewaysService_SetState(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/abc123/state", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		if got := r.URL.Query().Get("activate"); got != "false" {
			t.Errorf("activate = %q, want %q", got, "false")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","gateway":{"id":"abc123","name":"test-gateway","url":"https://example.com","enabled":false}}`)
	})

	gateway, _, err := client.Gateways.SetState(context.Background(), "abc123", false)
	if err != nil {
		t.Fatalf("Gateways.SetState returned error: %v", err)
	}
	if gateway == nil {
		t.Fatal("Gateways.SetState returned nil gateway")
	}
	if gateway.Enabled {
		t.Errorf("Gateways.SetState returned enabled = %v, want false", gateway.Enabled)
	}
}

func TestGatewaysService_RefreshTools(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/gateways/abc123/tools/refresh", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		if got := r.URL.Query().Get("include_resources"); got != "true" {
			t.Errorf("include_resources = %q, want %q", got, "true")
		}
		if got := r.URL.Query().Get("include_prompts"); got != "true" {
			t.Errorf("include_prompts = %q, want %q", got, "true")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"gateway_id":"abc123","success":true,"tools_added":2,"resources_added":1,"prompts_added":1,"validation_errors":[],"duration_ms":21.5,"refreshed_at":"2026-02-05T21:00:00Z"}`)
	})

	opts := &GatewayRefreshOptions{IncludeResources: true, IncludePrompts: true}
	result, _, err := client.Gateways.RefreshTools(context.Background(), "abc123", opts)
	if err != nil {
		t.Fatalf("Gateways.RefreshTools returned error: %v", err)
	}
	if result.GatewayID != "abc123" {
		t.Errorf("GatewayID = %q, want %q", result.GatewayID, "abc123")
	}
	if result.ToolsAdded != 2 {
		t.Errorf("ToolsAdded = %d, want %d", result.ToolsAdded, 2)
	}
}
