package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setup() (client *Client, mux *http.ServeMux, serverURL string, teardown func()) {
	mux = http.NewServeMux()
	server := httptest.NewServer(mux)

	var err error
	client, err = NewClient(nil, server.URL+"/", "test-token")
	if err != nil {
		panic(fmt.Sprintf("Failed to create client: %v", err))
	}

	return client, mux, server.URL, server.Close
}

func testMethod(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func TestToolsService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Next-Cursor", "next123")
		fmt.Fprint(w, `[{"id":"1","name":"test-tool","description":"A test tool","enabled":true}]`)
	})

	ctx := context.Background()
	tools, resp, err := client.Tools.List(ctx, nil)

	if err != nil {
		t.Errorf("Tools.List returned error: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Tools.List returned %d tools, want 1", len(tools))
	}

	if tools[0].Name != "test-tool" {
		t.Errorf("Tools.List returned tool name %q, want %q", tools[0].Name, "test-tool")
	}

	if resp.NextCursor != "next123" {
		t.Errorf("Response.NextCursor = %q, want %q", resp.NextCursor, "next123")
	}
}

func TestToolsService_List_PaginatedEnvelope(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		if got := r.URL.Query().Get("include_pagination"); got != "true" {
			t.Errorf("include_pagination = %q, want %q", got, "true")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tools":[{"id":"1","name":"tool-one","enabled":true}],"nextCursor":"tool-cursor-1"}`)
	})

	tools, resp, err := client.Tools.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("Tools.List returned error: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("Tools.List returned %d tools, want 1", len(tools))
	}
	if resp.NextCursor != "tool-cursor-1" {
		t.Errorf("Response.NextCursor = %q, want %q", resp.NextCursor, "tool-cursor-1")
	}
}

func TestToolsService_Get(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/tools/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-tool","description":"A test tool","enabled":true}`)
	})

	ctx := context.Background()
	tool, _, err := client.Tools.Get(ctx, "123")

	if err != nil {
		t.Errorf("Tools.Get returned error: %v", err)
	}

	if tool.ID != "123" {
		t.Errorf("Tools.Get returned tool ID %q, want %q", tool.ID, "123")
	}

	if tool.Name != "test-tool" {
		t.Errorf("Tools.Get returned tool name %q, want %q", tool.Name, "test-tool")
	}
}

func TestToolsService_Create(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &Tool{
		Name:        "new-tool",
		Description: String("A new tool"),
		InputSchema: map[string]any{"type": "object"},
	}

	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has the tool wrapped in "tool" key
		var body map[string]*Tool
		json.NewDecoder(r.Body).Decode(&body)
		if body["tool"] == nil {
			t.Error("Expected request body to have 'tool' key")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"456","name":"new-tool","description":"A new tool","enabled":true}`)
	})

	ctx := context.Background()
	tool, _, err := client.Tools.Create(ctx, input, nil)

	if err != nil {
		t.Errorf("Tools.Create returned error: %v", err)
	}

	if tool.ID != "456" {
		t.Errorf("Tools.Create returned tool ID %q, want %q", tool.ID, "456")
	}

	if tool.Name != "new-tool" {
		t.Errorf("Tools.Create returned tool name %q, want %q", tool.Name, "new-tool")
	}
}

func TestToolsService_Update(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &Tool{
		Name:        "updated-tool",
		Description: String("An updated tool"),
	}

	mux.HandleFunc("/tools/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		// Verify the request body has the tool directly (not wrapped)
		var body Tool
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "updated-tool" {
			t.Errorf("Expected tool name 'updated-tool', got %q", body.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"updated-tool","description":"An updated tool","enabled":true}`)
	})

	ctx := context.Background()
	tool, _, err := client.Tools.Update(ctx, "123", input)

	if err != nil {
		t.Errorf("Tools.Update returned error: %v", err)
	}

	if tool.Name != "updated-tool" {
		t.Errorf("Tools.Update returned tool name %q, want %q", tool.Name, "updated-tool")
	}
}

func TestToolsService_Delete(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/tools/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	_, err := client.Tools.Delete(ctx, "123")

	if err != nil {
		t.Errorf("Tools.Delete returned error: %v", err)
	}
}

func TestToolsService_Toggle(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/tools/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "true" {
			t.Errorf("Expected activate=true query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","message":"Tool toggled","tool":{"id":"123","name":"test-tool","description":"A test tool","enabled":true}}`)
	})

	ctx := context.Background()
	tool, _, err := client.Tools.Toggle(ctx, "123", true)

	if err != nil {
		t.Errorf("Tools.Toggle returned error: %v", err)
	}

	if !tool.Enabled {
		t.Errorf("Tools.Toggle returned tool with enabled=%v, want true", tool.Enabled)
	}
}

func TestToolsService_SetState(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/tools/123/state", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		if got := r.URL.Query().Get("activate"); got != "false" {
			t.Errorf("activate = %q, want %q", got, "false")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","message":"Tool deactivated","tool":{"id":"123","name":"test-tool","enabled":false}}`)
	})

	tool, _, err := client.Tools.SetState(context.Background(), "123", false)
	if err != nil {
		t.Fatalf("Tools.SetState returned error: %v", err)
	}
	if tool == nil {
		t.Fatal("Tools.SetState returned nil tool")
	}
	if tool.Enabled {
		t.Errorf("Tools.SetState returned enabled = %v, want false", tool.Enabled)
	}
}
