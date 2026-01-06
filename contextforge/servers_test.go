package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestServersService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Next-Cursor", "next123")
		fmt.Fprint(w, `[{"id":"1","name":"test-server","description":"A test","isActive":true,"metrics":{"totalExecutions":10,"successfulExecutions":9,"failedExecutions":1,"failureRate":0.1}}]`)
	})

	ctx := context.Background()
	servers, resp, err := client.Servers.List(ctx, nil)

	if err != nil {
		t.Errorf("Servers.List returned error: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("Servers.List returned %d servers, want 1", len(servers))
	}

	if servers[0].Name != "test-server" {
		t.Errorf("Servers.List returned server name %q, want %q", servers[0].Name, "test-server")
	}

	if resp.NextCursor != "next123" {
		t.Errorf("Response.NextCursor = %q, want %q", resp.NextCursor, "next123")
	}
}

func TestServersService_List_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameters
		q := r.URL.Query()
		if got := q.Get("include_inactive"); got != "true" {
			t.Errorf("include_inactive = %q, want %q", got, "true")
		}
		if got := q.Get("tags"); got != "test,demo" {
			t.Errorf("tags = %q, want %q", got, "test,demo")
		}
		if got := q.Get("visibility"); got != "public" {
			t.Errorf("visibility = %q, want %q", got, "public")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	opts := &ServerListOptions{
		IncludeInactive: true,
		Tags:            "test,demo",
		Visibility:      "public",
	}

	ctx := context.Background()
	_, _, err := client.Servers.List(ctx, opts)

	if err != nil {
		t.Errorf("Servers.List returned error: %v", err)
	}
}

func TestServersService_Get(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-server","description":"A test","isActive":true,"metrics":{"totalExecutions":10,"successfulExecutions":9,"failedExecutions":1,"failureRate":0.1}}`)
	})

	ctx := context.Background()
	server, _, err := client.Servers.Get(ctx, "123")

	if err != nil {
		t.Errorf("Servers.Get returned error: %v", err)
	}

	if server.ID != "123" {
		t.Errorf("Servers.Get returned server ID %q, want %q", server.ID, "123")
	}

	if server.Name != "test-server" {
		t.Errorf("Servers.Get returned server name %q, want %q", server.Name, "test-server")
	}
}

