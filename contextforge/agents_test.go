package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestAgentsService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id":"1","name":"test-agent","slug":"test-agent","endpointUrl":"https://example.com/agent","agentType":"generic","protocolVersion":"1.0","enabled":true,"reachable":true}]`)
	})

	ctx := context.Background()
	agents, _, err := client.Agents.List(ctx, nil)

	if err != nil {
		t.Errorf("Agents.List returned error: %v", err)
	}

	if len(agents) != 1 {
		t.Errorf("Agents.List returned %d agents, want 1", len(agents))
	}

	if agents[0].Name != "test-agent" {
		t.Errorf("Agents.List returned agent name %q, want %q", agents[0].Name, "test-agent")
	}

	if agents[0].EndpointURL != "https://example.com/agent" {
		t.Errorf("Agents.List returned endpoint URL %q, want %q", agents[0].EndpointURL, "https://example.com/agent")
	}
}

func TestAgentsService_List_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		// Verify query parameters (skip/limit instead of cursor)
		q := r.URL.Query()
		if got := q.Get("skip"); got != "10" {
			t.Errorf("skip = %q, want %q", got, "10")
		}
		if got := q.Get("limit"); got != "50" {
			t.Errorf("limit = %q, want %q", got, "50")
		}
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

	opts := &AgentListOptions{
		Skip:            10,
		Limit:           50,
		IncludeInactive: true,
		Tags:            "test,demo",
		Visibility:      "public",
	}

	ctx := context.Background()
	_, _, err := client.Agents.List(ctx, opts)

	if err != nil {
		t.Errorf("Agents.List returned error: %v", err)
	}
}

func TestAgentsService_Get(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-agent","slug":"test-agent","endpointUrl":"https://example.com/agent","agentType":"generic","protocolVersion":"1.0","enabled":true,"reachable":true}`)
	})

	ctx := context.Background()
	agent, _, err := client.Agents.Get(ctx, "123")

	if err != nil {
		t.Errorf("Agents.Get returned error: %v", err)
	}

	if agent.ID != "123" {
		t.Errorf("Agents.Get returned agent ID %q, want %q", agent.ID, "123")
	}

	if agent.Name != "test-agent" {
		t.Errorf("Agents.Get returned agent name %q, want %q", agent.Name, "test-agent")
	}
}

func TestAgentsService_Create(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &AgentCreate{
		Name:        "new-agent",
		EndpointURL: "https://example.com/new-agent",
		Description: String("A new agent"),
		Tags:        []string{"test"},
	}

	mux.HandleFunc("/a2a", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has the agent wrapped in "agent" key
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["agent"] == nil {
			t.Error("Expected request body to have 'agent' key")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"456","name":"new-agent","slug":"new-agent","endpointUrl":"https://example.com/new-agent","description":"A new agent","agentType":"generic","protocolVersion":"1.0","enabled":true,"reachable":false,"tags":["test"]}`)
	})

	ctx := context.Background()
	agent, _, err := client.Agents.Create(ctx, input, nil)

	if err != nil {
		t.Errorf("Agents.Create returned error: %v", err)
	}

	if agent.ID != "456" {
		t.Errorf("Agents.Create returned agent ID %q, want %q", agent.ID, "456")
	}

	if agent.Name != "new-agent" {
		t.Errorf("Agents.Create returned agent name %q, want %q", agent.Name, "new-agent")
	}
}

func TestAgentsService_Create_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &AgentCreate{
		Name:        "new-agent",
		EndpointURL: "https://example.com/new-agent",
	}

	opts := &AgentCreateOptions{
		TeamID:     String("team-123"),
		Visibility: String("private"),
	}

	mux.HandleFunc("/a2a", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has team_id and visibility at top level
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["team_id"] != "team-123" {
			t.Errorf("Expected team_id = %q, got %v", "team-123", body["team_id"])
		}
		if body["visibility"] != "private" {
			t.Errorf("Expected visibility = %q, got %v", "private", body["visibility"])
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"789","name":"new-agent","slug":"new-agent","endpointUrl":"https://example.com/new-agent","agentType":"generic","protocolVersion":"1.0","enabled":true,"reachable":false}`)
	})

	ctx := context.Background()
	agent, _, err := client.Agents.Create(ctx, input, opts)

	if err != nil {
		t.Errorf("Agents.Create returned error: %v", err)
	}

	if agent.ID != "789" {
		t.Errorf("Agents.Create returned agent ID %q, want %q", agent.ID, "789")
	}
}

