package contextforge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ToolsService handles communication with the tool-related
// methods of the ContextForge API.
//
// Note: All /tools/* endpoints are REST API management endpoints.
// There are no MCP protocol endpoints to exclude for this service.

// List retrieves a paginated list of tools from the ContextForge API.
func (s *ToolsService) List(ctx context.Context, opts *ToolListOptions) ([]*Tool, *Response, error) {
	reqOpts := &ToolListOptions{}
	if opts != nil {
		*reqOpts = *opts
	}
	reqOpts.IncludePagination = true

	u := "tools"
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

	tools, nextCursor, err := decodeListResponse[Tool](raw, "tools")
	if err != nil {
		return nil, resp, err
	}
	if nextCursor != "" {
		resp.NextCursor = nextCursor
	}

	return tools, resp, nil
}

// Get retrieves a specific tool by its ID.
func (s *ToolsService) Get(ctx context.Context, toolID string) (*Tool, *Response, error) {
	u := fmt.Sprintf("tools/%s", url.PathEscape(toolID))

	req, err := s.client.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}

	var tool *Tool
	resp, err := s.client.Do(ctx, req, &tool)
	if err != nil {
		return nil, resp, err
	}

	return tool, resp, nil
}

// Create creates a new tool.
// The opts parameter allows setting team_id and visibility at the request wrapper level.
func (s *ToolsService) Create(ctx context.Context, tool *Tool, opts *ToolCreateOptions) (*Tool, *Response, error) {
	u := "tools"

	// Build the request wrapper with tool and additional fields
	body := map[string]any{
		"tool": tool,
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

	var created *Tool
	resp, err := s.client.Do(ctx, req, &created)
	if err != nil {
		return nil, resp, err
	}

	return created, resp, nil
}

// Update updates an existing tool.
func (s *ToolsService) Update(ctx context.Context, toolID string, tool *Tool) (*Tool, *Response, error) {
	u := fmt.Sprintf("tools/%s", url.PathEscape(toolID))

	// Send the tool directly (UPDATE endpoint does not use wrapper, unlike CREATE)
	body := tool

	req, err := s.client.NewRequest(http.MethodPut, u, body)
	if err != nil {
		return nil, nil, err
	}

	var updated *Tool
	resp, err := s.client.Do(ctx, req, &updated)
	if err != nil {
		return nil, resp, err
	}

	return updated, resp, nil
}

// Delete deletes a tool by its ID.
func (s *ToolsService) Delete(ctx context.Context, toolID string) (*Response, error) {
	u := fmt.Sprintf("tools/%s", url.PathEscape(toolID))

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

// SetState sets a tool's active status using the preferred /state endpoint.
func (s *ToolsService) SetState(ctx context.Context, toolID string, activate bool) (*Tool, *Response, error) {
	return s.setState(ctx, toolID, activate, "state")
}

// Toggle toggles a tool's active status using the legacy /toggle endpoint.
func (s *ToolsService) Toggle(ctx context.Context, toolID string, activate bool) (*Tool, *Response, error) {
	return s.setState(ctx, toolID, activate, "toggle")
}

func (s *ToolsService) setState(ctx context.Context, toolID string, activate bool, endpoint string) (*Tool, *Response, error) {
	u := fmt.Sprintf("tools/%s/%s?activate=%t", url.PathEscape(toolID), endpoint, activate)

	req, err := s.client.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, nil, err
	}

	// State endpoints return a response with the tool data nested in the response.
	var result map[string]any
	resp, err := s.client.Do(ctx, req, &result)
	if err != nil {
		return nil, resp, err
	}

	// Extract the tool from the response.
	var tool *Tool
	if toolData, ok := result["tool"]; ok {
		toolJSON, _ := json.Marshal(toolData)
		_ = json.Unmarshal(toolJSON, &tool)
	}

	return tool, resp, nil
}
