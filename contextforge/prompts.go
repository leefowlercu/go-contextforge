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
// Note: This service intentionally excludes certain MCP client endpoints:
// - POST /prompts/{id} - MCP spec endpoint for getting rendered prompts with arguments
// - GET /prompts/{id} - MCP convenience endpoint for template information
// These endpoints are for MCP client communication, not REST API management.

// List retrieves a paginated list of prompts from the ContextForge API.
func (s *PromptsService) List(ctx context.Context, opts *PromptListOptions) ([]*Prompt, *Response, error) {
	u := "prompts"
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var prompts []*Prompt
	resp, err := s.client.Do(ctx, req, &prompts)
	if err != nil {
		return nil, resp, err
	}

	return prompts, resp, nil
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
func (s *PromptsService) Update(ctx context.Context, promptID int, prompt *PromptUpdate) (*Prompt, *Response, error) {
	u := fmt.Sprintf("prompts/%d", promptID)

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
func (s *PromptsService) Delete(ctx context.Context, promptID int) (*Response, error) {
	u := fmt.Sprintf("prompts/%d", promptID)

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

// Toggle toggles a prompt's active status.
func (s *PromptsService) Toggle(ctx context.Context, promptID int, activate bool) (*Prompt, *Response, error) {
	u := fmt.Sprintf("prompts/%d/toggle?activate=%t", promptID, activate)

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
		json.Unmarshal(promptJSON, &prompt)
	}

	return prompt, resp, nil
}
