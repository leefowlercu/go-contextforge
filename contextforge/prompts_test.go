package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestPromptsService_List(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/prompts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Next-Cursor", "next123")
		fmt.Fprint(w, `[{"id":1,"name":"test-prompt","description":"A test","template":"Hello {{name}}","arguments":[],"isActive":true,"tags":[],"metrics":{"totalExecutions":10,"successfulExecutions":9,"failedExecutions":1,"failureRate":0.1}}]`)
	})

	ctx := context.Background()
	prompts, resp, err := client.Prompts.List(ctx, nil)

	if err != nil {
		t.Errorf("Prompts.List returned error: %v", err)
	}

	if len(prompts) != 1 {
		t.Errorf("Prompts.List returned %d prompts, want 1", len(prompts))
	}

	if prompts[0].Name != "test-prompt" {
		t.Errorf("Prompts.List returned prompt name %q, want %q", prompts[0].Name, "test-prompt")
	}

	if resp.NextCursor != "next123" {
		t.Errorf("Response.NextCursor = %q, want %q", resp.NextCursor, "next123")
	}
}

func TestPromptsService_List_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/prompts", func(w http.ResponseWriter, r *http.Request) {
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
		if got := q.Get("team_id"); got != "team-123" {
			t.Errorf("team_id = %q, want %q", got, "team-123")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	})

	opts := &PromptListOptions{
		IncludeInactive: true,
		Tags:            "test,demo",
		Visibility:      "public",
		TeamID:          "team-123",
	}

	ctx := context.Background()
	_, _, err := client.Prompts.List(ctx, opts)

	if err != nil {
		t.Errorf("Prompts.List returned error: %v", err)
	}
}

func TestPromptsService_Create(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &PromptCreate{
		Name:        "new-prompt",
		Description: String("A new prompt"),
		Template:    "Hello {{name}}",
		Arguments: []PromptArgument{
			{Name: "name", Required: true},
		},
		Tags: []string{"test"},
	}

	mux.HandleFunc("/prompts", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify the request body has the prompt wrapped in "prompt" key
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["prompt"] == nil {
			t.Error("Expected request body to have 'prompt' key")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":456,"name":"new-prompt","description":"A new prompt","template":"Hello {{name}}","arguments":[{"name":"name","required":true}],"isActive":true,"tags":["test"],"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	prompt, _, err := client.Prompts.Create(ctx, input, nil)

	if err != nil {
		t.Errorf("Prompts.Create returned error: %v", err)
	}

	if prompt.ID != 456 {
		t.Errorf("Prompts.Create returned prompt ID %d, want %d", prompt.ID, 456)
	}

	if prompt.Name != "new-prompt" {
		t.Errorf("Prompts.Create returned prompt name %q, want %q", prompt.Name, "new-prompt")
	}
}

func TestPromptsService_Create_WithOptions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &PromptCreate{
		Name:     "new-prompt",
		Template: "Hello {{name}}",
	}

	opts := &PromptCreateOptions{
		TeamID:     String("team-123"),
		Visibility: String("private"),
	}

	mux.HandleFunc("/prompts", func(w http.ResponseWriter, r *http.Request) {
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
		fmt.Fprint(w, `{"id":456,"name":"new-prompt","template":"Hello {{name}}","arguments":[],"isActive":true,"teamId":"team-123","visibility":"private","metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	prompt, _, err := client.Prompts.Create(ctx, input, opts)

	if err != nil {
		t.Errorf("Prompts.Create returned error: %v", err)
	}

	if prompt.TeamID == nil || *prompt.TeamID != "team-123" {
		t.Errorf("Prompts.Create returned prompt with teamId %v, want %q", prompt.TeamID, "team-123")
	}
}

func TestPromptsService_Update(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &PromptUpdate{
		Name:        String("updated-prompt"),
		Description: String("An updated prompt"),
	}

	mux.HandleFunc("/prompts/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")

		// Verify the request body is NOT wrapped (PromptUpdate is sent directly)
		var body PromptUpdate
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body as PromptUpdate: %v", err)
		}
		if body.Name == nil || *body.Name != "updated-prompt" {
			t.Error("Expected request body to be PromptUpdate (not wrapped)")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":123,"name":"updated-prompt","description":"An updated prompt","template":"Hello {{name}}","arguments":[],"isActive":true,"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}`)
	})

	ctx := context.Background()
	prompt, _, err := client.Prompts.Update(ctx, 123, input)

	if err != nil {
		t.Errorf("Prompts.Update returned error: %v", err)
	}

	if prompt.Name != "updated-prompt" {
		t.Errorf("Prompts.Update returned prompt name %q, want %q", prompt.Name, "updated-prompt")
	}
}

func TestPromptsService_Delete(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/prompts/123", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	ctx := context.Background()
	_, err := client.Prompts.Delete(ctx, 123)

	if err != nil {
		t.Errorf("Prompts.Delete returned error: %v", err)
	}
}

func TestPromptsService_Toggle(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/prompts/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "true" {
			t.Errorf("Expected activate=true query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","message":"Prompt activated","prompt":{"id":123,"name":"test-prompt","template":"Hello {{name}}","arguments":[],"isActive":true,"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}}`)
	})

	ctx := context.Background()
	prompt, _, err := client.Prompts.Toggle(ctx, 123, true)

	if err != nil {
		t.Errorf("Prompts.Toggle returned error: %v", err)
	}

	if prompt == nil {
		t.Fatal("Prompts.Toggle returned nil prompt")
	}

	if !prompt.IsActive {
		t.Errorf("Prompts.Toggle returned prompt with isActive=%v, want true", prompt.IsActive)
	}
}

func TestPromptsService_Toggle_Deactivate(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/prompts/123/toggle", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		// Verify query parameter
		activate := r.URL.Query().Get("activate")
		if activate != "false" {
			t.Errorf("Expected activate=false query parameter, got %q", activate)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"success","message":"Prompt deactivated","prompt":{"id":123,"name":"test-prompt","template":"Hello {{name}}","arguments":[],"isActive":false,"metrics":{"totalExecutions":0,"successfulExecutions":0,"failedExecutions":0,"failureRate":0}}}`)
	})

	ctx := context.Background()
	prompt, _, err := client.Prompts.Toggle(ctx, 123, false)

	if err != nil {
		t.Errorf("Prompts.Toggle returned error: %v", err)
	}

	if prompt == nil {
		t.Fatal("Prompts.Toggle returned nil prompt")
	}

	if prompt.IsActive {
		t.Errorf("Prompts.Toggle returned prompt with isActive=%v, want false", prompt.IsActive)
	}
}

func TestPromptsService_Create_NilInput(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	ctx := context.Background()
	_, _, err := client.Prompts.Create(ctx, nil, nil)

	if err == nil {
		t.Error("Prompts.Create with nil input should return error")
	}
}