func TestServersService_Create(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &ServerCreate{
		Name:        "new-server",
		Description: String("A new server"),
		Tags:        []string{"test"},
	}

	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has the server wrapped in "server" key
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["server"] == nil {
			t.Error("Expected request body to have 'server' key")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"456","name":"new-server","description":"A new server","isActive":true,"tags":[{"id":"test","label":"test"}],"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	server, _, err := client.Servers.Create(ctx, input, nil)

	if err != nil {
		t.Errorf("Servers.Create returned error: %v", err)
	}

	if server.ID != "456" {
		t.Errorf("Servers.Create returned server ID %q, want %q", server.ID, "456")
	}

	if server.Name != "new-server" {
		t.Errorf("Servers.Create returned server name %q, want %q", server.Name, "new-server")
	}
}

func TestServersService_Create_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &ServerCreate{
		Name: "new-server",
	}

	opts := &ServerCreateOptions{
		TeamID:     String("team-123"),
		Visibility: String("private"),
	}

	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has team_id and visibility at top level
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["team_id"] == nil {
			t.Error("Expected request body to have 'team_id' key")
		}
		if body["visibility"] == nil {
			t.Error("Expected request body to have 'visibility' key")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"456","name":"new-server","isActive":true,"teamId":"team-123","visibility":"private","metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	server, _, err := client.Servers.Create(ctx, input, opts)

	if err != nil {
		t.Errorf("Servers.Create returned error: %v", err)
	}

	if server.TeamID == nil || *server.TeamID != "team-123" {
		t.Errorf("Servers.Create returned server with teamId %v, want %q", server.TeamID, "team-123")
	}
}

func TestServersService_Update(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &ServerUpdate{
		Name:        String("updated-server"),
		Description: String("An updated server"),
	}

	mux.HandleFunc("/servers/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		// Verify the request body is NOT wrapped (ServerUpdate is sent directly)
		var body ServerUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body as ServerUpdate: %v", err)
		}
		if body.Name == nil || *body.Name != "updated-server" {
			t.Error("Expected request body to be ServerUpdate (not wrapped)")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"updated-server","description":"An updated server","isActive":true,"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	server, _, err := client.Servers.Update(ctx, "123", input)

	if err != nil {
		t.Errorf("Servers.Update returned error: %v", err)
	}

	if server.Name != "updated-server" {
		t.Errorf("Servers.Update returned server name %q, want %q", server.Name, "updated-server")
	}
}

func TestServersService_Delete(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	_, err := client.Servers.Delete(ctx, "123")

	if err != nil {
		t.Errorf("Servers.Delete returned error: %v", err)
	}
}

func TestServersService_Toggle(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "true" {
			t.Errorf("Expected activate=true query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-server","isActive":true,"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	server, _, err := client.Servers.Toggle(ctx, "123", true)

	if err != nil {
		t.Errorf("Servers.Toggle returned error: %v", err)
	}

	if !server.IsActive {
		t.Errorf("Servers.Toggle returned server with isActive=%v, want true", server.IsActive)
	}
}

func TestServersService_Toggle_Deactivate(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "false" {
			t.Errorf("Expected activate=false query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-server","isActive":false,"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	server, _, err := client.Servers.Toggle(ctx, "123", false)

	if err != nil {
		t.Errorf("Servers.Toggle returned error: %v", err)
	}

	if server.IsActive {
		t.Errorf("Servers.Toggle returned server with isActive=%v, want false", server.IsActive)
	}
}

func TestServersService_ListTools(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/tools", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"tool-1","name":"test-tool","description":"A test tool","enabled":true}]`)
	})

	ctx := context.Background()
	tools, _, err := client.Servers.ListTools(ctx, "123", nil)

	if err != nil {
		t.Errorf("Servers.ListTools returned error: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Servers.ListTools returned %d tools, want 1", len(tools))
	}

	if tools[0].Name != "test-tool" {
		t.Errorf("Servers.ListTools returned tool name %q, want %q", tools[0].Name, "test-tool")
	}
}

func TestServersService_ListTools_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/tools", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameter
		if got := r.URL.Query().Get("include_inactive"); got != "true" {
			t.Errorf("include_inactive = %q, want %q", got, "true")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	opts := &ServerAssociationOptions{
		IncludeInactive: true,
	}

	ctx := context.Background()
	_, _, err := client.Servers.ListTools(ctx, "123", opts)

	if err != nil {
		t.Errorf("Servers.ListTools returned error: %v", err)
	}
}

func TestServersService_ListResources(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/resources", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","uri":"file://test.txt","name":"test-resource","isActive":true}]`)
	})

	ctx := context.Background()
	resources, _, err := client.Servers.ListResources(ctx, "123", nil)

	if err != nil {
		t.Errorf("Servers.ListResources returned error: %v", err)
	}

	if len(resources) != 1 {
		t.Errorf("Servers.ListResources returned %d resources, want 1", len(resources))
	}

	if resources[0].Name != "test-resource" {
		t.Errorf("Servers.ListResources returned resource name %q, want %q", resources[0].Name, "test-resource")
	}
}

func TestServersService_ListResources_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/resources", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameter
		if got := r.URL.Query().Get("include_inactive"); got != "true" {
			t.Errorf("include_inactive = %q, want %q", got, "true")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	opts := &ServerAssociationOptions{
		IncludeInactive: true,
	}

	ctx := context.Background()
	_, _, err := client.Servers.ListResources(ctx, "123", opts)

	if err != nil {
		t.Errorf("Servers.ListResources returned error: %v", err)
	}
}

func TestServersService_ListPrompts(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/prompts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","name":"test-prompt","template":"Hello {{name}}","arguments":[],"isActive":true,"tags":[],"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}]`)
	})

	ctx := context.Background()
	prompts, _, err := client.Servers.ListPrompts(ctx, "123", nil)

	if err != nil {
		t.Errorf("Servers.ListPrompts returned error: %v", err)
	}

	if len(prompts) != 1 {
		t.Errorf("Servers.ListPrompts returned %d prompts, want 1", len(prompts))
	}

	if prompts[0].Name != "test-prompt" {
		t.Errorf("Servers.ListPrompts returned prompt name %q, want %q", prompts[0].Name, "test-prompt")
	}
}

func TestServersService_ListPrompts_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/servers/123/prompts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameter
		if got := r.URL.Query().Get("include_inactive"); got != "true" {
			t.Errorf("include_inactive = %q, want %q", got, "true")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	opts := &ServerAssociationOptions{
		IncludeInactive: true,
	}

	ctx := context.Background()
	_, _, err := client.Servers.ListPrompts(ctx, "123", opts)

	if err != nil {
		t.Errorf("Servers.ListPrompts returned error: %v", err)
	}
}
