package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PromptsService handles communication with the prompt-related
// methods of the ContextForge API.
//
// The /rpc endpoint handles MCP JSON-RPC protocol (prompts/get, etc.)
// which is separate from these REST management endpoints.

// List retrieves a paginated list of prompts from the ContextForge API.
func (s *PromptsService) List(ctx context.Context, opts *PromptListOptions) ([]*Prompt, *Response, error) {
	reqOpts := &PromptListOptions{}
	if opts != nil {
		*reqOpts = *opts
	}
	reqOpts.IncludePagination = true

	u := "prompts"
	u, err := addOptions(u, reqOpts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var raw json.RawMessage
	resp, err := s.client.Do(ctx, req, &raw)
	if err != nil {
		return nil, resp, err
	}

	prompts, nextCursor, err := decodeListResponse[Prompt](raw, "prompts")
	if err != nil {
		return nil, resp, err
	}
	if nextCursor != "" {
		resp.NextCursor = nextCursor
	}

	return prompts, resp, nil
}

// Get retrieves a prompt by ID and renders it with the provided arguments.
// This is a hybrid REST endpoint (POST /prompts/{id}) that provides MCP prompts/get functionality via REST.
func (s *PromptsService) Get(ctx context.Context, promptID string, args map[string]string) (*PromptResult, *Response, error) {
	u := fmt.Sprintf("prompts/%s", promptID)

	// Wrap args in request body format
	body := args
	if body == nil {
		body = make(map[string]string)
	}

	req, err := s.client.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, nil, err
	}

	var result *PromptResult
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

// GetNoArgs retrieves a prompt by ID without arguments.
// This is a convenience method that calls GET /prompts/{id}.
func (s *PromptsService) GetNoArgs(ctx context.Context, promptID string) (*PromptResult, *Response, error) {
	u := fmt.Sprintf("prompts/%s", promptID)

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var result *PromptResult
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

// Create creates a new prompt.
// The opts parameter allows setting team_id and visibility at the request wrapper level.
func (s *PromptsService) Create(ctx context.Context, prompt *PromptCreate, opts *PromptCreateOptions) (*Prompt, *Response, error) {
	u := "prompts"

	// Build the request wrapper with prompt and additional fields
	body := map[string]any{
		"prompt": prompt,
	}

	// Add optional fields from opts if provided
	if opts != nil {
		if opts.TeamID != nil {
			body["team_id"] = *opts.TeamID
		}
		if opts.Visibility != nil {
			body["visibility"] = *opts.Visibility
		}
	}

	req, err := s.client.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, nil, err
	}

	var created *Prompt
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// Update updates an existing prompt.
// Note: The API does not wrap the request body for prompt updates.
// Note: promptID changed from int to string in v1.0.0.
func (s *PromptsService) Update(ctx context.Context, promptID string, prompt *PromptUpdate) (*Prompt, *Response, error) {
	u := fmt.Sprintf("prompts/%s", promptID)

	req, err := s.client.NewRequest(http.MethodPut, u, prompt)
	if err != nil {
		return nil, nil, err
	}

	var updated *Prompt
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return updated, resp, nil
}

// Delete deletes a prompt by its ID.
// Note: promptID changed from int to string in v1.0.0.
func (s *PromptsService) Delete(ctx context.Context, promptID string) (*Response, error) {
	u := fmt.Sprintf("prompts/%s", promptID)

	req, err := s.client.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// SetState sets a prompt's active status using the preferred /state endpoint.
func (s *PromptsService) SetState(ctx context.Context, promptID string, activate bool) (*Prompt, *Response, error) {
	return s.setState(ctx, promptID, activate, "state")
}

// Toggle toggles a prompt's active status using the legacy /toggle endpoint.
// Note: promptID changed from int to string in v1.0.0.
func (s *PromptsService) Toggle(ctx context.Context, promptID string, activate bool) (*Prompt, *Response, error) {
	return s.setState(ctx, promptID, activate, "toggle")
}

func (s *PromptsService) setState(ctx context.Context, promptID string, activate bool, endpoint string) (*Prompt, *Response, error) {
	u := fmt.Sprintf("prompts/%s/%s?activate=%t", promptID, endpoint, activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	// The API returns a response with the prompt data nested in the response
	var result map[string]any
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	// Extract the prompt from the response
	var prompt *Prompt
	if promptData, ok := result["prompt"]; ok {
		// Re-marshal and unmarshal to convert to Prompt struct
		promptJSON, _ := json.Marshal(promptData)
		_ = json.Unmarshal(promptJSON, &prompt)
	}

	return prompt, resp, nil
}