func TestAgentsService_Update(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &AgentUpdate{
		Description: String("Updated description"),
		Tags:        []string{"updated"},
	}

	mux.HandleFunc("/a2a/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		// Verify the request body is NOT wrapped (direct body)
		var body AgentUpdate
		json.NewDecoder(r.Body).Decode(&body)
		if body.Description == nil || *body.Description != "Updated description" {
			t.Error("Expected request body to have description field directly")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-agent","slug":"test-agent","endpointUrl":"https://example.com/agent","description":"Updated description","agentType":"generic","protocolVersion":"1.0","enabled":true,"reachable":true,"tags":["updated"]}`)
	})

	ctx := context.Background()
	agent, _, err := client.Agents.Update(ctx, "123", input)

	if err != nil {
		t.Errorf("Agents.Update returned error: %v", err)
	}

	if agent.ID != "123" {
		t.Errorf("Agents.Update returned agent ID %q, want %q", agent.ID, "123")
	}

	if agent.Description == nil || *agent.Description != "Updated description" {
		t.Errorf("Agents.Update did not update description")
	}
}

func TestAgentsService_Delete(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	_, err := client.Agents.Delete(ctx, "123")

	if err != nil {
		t.Errorf("Agents.Delete returned error: %v", err)
	}
}

func TestAgentsService_Toggle(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		if got := r.URL.Query().Get("activate"); got != "false" {
			t.Errorf("activate = %q, want %q", got, "false")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-agent","slug":"test-agent","endpointUrl":"https://example.com/agent","agentType":"generic","protocolVersion":"1.0","enabled":false,"reachable":true}`)
	})

	ctx := context.Background()
	agent, _, err := client.Agents.Toggle(ctx, "123", false)

	if err != nil {
		t.Errorf("Agents.Toggle returned error: %v", err)
	}

	if agent.Enabled {
		t.Errorf("Agents.Toggle returned enabled = true, want false")
	}
}

func TestAgentsService_Toggle_Activate(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		if got := r.URL.Query().Get("activate"); got != "true" {
			t.Errorf("activate = %q, want %q", got, "true")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"123","name":"test-agent","slug":"test-agent","endpointUrl":"https://example.com/agent","agentType":"generic","protocolVersion":"1.0","enabled":true,"reachable":true}`)
	})

	ctx := context.Background()
	agent, _, err := client.Agents.Toggle(ctx, "123", true)

	if err != nil {
		t.Errorf("Agents.Toggle returned error: %v", err)
	}

	if !agent.Enabled {
		t.Errorf("Agents.Toggle returned enabled = false, want true")
	}
}

func TestAgentsService_Invoke(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &AgentInvokeRequest{
		Parameters: map[string]any{
			"query": "test query",
		},
		InteractionType: "query",
	}

	mux.HandleFunc("/a2a/test-agent/invoke", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body
		var body AgentInvokeRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Parameters == nil {
			t.Error("Expected request body to have parameters")
		}
		if body.InteractionType != "query" {
			t.Errorf("Expected interaction_type = %q, got %q", "query", body.InteractionType)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":"success","data":"response data"}`)
	})

	ctx := context.Background()
	result, _, err := client.Agents.Invoke(ctx, "test-agent", input)

	if err != nil {
		t.Errorf("Agents.Invoke returned error: %v", err)
	}

	if result["result"] != "success" {
		t.Errorf("Agents.Invoke returned result %q, want %q", result["result"], "success")
	}
}

func TestAgentsService_Invoke_Nil(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/a2a/test-agent/invoke", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":"success"}`)
	})

	ctx := context.Background()
	result, _, err := client.Agents.Invoke(ctx, "test-agent", nil)

	if err != nil {
		t.Errorf("Agents.Invoke returned error: %v", err)
	}

	if result["result"] != "success" {
		t.Errorf("Agents.Invoke returned result %q, want %q", result["result"], "success")
	}
}

func TestAgentsService_URLEscaping(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	// Test that agent IDs and names with special characters are properly escaped
	mux.HandleFunc("/a2a/agent%20with%20spaces/invoke", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"result":"success"}`)
	})

	ctx := context.Background()
	_, _, err := client.Agents.Invoke(ctx, "agent with spaces", nil)

	if err != nil {
		t.Errorf("Agents.Invoke with URL escaping returned error: %v", err)
	}
}
